package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"aoe-tunnel/internal/panel/auth"
	"aoe-tunnel/internal/panel/model"
	"aoe-tunnel/pkg/mqttcli"
	"aoe-tunnel/pkg/signal"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
)

type Service struct {
	db         *gorm.DB
	mqtt       *mqttcli.Client
	tokens     *auth.Manager
	sessions   *sessionGuard
	monitor    *monitorCache
	log        *zap.Logger
	nodeSecret []byte
}

func New(db *gorm.DB, mqttCli *mqttcli.Client, tokens *auth.Manager, log *zap.Logger) *Service {
	return &Service{
		db:       db,
		mqtt:     mqttCli,
		tokens:   tokens,
		sessions: newSessionGuard(),
		monitor:  newMonitorCache(),
		log:      log,
	}
}

// SetNodeSecret 注入节点令牌派生密钥，必须在 Bootstrap 之前调用。
func (s *Service) SetNodeSecret(secret []byte) {
	s.nodeSecret = secret
}

// CreateNodeInput 名称与角色必填；对接地址可选，节点编码由面板生成。
type CreateNodeInput struct {
	Name       string `json:"name" binding:"required,max=128"`
	Type       int    `json:"type"`
	PublicAddr string `json:"public_addr,omitempty" binding:"max=255"`
}

// UpdateNodeInput 仅允许修改节点名称，编码、类型与地址均不可变更。
type UpdateNodeInput struct {
	Name *string `json:"name,omitempty" binding:"omitempty,max=128"`
}

// UpdateNodeRateInput 是入口节点级带宽上限（单位 MB/s）与计费比，0 或留空表示不限速。
type UpdateNodeRateInput struct {
	RateUpMBps   *float64 `json:"rate_up_mbps,omitempty"`
	RateDownMBps *float64 `json:"rate_down_mbps,omitempty"`
	RatioUp      *float64 `json:"ratio_up,omitempty"`
	RatioDown    *float64 `json:"ratio_down,omitempty"`
}

// CreateRuleInput 创建入口规则：入口与出口都必须属于当前登录用户绑定的套餐；
// 监听端口留空则自动分配；目标地址必填，由所选出口节点 Dial。
type CreateRuleInput struct {
	Name       string `json:"name" binding:"required,max=128"`
	Node       string `json:"node" binding:"required,max=64"`
	EgressNode string `json:"egress_node" binding:"required,max=64"`
	Protocol   string `json:"protocol,omitempty"`
	Transport  string `json:"transport,omitempty"`
	ListenPort int    `json:"listen_port"`
	TargetAddr string `json:"target_addr" binding:"required,max=255"`
}

