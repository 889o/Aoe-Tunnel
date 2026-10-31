package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"aoe-tunnel/internal/panel/auth"
	"aoe-tunnel/internal/panel/model"
	"aoe-tunnel/internal/panel/service"
	"aoe-tunnel/pkg/signal"
)

type Handler struct {
	svc  *service.Service
	opts Options
	deps *AuthDeps
}

// Options 是面板运行期只读配置，/system/settings 直接展示，避免额外建表持久化。
type Options struct {
	Tokens           *auth.Manager
	HTTPAddr         string
	MQTTBroker       string
	Version          string
	JWTSecretSource  string
	AgentDownloadURL string
	AgentUser        string
	AgentPass        string
	MQTTUser         string
	MQTTPass         string
	// PublicURL 是站点公网地址；配置后安装脚本直接用它，缺省按请求头自动拼接。
	PublicURL string
	// AgentBroker 是下发给 Agent 的 MQTT 地址，需远端可达；缺省与面板自身地址一致。
	AgentBroker string
}

func Register(r *gin.Engine, svc *service.Service, opts Options) {
	deps := &AuthDeps{Svc: svc, Tokens: opts.Tokens}
	h := &Handler{svc: svc, opts: opts, deps: deps}

	api := r.Group("/api/v1")
	// 登录与刷新不需要凭证，其余接口一律走 Bearer JWT。
	api.POST("/auth/login", h.login)
	api.POST("/auth/refresh", h.refresh)
	// 开放注册与公开站点信息同样无需登录。
	api.POST("/auth/register", h.register)
	api.GET("/site/config", h.publicSiteConfig)

	// 节点上线是唯一使用节点令牌的接口，且令牌必须与路径节点编码一致。
	api.POST("/nodes/:code/online", deps.NodeToken(), h.nodeOnline)

	// 易支付回调由支付网关直接访问，不接受 JWT；notify 与 return 都按请求 Host 生成。
	api.GET("/payment/epay/notify", h.epayNotify)
	api.POST("/payment/epay/notify", h.epayNotify)
	api.GET("/payment/epay/return", h.epayReturn)

	web := api.Group("", deps.JWT())
	web.POST("/auth/logout", h.logout)
	web.GET("/auth/me", h.me)
	web.GET("/dashboard/summary", RequirePerm(service.PermDashboardView), h.dashboard)

	web.GET("/permissions", RequirePerm(service.PermRoleList), h.listPermissions)

	web.GET("/users", RequirePerm(service.PermUserList), h.listUsers)
	web.POST("/users", RequirePerm(service.PermUserCreate), h.createUser)
	web.PATCH("/users/:id", RequirePerm(service.PermUserUpdate), h.updateUser)
	web.DELETE("/users/:id", RequirePerm(service.PermUserDelete), h.deleteUser)

	web.GET("/roles", RequirePerm(service.PermRoleList), h.listRoles)
	web.POST("/roles", RequirePerm(service.PermRoleCreate), h.createRole)
	web.PATCH("/roles/:id", RequirePerm(service.PermRoleUpdate), h.updateRole)
	web.DELETE("/roles/:id", RequirePerm(service.PermRoleDelete), h.deleteRole)

	web.GET("/sessions", RequirePerm(service.PermSessionList), h.listSessions)
	web.DELETE("/sessions/:sid", RequirePerm(service.PermSessionKick), h.kickSession)

	web.GET("/profile", RequirePerm(service.PermProfileView), h.getProfile)
	web.GET("/profile/invite", RequirePerm(service.PermProfileView), h.inviteInfo)
	web.PATCH("/profile", RequirePerm(service.PermProfileUpdate), h.updateProfile)

	web.GET("/system/settings", RequirePerm(service.PermSystemView), h.systemSettings)
	web.PATCH("/system/settings", RequirePerm(service.PermSystemUpdate), h.updateSystemSettings)
	web.GET("/system/payment-channels", RequirePerm(service.PermSystemView), h.paymentChannels)
	web.PATCH("/system/payment-channels/:code", RequirePerm(service.PermSystemUpdate), h.updatePaymentChannel)

	// 节点与端口接口统一走 JWT，权限码按动作细分。
	web.POST("/nodes", RequirePerm(service.PermNodeCreate), h.createNode)
	web.GET("/nodes", RequirePerm(service.PermNodeRead), h.listNodes)
	web.PATCH("/nodes/:code", RequirePerm(service.PermNodeUpdate), h.updateNode)
	web.PATCH("/nodes/:code/rate-limit", RequirePerm(service.PermNodeUpdate), h.updateNodeRate)
	web.DELETE("/nodes/:code", RequirePerm(service.PermNodeDelete), h.deleteNode)
	web.GET("/nodes/:code/install", RequirePerm(service.PermNodeUpdate), h.nodeInstall)
	web.POST("/nodes/:code/rotate-token", RequirePerm(service.PermNodeUpdate), h.rotateNodeToken)
	web.POST("/nodes/:code/report", RequirePerm(service.PermNodeReport), h.nodeReport)
	web.GET("/nodes/:code/stat-history", RequirePerm(service.PermNodeRead), h.nodeStatHistory)
	web.GET("/monitor/overview", RequirePerm(service.PermNodeRead), h.monitorOverview)
	web.GET("/packages", RequirePerm(service.PermPackageList), h.listPackages)
	web.POST("/packages", RequirePerm(service.PermPackageCreate), h.createPackage)
	web.PATCH("/packages/:id", RequirePerm(service.PermPackageUpdate), h.updatePackage)
	web.DELETE("/packages/:id", RequirePerm(service.PermPackageDelete), h.deletePackage)
	web.GET("/wallet", RequirePerm(service.PermWalletView), h.wallet)
	web.POST("/wallet/purchase", RequirePerm(service.PermWalletView), h.purchasePackage)
	web.GET("/wallet/payment-methods", RequirePerm(service.PermWalletView), h.paymentMethods)
	web.GET("/wallet/recharge-options", RequirePerm(service.PermWalletView), h.rechargeOptions)
	web.POST("/wallet/payment-orders", RequirePerm(service.PermWalletView), h.createPaymentOrder)
	web.POST("/wallet/recharge-orders", RequirePerm(service.PermWalletView), h.createRechargeOrder)
	web.GET("/wallet/payment-orders/:no", RequirePerm(service.PermWalletView), h.paymentOrder)
	web.POST("/port-rules", RequirePerm(service.PermPortCreate), h.createPortRule)
	web.GET("/port-rules", RequirePerm(service.PermPortList), h.listPortRules)
	web.GET("/port-rules/:id", RequirePerm(service.PermPortRead), h.getPortRule)
	web.PATCH("/port-rules/:id", RequirePerm(service.PermPortUpdate), h.updatePortRule)
	web.DELETE("/port-rules/:id", RequirePerm(service.PermPortDelete), h.deletePortRule)
	web.GET("/port-rules/:id/stats", RequirePerm(service.PermPortStats), h.portRuleStats)
}

