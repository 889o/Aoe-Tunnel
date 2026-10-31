package forward

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"aoe-tunnel/internal/agent/stat"
	"aoe-tunnel/internal/agent/tunnel"
	"aoe-tunnel/pkg/signal"
)

// exitOpenTimeout 限制一次连接在多个出口间故障转移的总耗时，避免单连接长时间占住入口。
const exitOpenTimeout = 15 * time.Second

// openExitStream 从 start 位置开始轮询出口列表，遇到拨号或写头失败自动切换到下一个出口。
// 多出口部署时这既是负载均衡入口，也承担单节点故障转移。
func openExitStream(ctx context.Context, hops *tunnel.Manager, log *zap.Logger, exits []signal.ExitTarget, start int, network, transport string) (tunnel.DataStream, signal.ExitTarget, string, error) {
	if len(exits) == 0 {
		return nil, signal.ExitTarget{}, "", fmt.Errorf("no exit targets")
	}
	ctx, cancel := context.WithTimeout(ctx, exitOpenTimeout)
	defer cancel()

	if transport == "" {
		transport = "quic"
	}

	var lastErr error
	for i := 0; i < len(exits); i++ {
		target := exits[(start+i)%len(exits)]
		st, mode, err := hops.OpenStream(ctx, tunnel.Hop{
			Code:      target.Node,
			QUICAddr:  target.QUICAddr,
			TLSAddr:   target.TLSAddr,
			Transport: transport,
		})
		if err != nil {
			lastErr = fmt.Errorf("exit %s: %w", target.Node, err)
			log.Warn("entry tunnel failed, trying next exit", zap.String("exit", target.Node), zap.Error(err))
			continue
		}
		if err := st.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			_ = st.Close()
			lastErr = fmt.Errorf("exit %s: set write deadline: %w", target.Node, err)
			continue
		}
		if err := tunnel.WriteHeader(st, tunnel.Header{Version: tunnel.HeaderVersion, RuleID: uint32(target.RuleID), Network: network}); err != nil {
			_ = st.Close()
			lastErr = fmt.Errorf("exit %s: write header: %w", target.Node, err)
			log.Warn("entry write header failed, trying next exit", zap.String("exit", target.Node), zap.Error(err))
			continue
		}
		_ = st.SetWriteDeadline(time.Time{})
		return st, target, mode, nil
	}
	return nil, signal.ExitTarget{}, "", lastErr
}

type EntryStat struct {
	RuleID      uint
	ListenPort  int
	ActiveConns int64
	UpBytes     int64
	DownBytes   int64
}

type entryListener struct {
	ruleID     uint
	listenPort int
	ln         net.Listener

	mu    sync.RWMutex
	rule  signal.PortRule
	conns map[net.Conn]struct{}

	active atomic.Int64
	up     atomic.Int64
	down   atomic.Int64
	closed atomic.Bool
	// nextExit 让入口连接在套餐出口间轮询，避免所有流量压到同一个出口。
	nextExit atomic.Uint64
}

func (e *entryListener) update(r signal.PortRule) {
	e.mu.Lock()
	e.rule = r
	e.mu.Unlock()
}

func (e *entryListener) current() signal.PortRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rule
}

func (e *entryListener) addConn(c net.Conn) {
	e.mu.Lock()
	if e.conns == nil {
		e.conns = make(map[net.Conn]struct{})
	}
	e.conns[c] = struct{}{}
	e.mu.Unlock()
}

func (e *entryListener) removeConn(c net.Conn) {
	e.mu.Lock()
	delete(e.conns, c)
	e.mu.Unlock()
}

func (e *entryListener) close() {
	if e.closed.CompareAndSwap(false, true) {
		_ = e.ln.Close()
	}
	e.mu.Lock()
	for c := range e.conns {
		_ = c.Close()
	}
	e.conns = nil
	e.mu.Unlock()
}

// Forwarder 只负责入口侧：按规则暴露 TCP 监听或 UDP 监听，逐跳交给 Manager 拨号。
type Forwarder struct {
	nodeCode   string
	listenHost string
	hops       *tunnel.Manager
	log        *zap.Logger

	mu         sync.Mutex
	entries    map[uint]*entryListener
	udpEntries map[uint]*udpEntry
	// nodeUpBps/nodeDownBps/nodeLimits 是入口节点级限速，所有 TCP 入口监听共享同一对令牌桶。
	nodeUpBps   int64
	nodeDownBps int64
	nodeLimits  *stat.Limiters
	wg          sync.WaitGroup
}

func NewForwarder(nodeCode, listenHost string, hops *tunnel.Manager, log *zap.Logger) *Forwarder {
	return &Forwarder{
		nodeCode:   nodeCode,
		listenHost: listenHost,
		hops:       hops,
		log:        log,
		entries:    make(map[uint]*entryListener),
		udpEntries: make(map[uint]*udpEntry),
	}
}