type UpdateRuleInput struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,max=128"`
	TargetAddr *string `json:"target_addr,omitempty" binding:"omitempty,max=255"`
	EgressNode *string `json:"egress_node,omitempty" binding:"omitempty,max=64"`
	Transport  *string `json:"transport,omitempty"`
	ListenPort *int    `json:"listen_port,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

// CreateNode 自动生成节点编码，名称与角色是唯一输入；数据面地址等待 Agent 上线时回填。
func (s *Service) CreateNode(ctx context.Context, in CreateNodeInput) (*model.Node, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("node name required")
	}
	if err := validateNodeShape(in.Type); err != nil {
		return nil, err
	}

	var code string
	for attempt := 0; attempt < 5; attempt++ {
		candidate := "n-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", candidate).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			code = candidate
			break
		}
	}
	if code == "" {
		return nil, fmt.Errorf("%w: failed to allocate node code", ErrConflict)
	}

	node := &model.Node{
		Code:           code,
		Name:           name,
		Type:           in.Type,
		PublicAddr:     strings.TrimSpace(in.PublicAddr),
		NodeTokenEpoch: 1,
		Status:         "offline",
	}
	node.NodeTokenHash = s.nodeTokenHash(node.Code, node.NodeTokenEpoch)
	if err := s.db.WithContext(ctx).Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// ListNodes 返回节点目录，Web 与 Agent 共用；行级隔离由端口规则归属决定。
func (s *Service) ListNodes(ctx context.Context, actor *Principal) ([]model.Node, error) {
	var nodes []model.Node
	if err := s.db.WithContext(ctx).Order("code asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpdateNode 只允许修改名称；类型、编码与数据面地址由创建与 Agent 上线决定。
func (s *Service) UpdateNode(ctx context.Context, code string, in UpdateNodeInput) (*model.Node, error) {
	code = strings.TrimSpace(code)
	var node model.Node
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if in.Name == nil {
		return &node, nil
	}
	name := strings.TrimSpace(*in.Name)
	if name == "" {
		return nil, fmt.Errorf("node name required")
	}
	if err := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", code).
		Update("name", name).Error; err != nil {
		return nil, err
	}
	node.Name = name
	return &node, nil
}

// redispatchNodeEntries 入口节点限速变化后升版重推该节点上的入口规则，让 Agent 刷新节点级限速。
func (s *Service) redispatchNodeEntries(ctx context.Context, nodeCode string) {
	var entries []model.PortRule
	if err := s.db.WithContext(ctx).
		Where("node = ? AND direction = ?", nodeCode, model.RuleDirectionEntry).
		Order("id asc").Find(&entries).Error; err != nil {
		s.log.Error("redispatch node entries failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	for _, rule := range entries {
		rule.Revision++
		if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
			s.log.Warn("redispatch node entries: bump revision failed",
				zap.Uint("rule_id", rule.ID), zap.Error(err))
			continue
		}
		s.dispatch(ctx, rule, signal.TypePortUpdate)
	}
}

// redispatchEgressDependents 出口节点地址变化后，重推所有把它作为出口的套餐入口规则。
func (s *Service) redispatchEgressDependents(ctx context.Context, nodeCode string) {
	var pkgs []model.Package
	if err := s.db.WithContext(ctx).Find(&pkgs).Error; err != nil {
		s.log.Error("redispatch egress dependents: list packages failed",
			zap.String("node", nodeCode), zap.Error(err))
		return
	}
	for _, pkg := range pkgs {
		if containsCode(pkg.EgressNodes, nodeCode) {
			s.redispatchPackageEntries(ctx, pkg.ID)
		}
	}
}

func (s *Service) getNode(ctx context.Context, code string) (*model.Node, error) {
	var node model.Node
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &node, nil
}

const (
	allocatedPortMin = 10000
	allocatedPortMax = 60000
)

// CreatePortRule 只创建入口规则：套餐由当前登录用户绑定关系决定，入口节点必须属于该套餐；
// 目标地址默认落到出口本机相同端口，监听端口留空时自动分配，限速与计费比统一由入口节点配置。
func (s *Service) CreatePortRule(ctx context.Context, in CreateRuleInput, actor *Principal) (*model.PortRule, error) {
	if actor == nil || actor.User.PackageID == nil || *actor.User.PackageID == 0 {
		return nil, fmt.Errorf("%w: current user has no package bound", ErrConflict)
	}
	protocol := strings.ToLower(strings.TrimSpace(in.Protocol))
	if protocol == "" {
		protocol = "tcp"
	}
	// both 表示同一监听端口同时启用 TCP 与 UDP，前端以开关形式提交。
	if protocol != "tcp" && protocol != "udp" && protocol != "both" {
		return nil, fmt.Errorf("unsupported protocol %q", protocol)
	}
	transport := strings.ToLower(strings.TrimSpace(in.Transport))
	if transport == "" {
		transport = "quic"
	}
	// 协议固定生效：quic 只走 QUIC，tls 只走 TLS1.3+yamux，不做跨协议降级。
	if transport != "quic" && transport != "tls" {
		return nil, fmt.Errorf("unsupported transport %q", transport)
	}

	pkg, err := s.getPackage(ctx, *actor.User.PackageID)
	if err != nil {
		return nil, fmt.Errorf("package: %w", err)
	}
	if !pkg.Enabled {
		return nil, fmt.Errorf("%w: package %s is disabled", ErrConflict, pkg.Name)
	}
	node, err := s.getNode(ctx, in.Node)
	if err != nil {
		return nil, fmt.Errorf("node: %w", err)
	}
	if node.Type != model.NodeTypeIngress {
		return nil, fmt.Errorf("%w: node %s is not an ingress node", ErrConflict, node.Code)
	}
	if !containsCode(pkg.IngressNodes, node.Code) {
		return nil, fmt.Errorf("%w: node %s is not bound to package %s", ErrConflict, node.Code, pkg.Name)
	}
	egressCode := strings.TrimSpace(in.EgressNode)
	if egressCode == "" {
		return nil, fmt.Errorf("egress_node is required")
	}
	if !containsCode(pkg.EgressNodes, egressCode) {
		return nil, fmt.Errorf("%w: egress node %s is not bound to package %s", ErrConflict, egressCode, pkg.Name)
	}
	egressNode, err := s.getNode(ctx, egressCode)
	if err != nil {
		return nil, fmt.Errorf("egress node: %w", err)
	}
	if egressNode.Type != model.NodeTypeEgress {
		return nil, fmt.Errorf("%w: node %s is not an egress node", ErrConflict, egressCode)
	}
	if pkg.RuleLimit > 0 {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
			Where("package_id = ? AND direction = ?", pkg.ID, model.RuleDirectionEntry).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count >= int64(pkg.RuleLimit) {
			return nil, fmt.Errorf("%w: package %s allows at most %d port rules", ErrConflict, pkg.Name, pkg.RuleLimit)
		}
	}
	port, err := s.resolveListenPort(ctx, node.Code, protocol, in.ListenPort, 0)
	if err != nil {
		return nil, err
	}
	targetAddr := strings.TrimSpace(in.TargetAddr)
	if targetAddr == "" {
		return nil, fmt.Errorf("target_addr is required")
	}

	rule := &model.PortRule{
		Direction:   model.RuleDirectionEntry,
		PackageID:   pkg.ID,
		Node:        node.Code,
		EgressNode:  egressCode,
		Name:        strings.TrimSpace(in.Name),
		Protocol:    protocol,
		Transport:   transport,
		ListenPort:  port,
		TargetAddr:  targetAddr,
		OwnerUserID: &actor.User.ID,
		RatioUp:     1,
		RatioDown:   1,
		Enabled:     true,
		Status:      "active",
		Revision:    1,
	}

	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return nil, err
	}

	s.dispatch(ctx, *rule, signal.TypePortAdd)
	return rule, nil
}

type RuleFilter struct {
	Node      string
	Direction string
	PackageID uint
}

// ListPortRules 在 owner 非空时只返回该用户创建的规则，用于无跨用户权限的角色。
func (s *Service) ListPortRules(ctx context.Context, f RuleFilter, owner *uint) ([]model.PortRule, error) {
	var rules []model.PortRule
	q := s.db.WithContext(ctx).Order("id asc")
	if f.Node != "" {
		q = q.Where("node = ?", f.Node)
	}
	if f.Direction != "" {
		q = q.Where("direction = ?", f.Direction)
	}
	if f.PackageID > 0 {
		q = q.Where("package_id = ?", f.PackageID)
	}
	if owner != nil {
		q = q.Where("owner_user_id = ?", *owner)
	}
	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) GetPortRule(ctx context.Context, id uint) (*model.PortRule, error) {
	var rule model.PortRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rule, nil
}

// UpdatePortRule 允许修改名称、目标地址、出口节点、监听端口与启用状态；限速与计费比统一在节点配置。
func (s *Service) UpdatePortRule(ctx context.Context, id uint, in UpdateRuleInput) (*model.PortRule, error) {
	var rule model.PortRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	oldEgress := rule.EgressNode
	if in.Name != nil {
		rule.Name = strings.TrimSpace(*in.Name)
	}
	if in.TargetAddr != nil {
		target := strings.TrimSpace(*in.TargetAddr)
		if target == "" {
			return nil, fmt.Errorf("target_addr is required")
		}
		rule.TargetAddr = target
	}
	if in.EgressNode != nil {
		if rule.Direction != model.RuleDirectionEntry {
			return nil, fmt.Errorf("%w: only entry rules can change egress node", ErrConflict)
		}
		egressCode := strings.TrimSpace(*in.EgressNode)
		if egressCode == "" {
			return nil, fmt.Errorf("egress_node is required")
		}
		pkg, err := s.getPackage(ctx, rule.PackageID)
		if err != nil {
			return nil, fmt.Errorf("package: %w", err)
		}
		if !pkg.Enabled {
			return nil, fmt.Errorf("%w: package %s is disabled", ErrConflict, pkg.Name)
		}
		if !containsCode(pkg.EgressNodes, egressCode) {
			return nil, fmt.Errorf("%w: egress node %s is not bound to package %s", ErrConflict, egressCode, pkg.Name)
		}
		egressNode, err := s.getNode(ctx, egressCode)
		if err != nil {
			return nil, fmt.Errorf("egress node: %w", err)
		}
		if egressNode.Type != model.NodeTypeEgress {
			return nil, fmt.Errorf("%w: node %s is not an egress node", ErrConflict, egressCode)
		}
		rule.EgressNode = egressCode
	}
	if in.ListenPort != nil {
		port, err := s.resolveListenPort(ctx, rule.Node, rule.Protocol, *in.ListenPort, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.ListenPort = port
	}
	if in.Transport != nil {
		transport := strings.ToLower(strings.TrimSpace(*in.Transport))
		if transport != "quic" && transport != "tls" {
			return nil, fmt.Errorf("unsupported transport %q", *in.Transport)
		}
		rule.Transport = transport
	}
	if in.Enabled != nil {
		rule.Enabled = *in.Enabled
	}

	rule.Revision++
	if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
		return nil, err
	}

	// 换出口时先清理旧出口上的合成规则，避免旧节点继续保留落地规则。
	if oldEgress != rule.EgressNode {
		s.removeStaleExitRules(ctx, rule, oldEgress)
	}
	s.dispatch(ctx, rule, signal.TypePortUpdate)
	return &rule, nil
}

func (s *Service) DeletePortRule(ctx context.Context, id uint) error {
	var rule model.PortRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	// 入口规则删除时需要同时通知各出口节点清理合成出口规则。
	s.dispatchRemove(ctx, rule)
	if err := s.db.WithContext(ctx).Delete(&rule).Error; err != nil {
		return err
	}
	return nil
}

// resolveListenPort 校验显式端口；want 为 0 时在 10000..60000 内自动选一个未被占用的端口。
func (s *Service) resolveListenPort(ctx context.Context, node, protocol string, want int, excludeID uint) (int, error) {
	if want == 0 {
		return s.allocListenPort(ctx, node, protocol)
	}
	if want < 1 || want > 65535 {
		return 0, fmt.Errorf("listen_port must be within 1..65535")
	}
	type portProtocol struct {
		ID       uint
		Protocol string
	}
	var rows []portProtocol
	q := s.db.WithContext(ctx).Model(&model.PortRule{}).
		Select("id", "protocol").
		Where("direction = ? AND node = ? AND listen_port = ?", model.RuleDirectionEntry, node, want)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return 0, err
	}
	for _, row := range rows {
		if entryProtocolsConflict(protocol, row.Protocol) {
			return 0, fmt.Errorf("%w: listen port %d already used on node %s", ErrConflict, want, node)
		}
	}
	return want, nil
}

// entryProtocolsConflict 判断同一监听端口上的两条入口规则是否会争用协议栈。
func entryProtocolsConflict(a, b string) bool {
	normalize := func(v string) string {
		if v == "" {
			return "tcp"
		}
		return v
	}
	a = normalize(a)
	b = normalize(b)
	// both 同时占用 TCP 与 UDP，与任何协议都冲突。
	return a == "both" || b == "both" || a == b
}

func (s *Service) allocListenPort(ctx context.Context, node, protocol string) (int, error) {
	type portProtocol struct {
		ListenPort int
		Protocol   string
	}
	var rows []portProtocol
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
		Select("listen_port", "protocol").
		Where("direction = ? AND node = ?", model.RuleDirectionEntry, node).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	used := make(map[int][]string, len(rows))
	for _, row := range rows {
		used[row.ListenPort] = append(used[row.ListenPort], row.Protocol)
	}
	total := allocatedPortMax - allocatedPortMin + 1
	start := rand.IntN(total)
	for i := 0; i < total; i++ {
		port := allocatedPortMin + (start+i)%total
		conflict := false
		for _, p := range used[port] {
			if entryProtocolsConflict(protocol, p) {
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}
		return port, nil
	}
	return 0, fmt.Errorf("%w: no free listen port on node %s", ErrConflict, node)
}

// dispatch 下发入口规则及所选出口节点上的合成出口规则；历史规则未选出口时仍向套餐全部出口扇出。
func (s *Service) dispatch(ctx context.Context, rule model.PortRule, typ string) {
	if rule.Direction != model.RuleDirectionEntry {
		s.publishRule(typ, rule.Node, s.ruleDTO(ctx, rule))
		return
	}

	dto := s.ruleDTO(ctx, rule)
	nodes, err := s.ruleExitNodes(ctx, rule)
	if err != nil {
		s.log.Error("dispatch rule failed: resolve egress nodes",
			zap.Uint("rule_id", rule.ID), zap.Uint("package_id", rule.PackageID), zap.Error(err))
		return
	}
	targets := make([]signal.ExitTarget, 0, len(nodes))
	for _, n := range nodes {
		// 只下发在线出口：离线出口的旧地址可能是黑洞连接，避免入口连接被拖到超时。
		if n.Status != "online" {
			s.log.Debug("skip offline egress node",
				zap.String("node", n.Code), zap.Uint("rule_id", rule.ID), zap.String("status", n.Status))
			continue
		}
		// 同一出口节点的多台部署实例各下发一条地址，入口侧按列表轮询并逐条故障转移。
		targets = append(targets, s.nodeExitTargets(ctx, n, rule.ID)...)
	}
	dto.Exits = targets
	s.publishRule(typ, rule.Node, dto)

	for _, n := range nodes {
		// 离线出口无需下发合成规则，重新上线后会由状态变更重推。
		if n.Status != "online" {
			continue
		}
		s.publishRule(typ, n.Code, s.exitRuleDTO(ctx, rule, n.Code))
	}
}

// nodeExitTargets 返回该出口节点当前在线实例的数据面地址；无实例记录时回退节点级地址。
func (s *Service) nodeExitTargets(ctx context.Context, node model.Node, ruleID uint) []signal.ExitTarget {
	var instances []model.NodeInstance
	if err := s.db.WithContext(ctx).
		Where("node_code = ? AND online = ?", node.Code, true).
		Order("id asc").Find(&instances).Error; err != nil {
		s.log.Warn("resolve exit instances failed", zap.String("node", node.Code), zap.Error(err))
	}
	out := make([]signal.ExitTarget, 0, len(instances))
	for _, inst := range instances {
		quicAddr := resolveNodeAddr(node, inst.QUICAddr)
		tlsAddr := resolveNodeAddr(node, inst.TLSAddr)
		if quicAddr == "" && tlsAddr == "" {
			continue
		}
		out = append(out, signal.ExitTarget{
			RuleID:   ruleID,
			Node:     node.Code,
			QUICAddr: quicAddr,
			TLSAddr:  tlsAddr,
		})
	}
	if len(out) > 0 {
		return out
	}
	quicAddr, tlsAddr := resolveNodeAddrs(node)
	if quicAddr == "" && tlsAddr == "" {
		s.log.Warn("package egress node has no data-plane address",
			zap.String("node", node.Code), zap.Uint("rule_id", ruleID))
		return out
	}
	return append(out, signal.ExitTarget{
		RuleID:   ruleID,
		Node:     node.Code,
		QUICAddr: quicAddr,
		TLSAddr:  tlsAddr,
	})
}

// dispatchRemove 删除入口规则时同步通知所选出口节点清理合成出口规则。
func (s *Service) dispatchRemove(ctx context.Context, rule model.PortRule) {
	s.publishRule(signal.TypePortRemove, rule.Node, s.ruleDTO(ctx, rule))
	if rule.Direction != model.RuleDirectionEntry {
		return
	}
	if rule.EgressNode != "" {
		if node, err := s.getNode(ctx, rule.EgressNode); err == nil {
			s.publishRule(signal.TypePortRemove, node.Code, s.exitRuleDTO(ctx, rule, node.Code))
		}
		return
	}
	nodes, err := s.packageEgressNodes(ctx, rule.PackageID)
	if err != nil {
		s.log.Error("dispatch remove failed: resolve package egress nodes",
			zap.Uint("rule_id", rule.ID), zap.Uint("package_id", rule.PackageID), zap.Error(err))
		return
	}
	for _, n := range nodes {
		s.publishRule(signal.TypePortRemove, n.Code, s.exitRuleDTO(ctx, rule, n.Code))
	}
}

// removeStaleExitRules 在规则切换出口后清理旧节点上的合成规则。
func (s *Service) removeStaleExitRules(ctx context.Context, rule model.PortRule, oldEgress string) {
	var nodes []model.Node
	if oldEgress != "" {
		node, err := s.getNode(ctx, oldEgress)
		if err != nil {
			return
		}
		nodes = append(nodes, *node)
	} else {
		pkgNodes, err := s.packageEgressNodes(ctx, rule.PackageID)
		if err != nil {
			return
		}
		nodes = pkgNodes
	}
	for _, n := range nodes {
		if n.Code == rule.EgressNode {
			continue
		}
		s.publishRule(signal.TypePortRemove, n.Code, s.exitRuleDTO(ctx, rule, n.Code))
	}
}

// purgePackageExitRules 在套餐移除出口节点或停用时，向不再承担该套餐的出口节点清理合成出口规则。
func (s *Service) purgePackageExitRules(ctx context.Context, pkgID uint, nodes []string) {
	if pkgID == 0 || len(nodes) == 0 {
		return
	}
	var entries []model.PortRule
	if err := s.db.WithContext(ctx).
		Where("package_id = ? AND direction = ?", pkgID, model.RuleDirectionEntry).
		Order("id asc").Find(&entries).Error; err != nil {
		s.log.Error("purge package exit rules: list entries failed",
			zap.Uint("package_id", pkgID), zap.Error(err))
		return
	}
	for _, rule := range entries {
		// 指定出口的规则只在目标节点清理；历史规则仍按套餐全部出口清理。
		if rule.EgressNode != "" && !containsCode(nodes, rule.EgressNode) {
			continue
		}
		for _, node := range nodes {
			s.publishRule(signal.TypePortRemove, node, s.exitRuleDTO(ctx, rule, node))
		}
	}
}

func (s *Service) publishRule(typ, node string, dto signal.PortRule) {
	env, err := signal.NewEnvelope(typ, node, dto)
	if err != nil {
		s.log.Error("dispatch rule failed: marshal", zap.Uint("rule_id", dto.RuleID), zap.Error(err))
		return
	}
	raw, err := env.Marshal()
	if err != nil {
		s.log.Error("dispatch rule failed: encode envelope", zap.Uint("rule_id", dto.RuleID), zap.Error(err))
		return
	}
	if err := s.mqtt.Publish(signal.CmdTopic(node), raw); err != nil {
		s.log.Warn("dispatch rule publish failed",
			zap.String("type", typ), zap.String("node", node),
			zap.Uint("rule_id", dto.RuleID), zap.Error(err))
	}
}

// packageEgressNodes 返回套餐内可用于扇出的出口节点；套餐停用时返回空，入口侧据此 fail-closed。
func (s *Service) packageEgressNodes(ctx context.Context, pkgID uint) ([]model.Node, error) {
	var pkg model.Package
	if err := s.db.WithContext(ctx).First(&pkg, pkgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !pkg.Enabled {
		return nil, nil
	}
	out := make([]model.Node, 0, len(pkg.EgressNodes))
	for _, code := range pkg.EgressNodes {
		node, err := s.getNode(ctx, code)
		if err != nil {
			s.log.Warn("package egress node missing", zap.Uint("package_id", pkg.ID), zap.String("node", code), zap.Error(err))
			continue
		}
		if node.Type != model.NodeTypeEgress {
			continue
		}
		out = append(out, *node)
	}
	return out, nil
}

// ruleExitNodes 返回规则实际使用的出口节点；历史规则未指定出口时回退套餐全部出口。
func (s *Service) ruleExitNodes(ctx context.Context, rule model.PortRule) ([]model.Node, error) {
	nodes, err := s.packageEgressNodes(ctx, rule.PackageID)
	if err != nil {
		return nil, err
	}
	if rule.EgressNode == "" {
		return nodes, nil
	}
	for _, n := range nodes {
		if n.Code == rule.EgressNode {
			return []model.Node{n}, nil
		}
	}
	// 套餐停用或所选出口已被移除时返回空，入口规则无法解析出口即 fail-closed。
	return nil, nil
}

// exitRuleDTO 构造下发给出口节点的合成规则：RuleID 与入口规则一致，出口侧按 Header 校验；
// 出口计费比取出口节点配置，出口节点不限速因此速率字段保持 0。
func (s *Service) exitRuleDTO(ctx context.Context, rule model.PortRule, node string) signal.PortRule {
	dto := signal.PortRule{
		RuleID:     rule.ID,
		Revision:   rule.Revision,
		Name:       rule.Name,
		Direction:  model.RuleDirectionExit,
		PackageID:  rule.PackageID,
		Node:       node,
		Protocol:   rule.Protocol,
		Transport:  normalizeTransport(rule.Transport),
		TargetAddr: rule.TargetAddr,
		RatioUp:    1,
		RatioDown:  1,
		Enabled:    rule.Enabled,
	}
	if n, err := s.getNode(ctx, node); err == nil {
		dto.RatioUp = normalizeRatio(n.RatioUp)
		dto.RatioDown = normalizeRatio(n.RatioDown)
	}
	return dto
}

// normalizeTransport 兼容历史数据：未记录传输方式的规则按 QUIC 优先。
func normalizeTransport(v string) string {
	if strings.TrimSpace(strings.ToLower(v)) == "tls" {
		return "tls"
	}
	return "quic"
}

func mbpsToBps(mbps float64) int64 {
	if mbps <= 0 {
		return 0
	}
	return int64(mbps * 1024 * 1024)
}

// redispatchPackageEntries 在套餐出口集合变化后升版重推入口规则，让入口 Agent 刷新出口列表。
func (s *Service) redispatchPackageEntries(ctx context.Context, pkgID uint) {
	if pkgID == 0 {
		return
	}
	var entries []model.PortRule
	if err := s.db.WithContext(ctx).
		Where("package_id = ? AND direction = ?", pkgID, model.RuleDirectionEntry).
		Order("id asc").Find(&entries).Error; err != nil {
		s.log.Error("redispatch package entries failed",
			zap.Uint("package_id", pkgID), zap.Error(err))
		return
	}
	for _, rule := range entries {
		rule.Revision++
		if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
			s.log.Warn("redispatch package entries: bump revision failed",
				zap.Uint("rule_id", rule.ID), zap.Error(err))
			continue
		}
		s.dispatch(ctx, rule, signal.TypePortUpdate)
	}
}

func (s *Service) HandleMessage(topic string, payload []byte) {
	nodeCode := signal.NodeFromTopic(topic)
	if nodeCode == "" {
		s.log.Warn("mqtt message with unknown topic", zap.String("topic", topic))
		return
	}
	// 控制面只保留离线状态等信令，业务上报统一走带 JWT 的 HTTP 接口。
	if strings.HasSuffix(topic, signal.StateSuffix) {
		s.handleState(nodeCode, payload)
		return
	}
	s.log.Debug("mqtt message ignored", zap.String("topic", topic))
}

// HandleReport 处理 Agent 经 JWT 上报的信令，type 与 MQTT 信封保持一致。
func (s *Service) HandleReport(ctx context.Context, nodeCode string, env signal.Envelope) {
	s.touchNode(nodeCode)

	switch env.Type {
	case signal.TypeTrafficReport:
		s.ingestTraffic(ctx, nodeCode, env)
	case signal.TypeNodeStatus:
		s.ingestNodeStatus(ctx, nodeCode, env)
	case signal.TypeNodeMetrics:
		s.ingestNodeMetrics(ctx, nodeCode, env)
	case signal.TypePortEvent:
		s.ingestPortEvent(ctx, env)
	case signal.TypeNodeSyncRequest:
		s.syncRules(ctx, nodeCode)
	default:
		s.log.Debug("report type ignored", zap.String("node", nodeCode), zap.String("type", env.Type))
	}
}

func (s *Service) touchNode(nodeCode string) {
	if err := s.db.Model(&model.Node{}).Where("code = ?", nodeCode).
		Updates(map[string]any{"status": "online", "last_seen_at": time.Now()}).Error; err != nil {
		s.log.Warn("update node heartbeat failed", zap.String("node", nodeCode), zap.Error(err))
	}
}

func (s *Service) ingestTraffic(ctx context.Context, nodeCode string, env signal.Envelope) {
	var report signal.TrafficReport
	if err := json.Unmarshal(env.Payload, &report); err != nil {
		s.log.Warn("traffic report decode failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range report.Items {
			stat := model.TrafficStat{
				RuleID:     item.RuleID,
				NodeCode:   nodeCode,
				UpBytes:    item.UpBytes,
				DownBytes:  item.DownBytes,
				BillBytes:  item.BillBytes,
				RecordedAt: now,
			}
			if err := tx.Create(&stat).Error; err != nil {
				return err
			}
			res := tx.Model(&model.PortRule{}).Where("id = ?", item.RuleID).
				Update("used_bytes", item.UsedBytes)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				s.log.Warn("traffic for unknown rule", zap.Uint("rule_id", item.RuleID))
			}
		}
		return nil
	})
	if err != nil {
		s.log.Error("traffic report persist failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}

	// 套餐流量上限是汇总口径：本次上报涉及的套餐重新计算入口规则合计用量。
	ruleIDs := make([]uint, 0, len(report.Items))
	for _, item := range report.Items {
		ruleIDs = append(ruleIDs, item.RuleID)
	}
	if len(ruleIDs) == 0 {
		return
	}
	var pkgIDs []uint
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).Where("id IN ?", ruleIDs).
		Distinct().Pluck("package_id", &pkgIDs).Error; err != nil {
		s.log.Warn("traffic report: resolve package failed", zap.Error(err))
		return
	}
	for _, pkgID := range pkgIDs {
		if pkgID > 0 {
			s.enforcePackageQuota(ctx, pkgID)
		}
	}
}

// enforcePackageQuota 汇总套餐内入口规则计费流量，超过套餐上限时停用并清理全部入口规则。
func (s *Service) enforcePackageQuota(ctx context.Context, pkgID uint) {
	pkg, err := s.getPackage(ctx, pkgID)
	if err != nil {
		s.log.Warn("enforce package quota: load package failed", zap.Uint("package_id", pkgID), zap.Error(err))
		return
	}
	if pkg.TrafficLimitGB <= 0 {
		return
	}
	limit := int64(pkg.TrafficLimitGB * 1024 * 1024 * 1024)
	var used int64
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).
		Where("package_id = ? AND direction = ?", pkgID, model.RuleDirectionEntry).
		Select("COALESCE(SUM(used_bytes),0)").Scan(&used).Error; err != nil {
		s.log.Warn("enforce package quota: sum used failed", zap.Uint("package_id", pkgID), zap.Error(err))
		return
	}
	if used < limit {
		return
	}

	var entries []model.PortRule
	if err := s.db.WithContext(ctx).
		Where("package_id = ? AND direction = ? AND enabled = ?", pkgID, model.RuleDirectionEntry, true).
		Find(&entries).Error; err != nil {
		s.log.Warn("enforce package quota: list entries failed", zap.Uint("package_id", pkgID), zap.Error(err))
		return
	}
	for _, rule := range entries {
		rule.Enabled = false
		rule.Status = "quota_exhausted"
		rule.Revision++
		if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
			s.log.Warn("enforce package quota: disable rule failed", zap.Uint("rule_id", rule.ID), zap.Error(err))
			continue
		}
		s.dispatchRemove(ctx, rule)
	}
	if len(entries) > 0 {
		s.log.Info("package traffic limit reached",
			zap.Uint("package_id", pkgID), zap.String("package", pkg.Name),
			zap.Int64("used_bytes", used), zap.Int64("limit_bytes", limit))
	}
}

func (s *Service) ingestNodeStatus(ctx context.Context, nodeCode string, env signal.Envelope) {
	var status signal.NodeStatus
	if err := json.Unmarshal(env.Payload, &status); err != nil {
		s.log.Warn("node status decode failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	updates := map[string]any{
		"status":       "online",
		"last_seen_at": time.Now(),
		"version":      status.Version,
	}
	if err := s.db.WithContext(ctx).Model(&model.Node{}).Where("code = ?", nodeCode).Updates(updates).Error; err != nil {
		s.log.Warn("node status persist failed", zap.String("node", nodeCode), zap.Error(err))
	}
}

func (s *Service) ingestPortEvent(ctx context.Context, env signal.Envelope) {
	var ev signal.PortEvent
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		s.log.Warn("port event decode failed", zap.String("node", env.NodeCode), zap.Error(err))
		return
	}
	status := ""
	switch ev.Event {
	case "quota_exhausted":
		status = "quota_exhausted"
	case "bind_failed", "rule_error":
		status = "error"
	case "rule_applied":
		status = "active"
	default:
		s.log.Info("port event", zap.Uint("rule_id", ev.RuleID), zap.String("event", ev.Event), zap.String("detail", ev.Detail))
		return
	}
	if err := s.db.WithContext(ctx).Model(&model.PortRule{}).Where("id = ?", ev.RuleID).
		Update("status", status).Error; err != nil {
		s.log.Warn("port event persist failed", zap.Uint("rule_id", ev.RuleID), zap.Error(err))
	}
}

// syncRules 下发节点全量规则：自身承载的入口规则按套餐出口扇出，
// 出口节点则收到套餐内所有入口规则的合成出口规则。
func (s *Service) syncRules(ctx context.Context, nodeCode string) {
	var rules []model.PortRule
	if err := s.db.WithContext(ctx).
		Where("node = ?", nodeCode).
		Order("id asc").Find(&rules).Error; err != nil {
		s.log.Warn("sync rules query failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}

	dtos := make([]signal.PortRule, 0, len(rules))
	seen := make(map[uint]struct{}, len(rules))
	for _, rule := range rules {
		dto := s.ruleDTO(ctx, rule)
		if rule.Direction == model.RuleDirectionEntry {
			nodes, err := s.ruleExitNodes(ctx, rule)
			if err != nil {
				s.log.Warn("sync rules: resolve egress nodes failed",
					zap.Uint("rule_id", rule.ID), zap.Uint("package_id", rule.PackageID), zap.Error(err))
				continue
			}
			targets := make([]signal.ExitTarget, 0, len(nodes))
			for _, n := range nodes {
				// 出口节点可能有多台部署实例，按在线实例逐条下发，入口侧轮询即负载均衡。
				targets = append(targets, s.nodeExitTargets(ctx, n, rule.ID)...)
			}
			dto.Exits = targets
		}
		seen[rule.ID] = struct{}{}
		dtos = append(dtos, dto)
	}

	// 该节点若是出口节点，补齐套餐内所有入口规则对应的合成出口规则。
	var pkgs []model.Package
	if err := s.db.WithContext(ctx).Find(&pkgs).Error; err != nil {
		s.log.Warn("sync rules: list packages failed", zap.String("node", nodeCode), zap.Error(err))
	} else {
		for _, pkg := range pkgs {
			if !pkg.Enabled || !containsCode(pkg.EgressNodes, nodeCode) {
				continue
			}
			var entries []model.PortRule
			if err := s.db.WithContext(ctx).
				Where("package_id = ? AND direction = ?", pkg.ID, model.RuleDirectionEntry).
				Order("id asc").Find(&entries).Error; err != nil {
				s.log.Warn("sync rules: list package entries failed",
					zap.Uint("package_id", pkg.ID), zap.Error(err))
				continue
			}
			for _, entry := range entries {
				if _, ok := seen[entry.ID]; ok {
					continue
				}
				// 指定了其他出口的规则不应下发到本节点。
				if entry.EgressNode != "" && entry.EgressNode != nodeCode {
					continue
				}
				dtos = append(dtos, s.exitRuleDTO(ctx, entry, nodeCode))
			}
		}
	}

	env, err := signal.NewEnvelope(signal.TypePortSync, nodeCode, signal.PortSync{Rules: dtos})
	if err != nil {
		s.log.Warn("sync rules: build envelope failed", zap.Error(err))
		return
	}
	raw, err := env.Marshal()
	if err != nil {
		s.log.Warn("sync rules: marshal failed", zap.Error(err))
		return
	}
	if err := s.mqtt.Publish(signal.CmdTopic(nodeCode), raw); err != nil {
		s.log.Warn("sync rules publish failed", zap.String("node", nodeCode), zap.Error(err))
	}
}

func (s *Service) handleState(nodeCode string, payload []byte) {
	var state signal.StatePayload
	if err := json.Unmarshal(payload, &state); err != nil {
		s.log.Warn("state decode failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	var node model.Node
	if err := s.db.WithContext(context.Background()).Where("code = ?", nodeCode).First(&node).Error; err != nil {
		s.log.Warn("state node lookup failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}

	// 带实例 ID 的状态只影响该实例；节点在线状态由在线实例数汇总，避免一个实例下线误伤整台出口。
	if state.InstanceID != "" {
		res := s.db.WithContext(context.Background()).Model(&model.NodeInstance{}).
			Where("node_code = ? AND instance_id = ?", nodeCode, state.InstanceID).
			Update("online", state.Online)
		if res.Error != nil {
			s.log.Warn("state instance persist failed",
				zap.String("node", nodeCode), zap.String("instance", state.InstanceID), zap.Error(res.Error))
			return
		}
		s.recomputeNodeStatus(context.Background(), &node)
		// 任一出口实例上下线都会改变入口侧的可用地址集合，必须带版本号重推。
		if node.Type == model.NodeTypeEgress && res.RowsAffected > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.redispatchEgressDependents(ctx, nodeCode)
		}
		return
	}

	status := "offline"
	if state.Online {
		status = "online"
	}
	if node.Status == status {
		return
	}
	if err := s.db.Model(&model.Node{}).Where("code = ?", nodeCode).Update("status", status).Error; err != nil {
		s.log.Warn("state persist failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	// 出口上下线会改变入口规则的可用出口列表，需要重推入口规则让负载均衡跳过离线出口。
	if node.Type == model.NodeTypeEgress {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.redispatchEgressDependents(ctx, nodeCode)
	}
}

type StatsResult struct {
	UpBytes   int64               `json:"up_bytes"`
	DownBytes int64               `json:"down_bytes"`
	BillBytes int64               `json:"bill_bytes"`
	Records   []model.TrafficStat `json:"records"`
}

func (s *Service) PortRuleStats(ctx context.Context, id uint, from, to time.Time) (*StatsResult, error) {
	if _, err := s.GetPortRule(ctx, id); err != nil {
		return nil, err
	}
	var totals struct {
		Up   int64
		Down int64
		Bill int64
	}
	if err := s.db.WithContext(ctx).Model(&model.TrafficStat{}).
		Select("COALESCE(SUM(up_bytes),0) AS up, COALESCE(SUM(down_bytes),0) AS down, COALESCE(SUM(bill_bytes),0) AS bill").
		Where("rule_id = ? AND recorded_at BETWEEN ? AND ?", id, from, to).
		Scan(&totals).Error; err != nil {
		return nil, err
	}

	var records []model.TrafficStat
	if err := s.db.WithContext(ctx).
		Where("rule_id = ? AND recorded_at BETWEEN ? AND ?", id, from, to).
		Order("recorded_at desc").Limit(200).Find(&records).Error; err != nil {
		return nil, err
	}

	return &StatsResult{UpBytes: totals.Up, DownBytes: totals.Down, BillBytes: totals.Bill, Records: records}, nil
}

// ruleDTO 组装入口规则；限速与计费比取入口节点的节点级配置，规则自身不再携带。
func (s *Service) ruleDTO(ctx context.Context, rule model.PortRule) signal.PortRule {
	dto := signal.PortRule{
		RuleID:     rule.ID,
		Revision:   rule.Revision,
		Name:       rule.Name,
		Direction:  rule.Direction,
		PackageID:  rule.PackageID,
		Node:       rule.Node,
		Protocol:   rule.Protocol,
		Transport:  normalizeTransport(rule.Transport),
		ListenPort: rule.ListenPort,
		TargetAddr: rule.TargetAddr,
		RatioUp:    rule.RatioUp,
		RatioDown:  rule.RatioDown,
		QuotaBytes: rule.QuotaBytes,
		UsedBytes:  rule.UsedBytes,
		Enabled:    rule.Enabled,
	}
	if rule.Direction == model.RuleDirectionEntry {
		if node, err := s.getNode(ctx, rule.Node); err == nil {
			dto.RateUpBps = mbpsToBps(node.RateUpMBps)
			dto.RateDownBps = mbpsToBps(node.RateDownMBps)
			dto.RatioUp = normalizeRatio(node.RatioUp)
			dto.RatioDown = normalizeRatio(node.RatioDown)
		}
	}
	return dto
}

func normalizeRatio(v float64) float64 {
	if v <= 0 {
		return 1.0
	}
	return v
}

func containsCode(codes []string, code string) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