// ruleAllowed 实施行级隔离：无 port:manage-all 权限时只能操作本人创建的规则。
func (h *Handler) ruleAllowed(c *gin.Context, rule *model.PortRule) bool {
	p := Identity(c)
	if p == nil {
		return false
	}
	if p.Has(service.PermPortManageAll) {
		return true
	}
	return rule.OwnerUserID != nil && *rule.OwnerUserID == p.User.ID
}

func (h *Handler) createNode(c *gin.Context) {
	var in service.CreateNodeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.svc.CreateNode(c.Request.Context(), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, node)
}

func (h *Handler) listNodes(c *gin.Context) {
	nodes, err := h.svc.ListNodes(c.Request.Context(), Identity(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, nodes)
}

// publicBaseURL 决定站点公网地址：系统设置优先，其次环境变量，再取反向代理透传的头，最后回退请求 Host。
func (h *Handler) publicBaseURL(c *gin.Context) string {
	if cfg, err := h.svc.SiteConfig(c.Request.Context()); err == nil && cfg.PublicURL != "" {
		return cfg.PublicURL
	}
	if h.opts.PublicURL != "" {
		return h.opts.PublicURL
	}
	scheme := "http"
	if c.Request.TLS != nil || firstHeader(c, "X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := firstHeader(c, "X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		host = h.opts.HTTPAddr
	}
	return scheme + "://" + host
}

// firstHeader 取多级代理透传头的第一个值，避免把整条链路拼进地址。
func firstHeader(c *gin.Context, key string) string {
	v := c.GetHeader(key)
	if i := strings.Index(v, ","); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// nodeInstall 返回一键安装脚本；面板地址、下载目录与 MQTT 地址优先取系统设置。
func (h *Handler) nodeInstall(c *gin.Context) {
	downloadURL := h.opts.AgentDownloadURL
	broker := h.opts.AgentBroker
	if cfg, err := h.svc.SiteConfig(c.Request.Context()); err == nil {
		if cfg.AgentDownloadURL != "" {
			downloadURL = cfg.AgentDownloadURL
		}
		if cfg.AgentBroker != "" {
			broker = cfg.AgentBroker
		}
	}
	publicURL := h.publicBaseURL(c)
	// 默认全部由主控面板 /downloads 提供；只有显式配置了外部地址才改用外部 install.sh。
	binaryBase := strings.TrimSuffix(publicURL, "/") + "/downloads"
	if downloadURL == "" {
		downloadURL = binaryBase
	}
	bundle, err := h.svc.NodeInstall(c.Request.Context(), c.Param("code"),
		publicURL, downloadURL, binaryBase, broker,
		h.opts.MQTTUser, h.opts.MQTTPass, h.opts.AgentUser, h.opts.AgentPass)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, bundle)
}

func (h *Handler) rotateNodeToken(c *gin.Context) {
	node, err := h.svc.RotateNodeToken(c.Request.Context(), c.Param("code"))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{
		"code":  node.Code,
		"epoch": node.NodeTokenEpoch,
		"token": h.svc.NodeToken(node.Code, node.NodeTokenEpoch),
	})
}

