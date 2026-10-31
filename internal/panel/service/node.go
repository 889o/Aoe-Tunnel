package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"go.uber.org/zap"

	"aoe-tunnel/internal/panel/model"
	"aoe-tunnel/pkg/signal"
)

// NodeTokenPrefix 节点令牌统一前缀，便于日志与密钥扫描识别。
const NodeTokenPrefix = "aoe_nt_"

func nodeTokenPlain(secret []byte, code string, epoch int) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(fmt.Sprintf("%s:%d", code, epoch)))
	sig := hex.EncodeToString(mac.Sum(nil))[:32]
	return NodeTokenPrefix + code + "." + sig
}

func hashNodeToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// NodeToken 按节点编码与 epoch 重算明文令牌；DB 只存哈希，面板可随时渲染安装脚本。
func (s *Service) NodeToken(code string, epoch int) string {
	return nodeTokenPlain(s.nodeSecret, strings.TrimSpace(code), epoch)
}

func (s *Service) nodeTokenHash(code string, epoch int) string {
	return hashNodeToken(s.NodeToken(code, epoch))
}

// AuthenticateNodeToken 校验 X-Node-Token；令牌与节点绑定，只用于节点上线接口。
func (s *Service) AuthenticateNodeToken(ctx context.Context, plain string) (*Principal, error) {
	plain = strings.TrimSpace(plain)
	if !strings.HasPrefix(plain, NodeTokenPrefix) {
		return nil, ErrUnauthorized
	}
	body := strings.TrimPrefix(plain, NodeTokenPrefix)
	idx := strings.LastIndex(body, ".")
	if idx <= 0 || idx == len(body)-1 {
		return nil, ErrUnauthorized
	}
	code := body[:idx]

	var node model.Node
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	// 直接对 HMAC 派生令牌做常量时间比对：与 DB 中的哈希无关，避免密钥切换后历史哈希漂移导致误拒。
	epoch := node.NodeTokenEpoch
	if epoch <= 0 {
		epoch = 1
	}
	expected := nodeTokenPlain(s.nodeSecret, node.Code, epoch)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(plain)) != 1 {
		return nil, ErrUnauthorized
	}

	return &Principal{
		Via:      "node-token",
		NodeCode: node.Code,
		NodeType: node.Type,
	}, nil
}

