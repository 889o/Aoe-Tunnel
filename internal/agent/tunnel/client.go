package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	quic "github.com/refraction-networking/uquic"
	"go.uber.org/zap"

	"aoe-tunnel/pkg/signal"
)

type DialConfig struct {
	Certs            *Certs
	SNI              string
	HandshakeTimeout time.Duration
	Log              *zap.Logger
}

// ClientPool 按出口地址复用 QUIC 连接，Stream 多路复用，避免每条 TCP 连接都重建握手。
type ClientPool struct {
	cfg   DialConfig
	mu    sync.Mutex
	conns map[string]*pooledConn
}

type pooledConn struct {
	mu   sync.Mutex
	conn quic.Connection
}

func NewClientPool(cfg DialConfig) (*ClientPool, error) {
	// 启动时校验预设可用；每次拨号重新构造，避免多跳场景下共享 spec 被握手过程改写。
	if _, err := ChromeSpec(signal.ALPNQUIC); err != nil {
		return nil, fmt.Errorf("load quic preset: %w", err)
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 3 * time.Second
	}
	return &ClientPool{
		cfg:   cfg,
		conns: make(map[string]*pooledConn),
	}, nil
}

func (p *ClientPool) OpenStream(ctx context.Context, addr string) (DataStream, error) {
	if addr == "" {
		return nil, fmt.Errorf("tunnel: empty exit address")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		conn, err := p.conn(ctx, addr)
		if err != nil {
			return nil, err
		}
		st, err := conn.OpenStreamSync(ctx)
		if err == nil {
			return WrapQUICStream(st), nil
		}
		lastErr = err
		p.drop(addr, conn)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("tunnel: open stream to %s: %w", addr, lastErr)
}

func (p *ClientPool) conn(ctx context.Context, addr string) (quic.Connection, error) {
	e := p.entry(addr)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.conn != nil && e.conn.Context().Err() == nil {
		return e.conn, nil
	}
	if e.conn != nil {
		_ = e.conn.CloseWithError(0, "reconnect")
		e.conn = nil
	}
	c, err := p.dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	e.conn = c
	return c, nil
}

func (p *ClientPool) entry(addr string) *pooledConn {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.conns[addr]
	if !ok {
		e = &pooledConn{}
		p.conns[addr] = e
	}
	return e
}

func (p *ClientPool) drop(addr string, conn quic.Connection) {
	e := p.entry(addr)
	e.mu.Lock()
	if e.conn == conn {
		e.conn = nil
	}
	e.mu.Unlock()
	_ = conn.CloseWithError(0, "stale")
}

// Drop 关闭指定地址的连接，只影响该跳；节点地址变更后由 Manager 调用。
func (p *ClientPool) Drop(addr string) {
	e := p.entry(addr)
	e.mu.Lock()
	conn := e.conn
	e.conn = nil
	e.mu.Unlock()
	if conn != nil {
		_ = conn.CloseWithError(0, "hop address changed")
	}
}

func (p *ClientPool) dial(ctx context.Context, addr string) (quic.Connection, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("tunnel: resolve %s: %w", addr, err)
	}
	pkt, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("tunnel: open udp socket: %w", err)
	}
	// 每次拨号使用全新预设：uQUIC 的 ClientHello 扩展在握手过程中会被原地改写，
	// 多跳/多地址复用时共享同一 spec 会导致后续握手失败。
	spec, err := ChromeSpec(signal.ALPNQUIC)
	if err != nil {
		_ = pkt.Close()
		return nil, fmt.Errorf("tunnel: load quic preset: %w", err)
	}
	tr := &quic.UTransport{Transport: &quic.Transport{Conn: pkt}, QUICSpec: spec}
	// QUIC 握手超过 3s 即判定失败，交由 fallback 链路接管。
	dctx, cancel := context.WithTimeout(ctx, p.cfg.HandshakeTimeout)
	defer cancel()
	c, err := tr.Dial(dctx, udpAddr, p.cfg.Certs.ClientConfig(p.cfg.SNI, signal.ALPNQUIC), ClientQUICConfig(p.cfg.Log, addr))
	if err != nil {
		_ = tr.Close()
		_ = pkt.Close()
		return nil, fmt.Errorf("tunnel: quic dial %s: %w", addr, err)
	}
	p.cfg.Log.Info("quic tunnel established", zap.String("addr", addr))
	go func() {
		<-c.Context().Done()
		_ = tr.Close()
		_ = pkt.Close()
		p.cfg.Log.Debug("quic tunnel closed", zap.String("addr", addr), zap.Error(c.Context().Err()))
	}()
	return c, nil
}

func (p *ClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, e := range p.conns {
		e.mu.Lock()
		if e.conn != nil {
			_ = e.conn.CloseWithError(0, "shutdown")
			e.conn = nil
		}
		e.mu.Unlock()
		delete(p.conns, addr)
	}
}
