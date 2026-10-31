package forward

import (
	"context"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"aoe-tunnel/internal/agent/tunnel"
	"aoe-tunnel/pkg/signal"
)

// udpIdleTimeout 是单客户端无流量后释放隧道 Stream 的时间，避免 UDP 会话无限占用。
const udpIdleTimeout = 2 * time.Minute

// udpEntry 为入口 UDP 规则绑定一个 PacketConn；每个源地址对应一条隧道 Stream，
// 在 Stream 上按顺序收发数据包，出口侧再还原为 UDP。
type udpEntry struct {
	ruleID     uint
	listenPort int
	pkt        *net.UDPConn
	hops       *tunnel.Manager
	log        *zap.Logger

	mu      sync.RWMutex
	rule    signal.PortRule
	clients map[string]*udpClient

	up     atomic.Int64
	down   atomic.Int64
	closed atomic.Bool
	// nextExit 让同一规则的 UDP 客户端在套餐出口间轮询。
	nextExit atomic.Uint64
	done     chan struct{}
	wg       sync.WaitGroup
}

type udpClient struct {
	entry    *udpEntry
	addr     *net.UDPAddr
	key      string
	st       tunnel.DataStream
	lastSeen atomic.Int64
	once     sync.Once
}

func (c *udpClient) touch() {
	c.lastSeen.Store(time.Now().UnixNano())
}

func (c *udpClient) close() {
	c.once.Do(func() {
		_ = c.st.Close()
		c.st.CancelRead()
		c.entry.mu.Lock()
		if cur, ok := c.entry.clients[c.key]; ok && cur == c {
			delete(c.entry.clients, c.key)
		}
		c.entry.mu.Unlock()
	})
}

func (e *udpEntry) update(r signal.PortRule) {
	e.mu.Lock()
	e.rule = r
	e.mu.Unlock()
}

func (e *udpEntry) current() signal.PortRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rule
}

func (e *udpEntry) openClient(addr *net.UDPAddr) *udpClient {
	r := e.current()
	if len(r.Exits) == 0 {
		e.log.Debug("udp entry rule has no exits, drop packet", zap.Uint("rule_id", e.ruleID))
		return nil
	}
	idx := int(e.nextExit.Add(1)-1) % len(r.Exits)
	st, target, mode, err := openExitStream(context.Background(), e.hops, e.log, r.Exits, idx, "udp", r.Transport)
	if err != nil {
		e.log.Warn("udp entry tunnel failed for all exits",
			zap.Uint("rule_id", e.ruleID), zap.Int("exits", len(r.Exits)), zap.Error(err))
		return nil
	}

	c := &udpClient{entry: e, addr: addr, key: addr.String(), st: st}
	c.touch()
	e.mu.Lock()
	e.clients[c.key] = c
	e.mu.Unlock()
	e.log.Debug("udp entry stream opened",
		zap.Uint("rule_id", e.ruleID), zap.String("client", c.key), zap.String("exit", target.Node), zap.String("mode", mode))

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		c.readLoop()
	}()
	return c
}

func (c *udpClient) readLoop() {
	buf := make([]byte, bridgeBufferSize)
	for {
		n, err := c.st.Read(buf)
		if err != nil {
			c.close()
			return
		}
		if n > 0 {
			if _, werr := c.entry.pkt.WriteToUDP(buf[:n], c.addr); werr != nil {
				c.entry.log.Debug("udp entry write back failed", zap.Uint("rule_id", c.entry.ruleID), zap.Error(werr))
			}
			c.entry.down.Add(int64(n))
			c.touch()
		}
	}
}

func (e *udpEntry) readLoop() {
	defer e.wg.Done()
	buf := make([]byte, 64<<10)
	for {
		n, addr, err := e.pkt.ReadFromUDP(buf)
		if err != nil {
			if e.closed.Load() {
				return
			}
			e.log.Debug("udp entry read failed", zap.Uint("rule_id", e.ruleID), zap.Error(err))
			continue
		}
		key := addr.String()
		e.mu.RLock()
		c := e.clients[key]
		e.mu.RUnlock()
		if c == nil {
			c = e.openClient(addr)
			if c == nil {
				continue
			}
		}
		c.touch()
		if _, werr := c.st.Write(buf[:n]); werr != nil {
			e.log.Debug("udp entry write to stream failed", zap.Uint("rule_id", e.ruleID), zap.Error(werr))
			c.close()
			continue
		}
		e.up.Add(int64(n))
	}
}

func (e *udpEntry) sweepLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.done:
			return
		case <-ticker.C:
			deadline := time.Now().Add(-udpIdleTimeout).UnixNano()
			e.mu.RLock()
			clients := make([]*udpClient, 0, len(e.clients))
			for _, c := range e.clients {
				if c.lastSeen.Load() < deadline {
					clients = append(clients, c)
				}
			}
			e.mu.RUnlock()
			for _, c := range clients {
				c.close()
			}
		}
	}
}

func (e *udpEntry) close() {
	if e.closed.CompareAndSwap(false, true) {
		_ = e.pkt.Close()
		close(e.done)
	}
	e.mu.Lock()
	clients := e.clients
	e.clients = make(map[string]*udpClient)
	e.mu.Unlock()
	for _, c := range clients {
		c.close()
	}
}

func (e *udpEntry) stat() EntryStat {
	e.mu.RLock()
	active := len(e.clients)
	e.mu.RUnlock()
	return EntryStat{
		RuleID:      e.ruleID,
		ListenPort:  e.listenPort,
		ActiveConns: int64(active),
		UpBytes:     e.up.Load(),
		DownBytes:   e.down.Load(),
	}
}

// openUDP 绑定 UDP 监听并启动收包与空闲清理协程。
func (f *Forwarder) openUDP(r signal.PortRule) (*udpEntry, error) {
	addr := net.JoinHostPort(f.listenHost, strconv.Itoa(r.ListenPort))
	resolved, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	pkt, err := net.ListenUDP("udp", resolved)
	if err != nil {
		return nil, err
	}
	e := &udpEntry{
		ruleID:     r.RuleID,
		listenPort: r.ListenPort,
		pkt:        pkt,
		hops:       f.hops,
		log:        f.log,
		rule:       r,
		clients:    make(map[string]*udpClient),
		done:       make(chan struct{}),
	}
	f.log.Info("udp entry listener started",
		zap.Uint("rule_id", r.RuleID), zap.String("listen", pkt.LocalAddr().String()), zap.Int("exits", len(r.Exits)))

	e.wg.Add(2)
	go e.readLoop()
	go e.sweepLoop()
	return e, nil
}