// RotateNodeToken 递增 epoch 使旧令牌立即失效，并回收节点令牌哈希。
func (s *Service) RotateNodeToken(ctx context.Context, code string) (*model.Node, error) {
	node, err := s.getNode(ctx, code)
	if err != nil {
		return nil, err
	}
	node.NodeTokenEpoch++
	node.NodeTokenHash = s.nodeTokenHash(node.Code, node.NodeTokenEpoch)
	if err := s.db.WithContext(ctx).Save(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

func validateNodeShape(typ int) error {
	if typ != model.NodeTypeIngress && typ != model.NodeTypeEgress {
		return fmt.Errorf("type must be 1 (ingress) or 2 (egress)")
	}
	return nil
}

// InstallBundle 是节点详情弹窗展示的一键安装内容。
type InstallBundle struct {
	Code     string `json:"code"`
	Type     int    `json:"type"`
	Role     string `json:"role"`
	Filename string `json:"filename"`
	Command  string `json:"command"`
	Token    string `json:"token"`
	Script   string `json:"script"`
	// InstallURL 与 OnlineCommand 指向云存储上的通用脚本，适合无需下载文件的在线安装。
	InstallURL    string `json:"install_url"`
	OnlineCommand string `json:"online_command"`
}

// NodeInstall 生成一键安装脚本；panelURL 由请求 Host 推导，Agent 登录账号由面板配置注入。
func (s *Service) NodeInstall(ctx context.Context, code, panelURL, scriptBase, binaryBase, mqttBroker, mqttUser, mqttPass, agentUser, agentPass string) (*InstallBundle, error) {
	node, err := s.getNode(ctx, code)
	if err != nil {
		return nil, err
	}
	role := "ingress"
	if node.Type == model.NodeTypeEgress {
		role = "egress"
	}
	token := s.NodeToken(node.Code, node.NodeTokenEpoch)
	filename := fmt.Sprintf("aoe-agent-%s-%s.sh", role, node.Code)
	base := strings.TrimSuffix(strings.TrimSpace(scriptBase), "/")
	installURL := ""
	if base != "" {
		installURL = base + "/install.sh"
	}
	online := ""
	if installURL != "" {
		// 在线安装必须把远端可达的控制面地址与账号一并带上，否则 Agent 会连到服务器本机。
		args := []string{
			"--code", shellQuote(node.Code),
			"--token", shellQuote(token),
			"--panel", shellQuote(panelURL),
			"--broker", shellQuote(mqttBroker),
			// curl|bash 拿不到脚本自身路径，二进制与证书统一从主控面板下载。
			"--base", shellQuote(strings.TrimSuffix(binaryBase, "/")),
		}
		if mqttUser != "" {
			args = append(args, "--mqtt-user", shellQuote(mqttUser), "--mqtt-pass", shellQuote(mqttPass))
		}
		args = append(args, "--panel-user", shellQuote(agentUser), "--panel-pass", shellQuote(agentPass))
		online = fmt.Sprintf("curl -fsSL %s | sudo bash -s -- %s",
			shellQuote(installURL), strings.Join(args, " "))
	}
	return &InstallBundle{
		Code:          node.Code,
		Type:          node.Type,
		Role:          role,
		Filename:      filename,
		Command:       fmt.Sprintf("sudo bash %s", filename),
		Token:         token,
		Script:        renderInstallScript(node, token, panelURL, binaryBase, mqttBroker, mqttUser, mqttPass, agentUser, agentPass, role),
		InstallURL:    installURL,
		OnlineCommand: online,
	}, nil
}

// renderInstallScript 在通用安装脚本头部注入节点专属变量，脚本体内不含任何密钥字面量。
func renderInstallScript(node *model.Node, token, panelURL, downloadURL, mqttBroker, mqttUser, mqttPass, agentUser, agentPass, role string) string {
	exports := [][2]string{
		{"AOE_ROLE", role},
		{"AOE_NODE_CODE", node.Code},
		{"AOE_NODE_TOKEN", token},
		{"AOE_PANEL_URL", panelURL},
		{"AOE_PANEL_USER", agentUser},
		{"AOE_PANEL_PASS", agentPass},
		{"AOE_MQTT_BROKER", mqttBroker},
		{"AOE_MQTT_USER", mqttUser},
		{"AOE_MQTT_PASS", mqttPass},
		{"AOE_DOWNLOAD_BASE", downloadURL},
	}

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# 节点角色: " + role + "；本脚本由面板按节点编码与令牌动态生成，令牌仅用于上线校验。\n")
	for _, kv := range exports {
		fmt.Fprintf(&b, "export %s=%s\n", kv[0], shellQuote(kv[1]))
	}
	b.WriteString(strings.TrimPrefix(installAgentScript, "#!/usr/bin/env bash\n"))
	return b.String()
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

// NodeOnlineInput 是 Agent 上线时自报的数据面地址；IP 由 Agent 通过 curl 查询外网地址上报。
// InstanceID 是部署实例标识：入口只保留最后上线的实例，出口允许同一节点多实例并行。
type NodeOnlineInput struct {
	Version    string `json:"version,omitempty"`
	IP         string `json:"ip,omitempty"`
	QUICAddr   string `json:"quic_addr,omitempty"`
	TLSAddr    string `json:"tls_addr,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
}

// instanceStaleAfter 是实例心跳过期阈值：Agent 每 15s 心跳，允许连续三次丢失。
const instanceStaleAfter = 90 * time.Second

// SetNodeOnline 记录 Agent 外网 IP 与数据面地址并刷新在线状态；地址变化时重推依赖它的入口规则。
func (s *Service) SetNodeOnline(ctx context.Context, code, ip string, in NodeOnlineInput) error {
	code = strings.TrimSpace(code)
	var node model.Node
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	// 地址变化需要重推依赖该节点的入口规则，先记录变更前的可拨号地址与在线状态。
	oldQUIC, oldTLS := resolveNodeAddrs(node)
	oldStatus := node.Status

	updates := map[string]any{"status": "online", "last_seen_at": time.Now()}
	// 外网 IP 优先采用 Agent 通过 curl 上报的值，多机部署时请求来源 IP 可能是内网或代理地址。
	if v := normalizeNodeIP(in.IP); v != "" {
		node.IP = v
		updates["ip"] = v
	} else if v := normalizeNodeIP(ip); v != "" {
		node.IP = v
		updates["ip"] = v
	}
	if v := strings.TrimSpace(in.Version); v != "" {
		updates["version"] = v
	}
	if raw := strings.TrimSpace(in.QUICAddr); raw != "" {
		if addr := resolveNodeAddr(node, raw); addr != "" {
			node.QUICAddr = addr
			updates["quic_addr"] = addr
		}
	}
	if raw := strings.TrimSpace(in.TLSAddr); raw != "" {
		if addr := resolveNodeAddr(node, raw); addr != "" {
			node.TLSAddr = addr
			updates["tls_addr"] = addr
		}
	}

	instanceID := strings.TrimSpace(in.InstanceID)
	if instanceID == "" {
		// 未上报实例 ID 的旧版 Agent 维持节点级单一地址行为。
		res := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", code).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if newQUIC, newTLS := resolveNodeAddrs(node); newQUIC != oldQUIC || newTLS != oldTLS || oldStatus != "online" {
			s.redispatchEgressDependents(ctx, code)
		}
		return nil
	}

	var prev model.NodeInstance
	prevErr := s.db.WithContext(ctx).
		Where("node_code = ? AND instance_id = ?", code, instanceID).First(&prev).Error
	if prevErr != nil && !errors.Is(prevErr, gorm.ErrRecordNotFound) {
		return prevErr
	}
	isNewInstance := errors.Is(prevErr, gorm.ErrRecordNotFound)
	instanceChanged := isNewInstance || !prev.Online ||
		prev.QUICAddr != node.QUICAddr || prev.TLSAddr != node.TLSAddr

	// 入口节点独占：首次上线的实例或全新部署的实例接管；被接管实例的晚到心跳直接拒绝并踢下线。
	if node.Type == model.NodeTypeIngress && node.ActiveInstance != instanceID {
		if !isNewInstance && node.ActiveInstance != "" {
			s.publishNodeKick(ctx, code, instanceID)
			return fmt.Errorf("%w: node %s has been taken over by another instance", ErrConflict, code)
		}
		if err := s.takeOverIngress(ctx, code, instanceID); err != nil {
			return err
		}
		updates["active_instance"] = instanceID
	}

	res := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", code).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}

	now := time.Now()
	instance := model.NodeInstance{
		NodeCode:   code,
		InstanceID: instanceID,
		IP:         node.IP,
		QUICAddr:   node.QUICAddr,
		TLSAddr:    node.TLSAddr,
		Version:    strings.TrimSpace(in.Version),
		Online:     true,
		LastSeenAt: now,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_code"}, {Name: "instance_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"ip", "quic_addr", "tls_addr", "version", "online", "last_seen_at", "updated_at"}),
	}).Create(&instance).Error; err != nil {
		return err
	}

	if node.Type == model.NodeTypeEgress && (instanceChanged || oldStatus != "online") {
		s.redispatchEgressDependents(ctx, code)
	}
	return nil
}

// takeOverIngress 让新实例独占入口节点：其余实例标记离线并下发踢下线指令。
func (s *Service) takeOverIngress(ctx context.Context, code, activeInstance string) error {
	var others []model.NodeInstance
	if err := s.db.WithContext(ctx).Where("node_code = ? AND instance_id <> ?", code, activeInstance).
		Find(&others).Error; err != nil {
		return err
	}
	for _, inst := range others {
		if err := s.db.WithContext(ctx).Model(&model.NodeInstance{}).
			Where("id = ?", inst.ID).Update("online", false).Error; err != nil {
			return err
		}
		s.publishNodeKick(ctx, code, inst.InstanceID)
	}
	return nil
}

// publishNodeKick 向节点命令主题下发踢下线指令，目标 Agent 收到后主动退出。
func (s *Service) publishNodeKick(ctx context.Context, code, instanceID string) {
	if instanceID == "" {
		return
	}
	env, err := signal.NewEnvelope(signal.TypeNodeKick, code, signal.NodeKick{InstanceID: instanceID})
	if err != nil {
		s.log.Warn("node kick: build envelope failed", zap.String("node", code), zap.Error(err))
		return
	}
	raw, err := env.Marshal()
	if err != nil {
		s.log.Warn("node kick: marshal failed", zap.String("node", code), zap.Error(err))
		return
	}
	if err := s.mqtt.Publish(signal.CmdTopic(code), raw); err != nil {
		s.log.Warn("node kick: publish failed", zap.String("node", code), zap.String("instance", instanceID), zap.Error(err))
	}
}

// recomputeNodeStatus 按在线实例数刷新节点状态；返回状态是否发生变化。
func (s *Service) recomputeNodeStatus(ctx context.Context, node *model.Node) bool {
	var onlineCount int64
	if err := s.db.WithContext(ctx).Model(&model.NodeInstance{}).
		Where("node_code = ? AND online = ?", node.Code, true).Count(&onlineCount).Error; err != nil {
		s.log.Warn("recompute node status: count failed", zap.String("node", node.Code), zap.Error(err))
		return false
	}
	status := "offline"
	if onlineCount > 0 {
		status = "online"
	}
	if node.Status == status {
		return false
	}
	if err := s.db.WithContext(ctx).Model(&model.Node{}).
		Where("code = ?", node.Code).Update("status", status).Error; err != nil {
		s.log.Warn("recompute node status: update failed", zap.String("node", node.Code), zap.Error(err))
		return false
	}
	node.Status = status
	return true
}

// RunInstanceSweeper 定期把心跳超时的实例置为离线，并按实例状态刷新节点与入口规则。
func (s *Service) RunInstanceSweeper(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepStaleInstances(ctx)
		}
	}
}

func (s *Service) sweepStaleInstances(ctx context.Context) {
	cutoff := time.Now().Add(-instanceStaleAfter)
	var stale []model.NodeInstance
	if err := s.db.WithContext(ctx).
		Where("online = ? AND last_seen_at < ?", true, cutoff).Find(&stale).Error; err != nil {
		s.log.Warn("sweep instances: query failed", zap.Error(err))
		return
	}
	if len(stale) == 0 {
		return
	}
	if err := s.db.WithContext(ctx).Model(&model.NodeInstance{}).
		Where("online = ? AND last_seen_at < ?", true, cutoff).Update("online", false).Error; err != nil {
		s.log.Warn("sweep instances: update failed", zap.Error(err))
		return
	}
	affected := make(map[string]struct{}, len(stale))
	for _, inst := range stale {
		affected[inst.NodeCode] = struct{}{}
		s.log.Info("node instance offline", zap.String("node", inst.NodeCode), zap.String("instance", inst.InstanceID))
	}
	for code := range affected {
		node, err := s.getNode(ctx, code)
		if err != nil {
			continue
		}
		s.recomputeNodeStatus(ctx, node)
		// 出口少了一台实例，入口侧地址列表必须立刻收紧；与节点整体是否仍在线无关。
		if node.Type == model.NodeTypeEgress {
			s.redispatchEgressDependents(ctx, code)
		}
	}
}

// UpdateNodeRate 配置节点计费比；入口节点额外支持带宽上限（MB/s），出口节点不支持限速。
func (s *Service) UpdateNodeRate(ctx context.Context, code string, in UpdateNodeRateInput) (*model.Node, error) {
	code = strings.TrimSpace(code)
	var node model.Node
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if node.Type != model.NodeTypeIngress && node.Type != model.NodeTypeEgress {
		return nil, fmt.Errorf("%w: unsupported node type", ErrConflict)
	}
	if node.Type == model.NodeTypeEgress && (in.RateUpMBps != nil || in.RateDownMBps != nil) {
		return nil, fmt.Errorf("%w: egress nodes do not support rate limit", ErrConflict)
	}

	updates := make(map[string]any, 4)
	if in.RateUpMBps != nil {
		if *in.RateUpMBps < 0 {
			return nil, fmt.Errorf("rate_up_mbps must not be negative")
		}
		updates["rate_up_mbps"] = *in.RateUpMBps
		node.RateUpMBps = *in.RateUpMBps
	}
	if in.RateDownMBps != nil {
		if *in.RateDownMBps < 0 {
			return nil, fmt.Errorf("rate_down_mbps must not be negative")
		}
		updates["rate_down_mbps"] = *in.RateDownMBps
		node.RateDownMBps = *in.RateDownMBps
	}
	if in.RatioUp != nil {
		if *in.RatioUp <= 0 {
			return nil, fmt.Errorf("ratio_up must be positive")
		}
		updates["ratio_up"] = *in.RatioUp
		node.RatioUp = *in.RatioUp
	}
	if in.RatioDown != nil {
		if *in.RatioDown <= 0 {
			return nil, fmt.Errorf("ratio_down must be positive")
		}
		updates["ratio_down"] = *in.RatioDown
		node.RatioDown = *in.RatioDown
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", code).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	// 入口限速随入口规则下发；计费比变化需重推入口或出口的合成规则。
	if node.Type == model.NodeTypeIngress {
		s.redispatchNodeEntries(ctx, code)
	} else {
		s.redispatchEgressDependents(ctx, code)
	}
	return &node, nil
}

// DeleteNode 删除节点：先通知 Agent 撤销并以入口或被引用为出口的端口规则，
// 节点仍绑定在套餐时拒绝，避免套餐出现悬空引用。
func (s *Service) DeleteNode(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if _, err := s.getNode(ctx, code); err != nil {
		return err
	}
	var rules []model.PortRule
	if err := s.db.WithContext(ctx).Where("node = ? OR egress_node = ?", code, code).Find(&rules).Error; err != nil {
		return err
	}
	var pkgs []model.Package
	if err := s.db.WithContext(ctx).Find(&pkgs).Error; err != nil {
		return err
	}
	for i := range pkgs {
		pkg := &pkgs[i]
		ingressChanged := removeCode(&pkg.IngressNodes, code)
		egressChanged := removeCode(&pkg.EgressNodes, code)
		if !ingressChanged && !egressChanged {
			continue
		}
		if err := s.db.WithContext(ctx).Save(pkg).Error; err != nil {
			return err
		}
		s.log.Info("node unbound from package",
			zap.String("node", code), zap.String("package", pkg.Name))
	}
	for _, rule := range rules {
		s.dispatchRemove(ctx, rule)
	}
	if len(rules) > 0 {
		ids := make([]uint, 0, len(rules))
		for _, rule := range rules {
			ids = append(ids, rule.ID)
		}
		if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.PortRule{}).Error; err != nil {
			return err
		}
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_code = ?", code).Delete(&model.NodeInstance{}).Error; err != nil {
			return err
		}
		return tx.Where("code = ?", code).Delete(&model.Node{}).Error
	})
}

// removeCode 从节点编码列表中删除指定节点，返回是否发生变更。
func removeCode(codes *[]string, code string) bool {
	list := *codes
	for i, v := range list {
		if v == code {
			*codes = append(append([]string(nil), list[:i]...), list[i+1:]...)
			return true
		}
	}
	return false
}

// normalizeNodeIP 只接受合法 IP，防止 Agent 上报任意字符串污染节点地址。
func normalizeNodeIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || net.ParseIP(raw) == nil {
		return ""
	}
	return raw
}

// resolveNodeAddrs 把节点登记的数据面地址解析为可拨号地址。
func resolveNodeAddrs(node model.Node) (string, string) {
	return resolveNodeAddr(node, node.QUICAddr), resolveNodeAddr(node, node.TLSAddr)
}

// resolveNodeAddr 用节点外网 IP 替换空 host / 0.0.0.0 / :: 的监听地址，只登记端口时也按此拼装。
func resolveNodeAddr(node model.Node, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		p, convErr := strconv.Atoi(strings.TrimPrefix(raw, ":"))
		if convErr != nil || p < 1 || p > 65535 {
			return raw
		}
		host, port = "", strconv.Itoa(p)
	}
	if port == "" || port == "0" {
		return ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = node.IP
	}
	if host == "" {
		return ""
	}
	return net.JoinHostPort(host, port)
}
