package forward

import (
	"context"
	stdtls "crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	quic "github.com/refraction-networking/uquic"
	"go.uber.org/zap"

	"aoe-tunnel/internal/agent/rule"
	"aoe-tunnel/internal/agent/tunnel"
	"aoe-tunnel/pkg/signal"
)

// ExitServer 同时提供 QUIC 主隧道与 TLS1.3+yamux 降级隧道，两者共用同一套规则校验。
type ExitServer struct {
	nodeCode string
	certs    *tunnel.Certs
	mgr      *rule.Manager
	log      *zap.Logger

	quicLn *quic.Listener

	quicTr  *quic.Transport
	quicPkt net.PacketConn
	tlsLn   net.Listener

	mu     sync.Mutex
	active map[io.Closer]struct{}

	closed atomic.Bool
	wg     sync.WaitGroup
}

func NewExitServer(quicAddr, tlsAddr, nodeCode string, certs *tunnel.Certs, mgr *rule.Manager, log *zap.Logger) (*ExitServer, error) {
	if quicAddr == "" || tlsAddr == "" {
		return nil, fmt.Errorf("exit: quic and tls addr are required")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", quicAddr)
	if err != nil {
		return nil, fmt.Errorf("exit: resolve %s: %w", quicAddr, err)
	}
	pkt, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("exit: listen udp %s: %w", quicAddr, err)
	}
	tr := &quic.Transport{Conn: pkt, ConnectionIDLength: 8}
	ln, err := tr.Listen(certs.ServerConfig(signal.ALPNQUIC), tunnel.QUICConfig())
	if err != nil {
		_ = tr.Close()
		_ = pkt.Close()
		return nil, fmt.Errorf("exit: quic listen: %w", err)
	}

	tlsLn, err := net.Listen("tcp", tlsAddr)
	if err != nil {
		_ = ln.Close()
		_ = tr.Close()
		_ = pkt.Close()
		return nil, fmt.Errorf("exit: listen tls %s: %w", tlsAddr, err)
	}

	log.Info("exit server started",
		zap.String("quic_addr", pkt.LocalAddr().String()),
		zap.String("tls_addr", tlsLn.Addr().String()))

	return &ExitServer{
		nodeCode: nodeCode,
		certs:    certs,
		mgr:      mgr,
		log:      log,
		quicLn:   ln,
		quicTr:   tr,
		quicPkt:  pkt,
		tlsLn:    tlsLn,
		active:   make(map[io.Closer]struct{}),
	}, nil
}

func (s *ExitServer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.acceptQUIC(ctx)
	}()
	go func() {
		defer wg.Done()
		s.acceptTLS(ctx)
	}()
	wg.Wait()
}

func (s *ExitServer) acceptQUIC(ctx context.Context) {
	for {
		conn, err := s.quicLn.Accept(ctx)
		if err != nil {
			if s.closed.Load() || ctx.Err() != nil || errors.Is(err, quic.ErrServerClosed) {
				return
			}
			s.log.Warn("exit quic accept failed", zap.Error(err))
			continue
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serveQUICConn(ctx, conn)
		}()
	}
}

func (s *ExitServer) serveQUICConn(ctx context.Context, conn quic.Connection) {
	s.log.Debug("exit quic connection accepted", zap.String("remote", conn.RemoteAddr().String()))
	for {
		st, err := conn.AcceptStream(ctx)
		if err != nil {
			s.log.Debug("exit quic connection closed", zap.Error(err))
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serveStream(ctx, tunnel.WrapQUICStream(st))
		}()
	}
}

func (s *ExitServer) acceptTLS(ctx context.Context) {
	for {
		c, err := s.tlsLn.Accept()
		if err != nil {
			if s.closed.Load() || ctx.Err() != nil {
				return
			}
			s.log.Warn("exit tls accept failed", zap.Error(err))
			// 监听套接字未关闭时短暂退避，避免错误风暴。
			time.Sleep(100 * time.Millisecond)
			continue
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serveTLSConn(ctx, c)
		}()
	}
}

