package rule

import (
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"aoe-tunnel/pkg/signal"
)

type Manager struct {
	store *Store
	log   *zap.Logger

	mu       sync.RWMutex
	onApply  func(prev *signal.PortRule, cur signal.PortRule)
	onRemove func(cur signal.PortRule)
}

func NewManager(dataDir string, log *zap.Logger) (*Manager, error) {
	store, err := Open(dataDir)
	if err != nil {
		return nil, err
	}
	return &Manager{store: store, log: log}, nil
}

// SetHooks 注册规则变更回调，转发层据此开闭监听；回调在存储事务外执行。
func (m *Manager) SetHooks(onApply func(prev *signal.PortRule, cur signal.PortRule), onRemove func(cur signal.PortRule)) {
	m.mu.Lock()
	m.onApply = onApply
	m.onRemove = onRemove
	m.mu.Unlock()
}

func (m *Manager) HandleCmd(env signal.Envelope) error {
	switch env.Type {
	case signal.TypePortAdd, signal.TypePortUpdate:
		var r signal.PortRule
		if err := json.Unmarshal(env.Payload, &r); err != nil {
			return fmt.Errorf("decode rule: %w", err)
		}
		prev, changed, err := m.store.Apply(r)
		if err != nil {
			return err
		}
		if !changed {
			m.log.Debug("rule unchanged", zap.Uint("rule_id", r.RuleID), zap.Int("revision", r.Revision))
			return nil
		}
		m.log.Info("rule applied",
			zap.String("type", env.Type), zap.Uint("rule_id", r.RuleID),
			zap.Int("revision", r.Revision), zap.Int("listen_port", r.ListenPort),
			zap.String("target", r.TargetAddr))
		m.applyHook(prev, r)

	case signal.TypePortRemove:
		var r signal.PortRule
		if err := json.Unmarshal(env.Payload, &r); err != nil {
			return fmt.Errorf("decode rule: %w", err)
		}
		removed, err := m.store.Remove(r.RuleID)
		if err != nil {
			return err
		}
		if removed == nil {
			return nil
		}
		m.log.Info("rule removed", zap.Uint("rule_id", r.RuleID))
		m.removeHook(*removed)

	case signal.TypePortSync:
		var payload signal.PortSync
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return fmt.Errorf("decode port sync: %w", err)
		}
		added, removed, err := m.store.ReplaceAll(payload.Rules)
		if err != nil {
			return err
		}
		for _, r := range added {
			m.applyHook(nil, r)
		}
		for _, r := range removed {
			m.removeHook(r)
		}
		m.log.Info("rule sync applied", zap.Int("added", len(added)), zap.Int("removed", len(removed)), zap.Int("total", m.store.Count()))

	default:
		m.log.Debug("cmd type ignored", zap.String("type", env.Type))
	}
	return nil
}

func (m *Manager) applyHook(prev *signal.PortRule, cur signal.PortRule) {
	m.mu.RLock()
	hook := m.onApply
	m.mu.RUnlock()
	if hook != nil {
		hook(prev, cur)
	}
}

func (m *Manager) removeHook(cur signal.PortRule) {
	m.mu.RLock()
	hook := m.onRemove
	m.mu.RUnlock()
	if hook != nil {
		hook(cur)
	}
}

func (m *Manager) Snapshot() []signal.PortRule         { return m.store.Snapshot() }
func (m *Manager) Get(id uint) (signal.PortRule, bool) { return m.store.Get(id) }
func (m *Manager) Count() int                          { return m.store.Count() }
func (m *Manager) Close() error                        { return m.store.Close() }