func (f *Forwarder) Apply(prev *signal.PortRule, cur signal.PortRule) {
	if cur.Direction != "entry" || cur.Node != f.nodeCode {
		return
	}
	f.setNodeRate(cur.RateUpBps, cur.RateDownBps)
	if !cur.Enabled {
		f.Remove(cur)
		return
	}
	switch cur.Protocol {
	case "", "tcp":
		f.applyTCP(cur)
	case "udp":
		f.applyUDP(cur)
	case "both":
		// 同端口双协议：两种监听各自独立维护，任一侧失败不影响另一侧继续服务。
		f.upsertTCP(cur)
		f.upsertUDP(cur)
	default:
		f.log.Warn("entry: unsupported protocol", zap.Uint("rule_id", cur.RuleID), zap.String("protocol", cur.Protocol))
	}
}

func (f *Forwarder) applyTCP(cur signal.PortRule) {
	f.removeUDPListener(cur.RuleID)
	f.upsertTCP(cur)
}

func (f *Forwarder) applyUDP(cur signal.PortRule) {
	f.removeTCPListener(cur.RuleID)
	f.upsertUDP(cur)
}

func (f *Forwarder) removeTCPListener(ruleID uint) {
	f.mu.Lock()
	existing := f.entries[ruleID]
	delete(f.entries, ruleID)
	f.mu.Unlock()
	if existing != nil {
		existing.close()
		f.log.Info("entry listener replaced", zap.Uint("rule_id", ruleID), zap.Int("old_port", existing.listenPort))
	}
}

func (f *Forwarder) removeUDPListener(ruleID uint) {
	f.mu.Lock()
	existing := f.udpEntries[ruleID]
	delete(f.udpEntries, ruleID)
	f.mu.Unlock()
	if existing != nil {
		existing.close()
	}
}

func (f *Forwarder) upsertTCP(cur signal.PortRule) {
	f.mu.Lock()
	existing := f.entries[cur.RuleID]
	if existing != nil && existing.listenPort == cur.ListenPort {
		existing.update(cur)
		f.mu.Unlock()
		return
	}
	delete(f.entries, cur.RuleID)
	f.mu.Unlock()

	if existing != nil {
		existing.close()
		f.log.Info("entry listener replaced", zap.Uint("rule_id", cur.RuleID), zap.Int("old_port", existing.listenPort), zap.Int("new_port", cur.ListenPort))
	}

	el, err := f.open(cur)
	if err != nil {
		f.log.Error("entry listener start failed",
			zap.Uint("rule_id", cur.RuleID), zap.Int("listen_port", cur.ListenPort), zap.Error(err))
		return
	}
	f.mu.Lock()
	f.entries[cur.RuleID] = el
	f.mu.Unlock()

	f.wg.Add(1)
	go f.acceptLoop(el)
}

func (f *Forwarder) upsertUDP(cur signal.PortRule) {
	f.mu.Lock()
	existing := f.udpEntries[cur.RuleID]
	if existing != nil && existing.listenPort == cur.ListenPort {
		existing.update(cur)
		f.mu.Unlock()
		return
	}
	delete(f.udpEntries, cur.RuleID)
	f.mu.Unlock()

	if existing != nil {
		existing.close()
		f.log.Info("udp entry listener replaced", zap.Uint("rule_id", cur.RuleID), zap.Int("old_port", existing.listenPort), zap.Int("new_port", cur.ListenPort))
	}

	ue, err := f.openUDP(cur)
	if err != nil {
		f.log.Error("udp entry listener start failed",
			zap.Uint("rule_id", cur.RuleID), zap.Int("listen_port", cur.ListenPort), zap.Error(err))
		return
	}
	f.mu.Lock()
	f.udpEntries[cur.RuleID] = ue
	f.mu.Unlock()
}

// setNodeRate 在入口规则携带的节点级限速变化时重建共享令牌桶；值未变不重建，避免丢令牌。
func (f *Forwarder) setNodeRate(upBps, downBps int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.nodeLimits != nil && f.nodeUpBps == upBps && f.nodeDownBps == downBps {
		return
	}
	if f.nodeLimits == nil && f.nodeUpBps == upBps && f.nodeDownBps == downBps && upBps == 0 && downBps == 0 {
		return
	}
	f.nodeUpBps = upBps
	f.nodeDownBps = downBps
	f.nodeLimits = stat.NewLimiters(upBps, downBps)
}

func (f *Forwarder) currentLimits() *stat.Limiters {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.nodeLimits
}

func (f *Forwarder) Remove(cur signal.PortRule) {
	if cur.Direction != "entry" || cur.Node != f.nodeCode {
		return
	}
	f.mu.Lock()
	el := f.entries[cur.RuleID]
	delete(f.entries, cur.RuleID)
	ue := f.udpEntries[cur.RuleID]
	delete(f.udpEntries, cur.RuleID)
	f.mu.Unlock()
	if el != nil {
		el.close()
		f.log.Info("entry listener stopped", zap.Uint("rule_id", cur.RuleID), zap.Int("listen_port", el.listenPort))
	}
	if ue != nil {
		ue.close()
		f.log.Info("udp entry listener stopped", zap.Uint("rule_id", cur.RuleID), zap.Int("listen_port", ue.listenPort))
	}
}

