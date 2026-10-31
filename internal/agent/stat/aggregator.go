package stat

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"aoe-tunnel/internal/agent/rule"
	"aoe-tunnel/pkg/signal"
)

type RuleStat struct {
	RuleID      uint
	ActiveConns int64
	UpBytes     int64
	DownBytes   int64
}

// Publisher 把信号投递到 MQTT report 主题，具体编码由调用方负责。
type Publisher func(typ string, payload any)

type Aggregator struct {
	mgr      *rule.Manager
	source   func() []RuleStat
	publish  Publisher
	onQuota  func(ruleID uint)
	interval time.Duration
	log      *zap.Logger

	mu     sync.Mutex
	states map[uint]*ruleState
}

type ruleState struct {
	seeded    bool
	lastUp    int64
	lastDown  int64
	used      int64
	exhausted bool
}

func NewAggregator(mgr *rule.Manager, source func() []RuleStat, publish Publisher, onQuota func(ruleID uint), interval time.Duration, log *zap.Logger) *Aggregator {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Aggregator{
		mgr:      mgr,
		source:   source,
		publish:  publish,
		onQuota:  onQuota,
		interval: interval,
		log:      log,
		states:   make(map[uint]*ruleState),
	}
}

func (a *Aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.Collect()
		}
	}
}

// Active 返回当前入口侧活跃连接总数，供心跳上报使用。
func (a *Aggregator) Active() int64 {
	var total int64
	for _, st := range a.source() {
		total += st.ActiveConns
	}
	return total
}

// Collect 计算本轮增量并上报；返回本轮是否有需要落库的流量。
func (a *Aggregator) Collect() bool {
	stats := a.source()
	a.mu.Lock()
	defer a.mu.Unlock()

	items := make([]signal.TrafficItem, 0, len(stats))
	for _, st := range stats {
		r, ok := a.mgr.Get(st.RuleID)
		if !ok || !r.Enabled {
			continue
		}
		s := a.states[st.RuleID]
		if s == nil {
			s = &ruleState{used: r.UsedBytes}
			a.states[st.RuleID] = s
		}
		// 面板可能已累计了其他来源的流量，取较大值避免重复计数。
		if r.UsedBytes > s.used && s.seeded {
			s.used = r.UsedBytes
		}
		s.seeded = true

		deltaUp := st.UpBytes - s.lastUp
		if deltaUp < 0 {
			// 监听重建后计数器归零，按全新计数处理。
			deltaUp = st.UpBytes
		}
		deltaDown := st.DownBytes - s.lastDown
		if deltaDown < 0 {
			deltaDown = st.DownBytes
		}
		s.lastUp, s.lastDown = st.UpBytes, st.DownBytes

		if s.exhausted {
			if r.QuotaBytes > 0 && r.QuotaBytes > s.used {
				s.exhausted = false
			} else {
				continue
			}
		}

		bill := int64(math.Round(float64(deltaUp)*r.RatioUp + float64(deltaDown)*r.RatioDown))
		s.used += bill
		if deltaUp > 0 || deltaDown > 0 {
			items = append(items, signal.TrafficItem{
				RuleID:    r.RuleID,
				UpBytes:   deltaUp,
				DownBytes: deltaDown,
				BillBytes: bill,
				UsedBytes: s.used,
			})
		}

		if r.QuotaBytes > 0 && s.used >= r.QuotaBytes {
			s.exhausted = true
			a.log.Warn("rule quota exhausted", zap.Uint("rule_id", r.RuleID), zap.Int64("used", s.used), zap.Int64("quota", r.QuotaBytes))
			if a.onQuota != nil {
				go a.onQuota(r.RuleID)
			}
			if a.publish != nil {
				a.publish(signal.TypePortEvent, signal.PortEvent{
					RuleID: r.RuleID,
					Event:  "quota_exhausted",
					Detail: fmt.Sprintf("used=%d quota=%d", s.used, r.QuotaBytes),
				})
			}
		}
	}

	for id := range a.states {
		found := false
		for _, st := range stats {
			if st.RuleID == id {
				found = true
				break
			}
		}
		if !found {
			delete(a.states, id)
		}
	}

	if len(items) == 0 {
		return false
	}
	if a.publish != nil {
		a.publish(signal.TypeTrafficReport, signal.TrafficReport{
			IntervalSec: int(a.interval.Seconds()),
			Items:       items,
		})
	}
	return true
}