// nodeOnline 由节点令牌保护，登记 Agent 外网 IP 与数据面地址，只刷新在线状态，不授予业务权限。
func (h *Handler) nodeOnline(c *gin.Context) {
	var in service.NodeOnlineInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&in); err != nil {
			Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err := h.svc.SetNodeOnline(c.Request.Context(), c.Param("code"), c.ClientIP(), in); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"code": c.Param("code"), "online": true})
}

// nodeReport 接收 Agent 经 JWT 上报的信令，节点编码以路径为准，忽略请求体中的 node_code。
func (h *Handler) nodeReport(c *gin.Context) {
	var env signal.Envelope
	if err := c.ShouldBindJSON(&env); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	env.NodeCode = c.Param("code")
	h.svc.HandleReport(c.Request.Context(), env.NodeCode, env)
	OK(c, gin.H{"accepted": true})
}

func (h *Handler) nodeStatHistory(c *gin.Context) {
	points, err := h.svc.NodeStatHistory(c.Request.Context(), c.Param("code"), Identity(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, points)
}

func (h *Handler) monitorOverview(c *gin.Context) {
	nodes, err := h.svc.MonitorOverview(c.Request.Context(), Identity(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, nodes)
}

func (h *Handler) updateNode(c *gin.Context) {
	var in service.UpdateNodeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.svc.UpdateNode(c.Request.Context(), c.Param("code"), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, node)
}

// updateNodeRate 配置入口节点级带宽上限，单位 MB/s；0 表示不限速。
func (h *Handler) updateNodeRate(c *gin.Context) {
	var in service.UpdateNodeRateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.svc.UpdateNodeRate(c.Request.Context(), c.Param("code"), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, node)
}

// deleteNode 仅在节点未被套餐或端口规则引用时允许删除。
func (h *Handler) deleteNode(c *gin.Context) {
	if err := h.svc.DeleteNode(c.Request.Context(), c.Param("code")); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"deleted": c.Param("code")})
}

func (h *Handler) createPortRule(c *gin.Context) {
	var in service.CreateRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	p := Identity(c)
	rule, err := h.svc.CreatePortRule(c.Request.Context(), in, p)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, rule)
}

func (h *Handler) listPortRules(c *gin.Context) {
	var owner *uint
	if p := Identity(c); p != nil && !p.Has(service.PermPortManageAll) {
		owner = &p.User.ID
	}
	filter := service.RuleFilter{
		Node:      strings.TrimSpace(c.Query("node")),
		Direction: strings.TrimSpace(c.Query("direction")),
	}
	if v := c.Query("package_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			Fail(c, http.StatusBadRequest, "invalid package_id")
			return
		}
		filter.PackageID = uint(id)
	}
	rules, err := h.svc.ListPortRules(c.Request.Context(), filter, owner)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, rules)
}

func (h *Handler) listPackages(c *gin.Context) {
	pkgs, err := h.svc.ListPackages(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, pkgs)
}

func (h *Handler) createPackage(c *gin.Context) {
	var in service.CreatePackageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	pkg, err := h.svc.CreatePackage(c.Request.Context(), in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, pkg)
}

func (h *Handler) updatePackage(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var in service.UpdatePackageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	pkg, err := h.svc.UpdatePackage(c.Request.Context(), id, in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, pkg)
}

func (h *Handler) deletePackage(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.DeletePackage(c.Request.Context(), id); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"deleted": id})
}

func (h *Handler) getPortRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.GetPortRule(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	if !h.ruleAllowed(c, rule) {
		Fail(c, http.StatusForbidden, "无权访问该端口规则")
		return
	}
	OK(c, rule)
}

func (h *Handler) updatePortRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	existing, err := h.svc.GetPortRule(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	if !h.ruleAllowed(c, existing) {
		Fail(c, http.StatusForbidden, "无权修改该端口规则")
		return
	}
	var in service.UpdateRuleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.UpdatePortRule(c.Request.Context(), id, in)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, rule)
}

func (h *Handler) deletePortRule(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	existing, err := h.svc.GetPortRule(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	if !h.ruleAllowed(c, existing) {
		Fail(c, http.StatusForbidden, "无权删除该端口规则")
		return
	}
	if err := h.svc.DeletePortRule(c.Request.Context(), id); err != nil {
		handleErr(c, err)
		return
	}
	OK(c, gin.H{"deleted": id})
}

func (h *Handler) portRuleStats(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := h.svc.GetPortRule(c.Request.Context(), id)
	if err != nil {
		handleErr(c, err)
		return
	}
	if !h.ruleAllowed(c, rule) {
		Fail(c, http.StatusForbidden, "无权查看该端口规则流量")
		return
	}

	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			Fail(c, http.StatusBadRequest, "invalid from, want RFC3339")
			return
		}
		from = t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			Fail(c, http.StatusBadRequest, "invalid to, want RFC3339")
			return
		}
		to = t
	}

	stats, err := h.svc.PortRuleStats(c.Request.Context(), id, from, to)
	if err != nil {
		handleErr(c, err)
		return
	}
	OK(c, stats)
}

func parseID(c *gin.Context) (uint, error) {
	v := c.Param("id")
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id), nil
}