func (s *ExitServer) serveTLSConn(ctx context.Context, c net.Conn) {
	hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	tc := stdtls.Server(c, s.certs.StdServerConfig(signal.ALPNFallback))
	err := tc.HandshakeContext(hctx)
	cancel()
	if err != nil {
		_ = c.Close()
		s.log.Debug("exit tls handshake failed", zap.String("remote", c.RemoteAddr().String()), zap.Error(err))
		return
	}

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	sess, err := yamux.Server(tc, cfg)
	if err != nil {
		_ = tc.Close()
		s.log.Warn("exit yamux server failed", zap.Error(err))
		return
	}
	untrack := s.track(sess)
	defer untrack()
	defer sess.Close()

	s.log.Debug("exit fallback connection accepted", zap.String("remote", c.RemoteAddr().String()))
	for {
		st, err := sess.AcceptStream()
		if err != nil {
			s.log.Debug("exit fallback connection closed", zap.Error(err))
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serveStream(ctx, tunnel.WrapYamuxStream(st))
		}()
	}
}

func (s *ExitServer) serveStream(ctx context.Context, st tunnel.DataStream) {
	defer st.Close()

	_ = st.SetReadDeadline(time.Now().Add(10 * time.Second))
	hdr, err := tunnel.ReadHeader(st)
	if err != nil {
		s.log.Debug("exit: bad stream header", zap.Error(err))
		st.CancelRead()
		return
	}
	_ = st.SetReadDeadline(time.Time{})
	s.serveExit(ctx, st, hdr)
}

// serveExit 是最终出口：校验规则后 Dial 目标地址，TCP 与 UDP 走同一套 Stream 桥接。
func (s *ExitServer) serveExit(ctx context.Context, st tunnel.DataStream, hdr tunnel.Header) {
	ruleID := uint(hdr.RuleID)
	r, ok := s.mgr.Get(ruleID)
	if !ok || r.Direction != "exit" || r.Node != s.nodeCode || !r.Enabled ||
		(r.Protocol != "tcp" && r.Protocol != "udp" && r.Protocol != "both") {
		s.log.Warn("exit: reject stream",
			zap.Uint("rule_id", ruleID), zap.Bool("found", ok),
			zap.String("direction", r.Direction), zap.String("node", r.Node),
			zap.String("protocol", r.Protocol), zap.Bool("enabled", r.Enabled))
		st.CancelRead()
		return
	}

	network := r.Protocol
	if network == "" {
		network = "tcp"
	}
	// 双协议规则按入口 Stream 首部携带的网络类型落地，避免 UDP 流量被错误当作 TCP 拨号。
	if network == "both" {
		network = hdr.Network
	}
	if network != "tcp" && network != "udp" {
		s.log.Warn("exit: reject invalid stream network",
			zap.Uint("rule_id", ruleID), zap.String("rule_protocol", r.Protocol), zap.String("network", network))
		st.CancelRead()
		return
	}
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var d net.Dialer
	target, err := d.DialContext(dctx, network, r.TargetAddr)
	if err != nil {
		s.log.Warn("exit: dial target failed",
			zap.Uint("rule_id", r.RuleID), zap.String("network", network), zap.String("target", r.TargetAddr), zap.Error(err))
		st.CancelRead()
		return
	}

	up, down := Bridge(target, st, nil)
	s.log.Debug("exit connection closed",
		zap.Uint("rule_id", r.RuleID), zap.String("target", r.TargetAddr),
		zap.Int64("up_bytes", up), zap.Int64("down_bytes", down))
}

func (s *ExitServer) track(c io.Closer) func() {
	s.mu.Lock()
	s.active[c] = struct{}{}
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.active, c)
		s.mu.Unlock()
	}
}

func (s *ExitServer) Close() {
	if s.closed.CompareAndSwap(false, true) {
		_ = s.quicLn.Close()
		_ = s.tlsLn.Close()
	}
	_ = s.quicTr.Close()
	_ = s.quicPkt.Close()

	s.mu.Lock()
	for c := range s.active {
		_ = c.Close()
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.log.Warn("exit server close timeout")
	}
}
