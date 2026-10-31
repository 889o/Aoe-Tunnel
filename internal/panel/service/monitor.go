package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"

	"aoe-tunnel/internal/panel/model"
	"aoe-tunnel/pkg/signal"
)

// monitorPoints 是每个节点保留的时序点数：1s 一个点保留最近 60 秒，仅内存不持久化。
const monitorPoints = 60

type monitorSeries struct {
	points []signal.SystemMetrics
}

type monitorCache struct {
	mu   sync.Mutex
	data map[string]*monitorSeries
}

func newMonitorCache() *monitorCache {
	return &monitorCache{data: make(map[string]*monitorSeries)}
}

func (c *monitorCache) add(code string, m signal.SystemMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.data[code]
	if s == nil {
		s = &monitorSeries{points: make([]signal.SystemMetrics, 0, monitorPoints)}
		c.data[code] = s
	}
	s.points = append(s.points, m)
	if len(s.points) > monitorPoints {
		s.points = s.points[len(s.points)-monitorPoints:]
	}
}

func (c *monitorCache) history(code string) []signal.SystemMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.data[code]
	if s == nil {
		return []signal.SystemMetrics{}
	}
	out := make([]signal.SystemMetrics, len(s.points))
	copy(out, s.points)
	return out
}

func (c *monitorCache) latest(code string) (signal.SystemMetrics, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.data[code]
	if s == nil || len(s.points) == 0 {
		return signal.SystemMetrics{}, false
	}
	return s.points[len(s.points)-1], true
}

// ingestNodeMetrics 解析上报并写入内存时序缓存，节点不存在时忽略。
func (s *Service) ingestNodeMetrics(ctx context.Context, nodeCode string, env signal.Envelope) {
	var m signal.SystemMetrics
	if err := json.Unmarshal(env.Payload, &m); err != nil {
		s.log.Warn("node metrics decode failed", zap.String("node", nodeCode), zap.Error(err))
		return
	}
	s.monitor.add(nodeCode, m)
}

// MonitorNode 是监控总览卡片的一行：节点静态信息 + 最新资源快照。
type MonitorNode struct {
	model.Node
	Metrics *signal.SystemMetrics `json:"metrics,omitempty"`
}

// MonitorOverview 返回当前身份可见节点的最新资源快照；无 port:manage-all 时按端口规则归属隔离。
func (s *Service) MonitorOverview(ctx context.Context, actor *Principal) ([]MonitorNode, error) {
	nodes, err := s.ListNodes(ctx, actor)
	if err != nil {
		return nil, err
	}
	all := actor.Has(PermPortManageAll)
	visible := map[string]struct{}{}
	if !all {
		codes, err := s.ownedNodeCodes(ctx, actor.User.ID)
		if err != nil {
			return nil, err
		}
		visible = codes
	}

	out := make([]MonitorNode, 0, len(nodes))
	for _, n := range nodes {
		if !all {
			if _, ok := visible[n.Code]; !ok {
				continue
			}
		}
		item := MonitorNode{Node: n}
		if m, ok := s.monitor.latest(n.Code); ok {
			item.Metrics = &m
		}
		out = append(out, item)
	}
	return out, nil
}

// NodeStatHistory 返回节点最近 60 个时序点，遵循与总览一致的行级隔离。
func (s *Service) NodeStatHistory(ctx context.Context, code string, actor *Principal) ([]signal.SystemMetrics, error) {
	code = strings.TrimSpace(code)
	if _, err := s.getNode(ctx, code); err != nil {
		return nil, err
	}
	if err := s.ensureNodeAccess(ctx, actor, code); err != nil {
		return nil, err
	}
	return s.monitor.history(code), nil
}

// ownedNodeCodes 返回该用户名下端口规则涉及的节点，用于监控数据的行级隔离。
func (s *Service) ownedNodeCodes(ctx context.Context, ownerID uint) (map[string]struct{}, error) {
	var rules []model.PortRule
	if err := s.db.WithContext(ctx).
		Select("node").
		Where("owner_user_id = ?", ownerID).Find(&rules).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(rules))
	for _, r := range rules {
		if r.Node != "" {
			set[r.Node] = struct{}{}
		}
	}
	return set, nil
}

func (s *Service) ensureNodeAccess(ctx context.Context, actor *Principal, code string) error {
	if actor.Has(PermPortManageAll) {
		return nil
	}
	visible, err := s.ownedNodeCodes(ctx, actor.User.ID)
	if err != nil {
		return err
	}
	if _, ok := visible[code]; ok {
		return nil
	}
	return fmt.Errorf("%w: node %s is outside your port scope", ErrForbidden, code)
}
