package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	utls "github.com/refraction-networking/utls"
	"go.uber.org/zap"

	"aoe-tunnel/pkg/signal"
)

type FallbackConfig struct {
	Certs            *Certs
	SNI              string
	HandshakeTimeout time.Duration
	Log              *zap.Logger
}

// FallbackPool 按出口 TLS 地址复用 yamux 会话，QUIC 不可用时接管入口流量。
type FallbackPool struct {
	cfg      FallbackConfig
	mu       sync.Mutex
	sessions map[string]*fallbackSession
}

type fallbackSession struct {
	mu   sync.Mutex
	sess *yamux.Session
	conn net.Conn
}

func NewFallbackPool(cfg FallbackConfig) (*FallbackPool, error) {
	// 启动时校验预设可用；每次握手重新构造，避免共享 spec 被原地改写。
	if _, err := fallbackSpec(signal.ALPNFallback); err != nil {
		return nil, fmt.Errorf("load chrome tls preset: %w", err)
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 3 * time.Second
	}
	return &FallbackPool{
		cfg:      cfg,
		sessions: make(map[string]*fallbackSession),
	}, nil
}

// fallbackSpec 用 Chrome 的 TLS 预设伪装 JA4，ALPN/ALPS 必须替换成降级通道的私有值。
func fallbackSpec(alpn string) (*utls.ClientHelloSpec, error) {
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err != nil {
		return nil, err
	}
	for _, ext := range spec.Extensions {
		switch e := ext.(type) {
		case *utls.ALPNExtension:
			e.AlpnProtocols = []string{alpn}
		case *utls.ApplicationSettingsExtension:
			e.SupportedProtocols = []string{alpn}
		case *utls.ApplicationSettingsExtensionNew:
			e.SupportedProtocols = []string{alpn}
		}
	}
	return &spec, nil
}

func (p *FallbackPool) OpenStream(ctx context.Context, addr string) (DataStream, error) {
	if addr == "" {
		return nil, fmt.Errorf("tunnel: empty fallback address")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		sess, err := p.session(ctx, addr)
		if err != nil {
			return nil, err
		}
		st, err := sess.OpenStream()
		if err == nil {
			return WrapYamuxStream(st), nil
		}
		lastErr = err
		p.drop(addr, sess)
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("tunnel: open yamux stream to %s: %w", addr, lastErr)
}

func (p *FallbackPool) session(ctx context.Context, addr string) (*yamux.Session, error) {
	e := p.entry(addr)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sess != nil && !e.sess.IsClosed() {
		return e.sess, nil
	}
	if e.sess != nil {
		_ = e.sess.Close()
		_ = e.conn.Close()
		e.sess, e.conn = nil, nil
	}
	sess, conn, err := p.dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	e.sess, e.conn = sess, conn
	return sess, nil
}

func (p *FallbackPool) entry(addr string) *fallbackSession {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.sessions[addr]
	if !ok {
		e = &fallbackSession{}
		p.sessions[addr] = e
	}
	return e
}

func (p *FallbackPool) drop(addr string, sess *yamux.Session) {
	e := p.entry(addr)
	e.mu.Lock()
	if e.sess == sess {
		e.sess, e.conn = nil, nil
	}
	conn := e.conn
	e.mu.Unlock()
	_ = sess.Close()
	if conn != nil {
		_ = conn.Close()
	}
}

// Drop 关闭指定地址的 yamux 会话，只影响该跳；节点地址变更后由 Manager 调用。
func (p *FallbackPool) Drop(addr string) {
	e := p.entry(addr)
	e.mu.Lock()
	sess, conn := e.sess, e.conn
	e.sess, e.conn = nil, nil
	e.mu.Unlock()
	if sess != nil {
		_ = sess.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

func (p *FallbackPool) dial(ctx context.Context, addr string) (*yamux.Session, net.Conn, error) {
	var d net.Dialer
	d.Timeout = p.cfg.HandshakeTimeout
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("tunnel: dial fallback %s: %w", addr, err)
	}
	// TLS 握手与 QUIC 一样受 3s 上限约束，超时交由上层断流。
	_ = conn.SetDeadline(time.Now().Add(p.cfg.HandshakeTimeout))
	// 每次握手使用全新预设，避免多跳/多地址复用时共享 spec 被原地改写。
	spec, err := fallbackSpec(signal.ALPNFallback)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("tunnel: load tls preset: %w", err)
	}
	uc := utls.UClient(conn, p.cfg.Certs.FallbackClientConfig(p.cfg.SNI), utls.HelloCustom)
	if err := uc.ApplyPreset(spec); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("tunnel: apply tls preset: %w", err)
	}
	if err := uc.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("tunnel: fallback handshake %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Time{})

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	sess, err := yamux.Client(uc, cfg)
	if err != nil {
		_ = uc.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("tunnel: yamux client %s: %w", addr, err)
	}
	p.cfg.Log.Info("fallback tunnel established", zap.String("addr", addr))
	go func() {
		<-sess.CloseChan()
		_ = conn.Close()
		p.cfg.Log.Debug("fallback tunnel closed", zap.String("addr", addr))
	}()
	return sess, conn, nil
}

func (p *FallbackPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, e := range p.sessions {
		e.mu.Lock()
		sess, conn := e.sess, e.conn
		e.sess, e.conn = nil, nil
		e.mu.Unlock()
		if sess != nil {
			_ = sess.Close()
		}
		if conn != nil {
			_ = conn.Close()
		}
		delete(p.sessions, addr)
	}
}