func (f *Forwarder) open(r signal.PortRule) (*entryListener, error) {
	addr := net.JoinHostPort(f.listenHost, strconv.Itoa(r.ListenPort))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	f.log.Info("entry listener started",
		zap.Uint("rule_id", r.RuleID), zap.String("listen", addr), zap.Int("exits", len(r.Exits)))
	return &entryListener{
		ruleID:     r.RuleID,
		listenPort: r.ListenPort,
		ln:         ln,
		rule:       r,
		conns:      make(map[net.Conn]struct{}),
	}, nil
}

func (f *Forwarder) acceptLoop(el *entryListener) {
	defer f.wg.Done()
	for {
		tc, err := el.ln.Accept()
		if err != nil {
			if el.closed.Load() {
				return
			}
			f.log.Warn("entry accept failed", zap.Uint("rule_id", el.ruleID), zap.Error(err))
			// 监听套接字未关闭时短暂退避，避免错误风暴。
			time.Sleep(100 * time.Millisecond)
			continue
		}
		el.active.Add(1)
		el.addConn(tc)
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			f.handleConn(el, tc)
		}()
	}
}

func (f *Forwarder) handleConn(el *entryListener, tc net.Conn) {
	defer el.active.Add(-1)
	defer el.removeConn(tc)
	defer tc.Close()

	r := el.current()
	limits := f.currentLimits()
	if len(r.Exits) == 0 {
		// 套餐停用或尚无出口规则时直接断开，避免连接悬空（fail-closed）。
		f.log.Debug("entry rule has no exits, close connection", zap.Uint("rule_id", r.RuleID))
		return
	}
	idx := int(el.nextExit.Add(1)-1) % len(r.Exits)
	st, target, mode, err := openExitStream(context.Background(), f.hops, f.log, r.Exits, idx, "tcp", r.Transport)
	if err != nil {
		f.log.Warn("entry tunnel failed for all exits",
			zap.Uint("rule_id", r.RuleID), zap.Int("exits", len(r.Exits)), zap.Error(err))
		return
	}
	f.log.Debug("entry stream opened",
		zap.Uint("rule_id", r.RuleID), zap.String("exit", target.Node), zap.Uint("exit_rule_id", target.RuleID), zap.String("mode", mode))

	up, down := Bridge(tc, st, limits)
	el.up.Add(up)
	el.down.Add(down)
	f.log.Debug("entry connection closed",
		zap.Uint("rule_id", r.RuleID), zap.String("mode", mode), zap.Int64("up_bytes", up), zap.Int64("down_bytes", down))
}

// QuotaExhausted 配额耗尽时由统计层回调：停止监听并断开现有连接，规则本身仍保留。
func (f *Forwarder) QuotaExhausted(ruleID uint) {
	f.mu.Lock()
	el := f.entries[ruleID]
	delete(f.entries, ruleID)
	ue := f.udpEntries[ruleID]
	delete(f.udpEntries, ruleID)
	f.mu.Unlock()
	if el != nil {
		el.close()
		f.log.Info("entry listener stopped by quota", zap.Uint("rule_id", ruleID), zap.Int("listen_port", el.listenPort))
	}
	if ue != nil {
		ue.close()
		f.log.Info("udp entry listener stopped by quota", zap.Uint("rule_id", ruleID), zap.Int("listen_port", ue.listenPort))
	}
}

func (f *Forwarder) Stats() []EntryStat {
	f.mu.Lock()
	listeners := make([]*entryListener, 0, len(f.entries))
	for _, el := range f.entries {
		listeners = append(listeners, el)
	}
	udpListeners := make([]*udpEntry, 0, len(f.udpEntries))
	for _, ue := range f.udpEntries {
		udpListeners = append(udpListeners, ue)
	}
	f.mu.Unlock()

	out := make([]EntryStat, 0, len(listeners)+len(udpListeners))
	for _, el := range listeners {
		out = append(out, EntryStat{
			RuleID:      el.ruleID,
			ListenPort:  el.listenPort,
			ActiveConns: el.active.Load(),
			UpBytes:     el.up.Load(),
			DownBytes:   el.down.Load(),
		})
	}
	for _, ue := range udpListeners {
		out = append(out, ue.stat())
	}
	return out
}

func (f *Forwarder) Close() {
	f.mu.Lock()
	entries := f.entries
	f.entries = make(map[uint]*entryListener)
	udpEntries := f.udpEntries
	f.udpEntries = make(map[uint]*udpEntry)
	f.mu.Unlock()

	for _, el := range entries {
		el.close()
	}
	for _, ue := range udpEntries {
		ue.close()
	}

	done := make(chan struct{})
	go func() {
		f.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		f.log.Warn("entry forwarder close timeout")
	}
}
