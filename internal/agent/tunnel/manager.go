package tunnel

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Hop 是入口拨号到出口所需的数据面地址；链式中转已移除，每连接只拨一跳。
type Hop struct {
	Code     string
	QUICAddr string
	TLSAddr  string
	// Transport 为 quic 或 tls；协议固定生效，不做跨协议降级。
	Transport string
}

// Manager 负责单跳拨号；传输协议由规则决定，空值保留旧版自动降级行为。
type Manager struct {
	pool             *ClientPool
	fallback         *FallbackPool
	forceFallback    bool
	handshakeTimeout time.Duration
	log              *zap.Logger
}

func NewManager(pool *ClientPool, fallback *FallbackPool, forceFallback bool, handshakeTimeout time.Duration, log *zap.Logger) *Manager {
	if handshakeTimeout <= 0 {
		handshakeTimeout = 3 * time.Second
	}
	return &Manager{
		pool:             pool,
		fallback:         fallback,
		forceFallback:    forceFallback,
		handshakeTimeout: handshakeTimeout,
		log:              log,
	}
}

// OpenStream 对单跳拨号：显式协议只走对应链路，失败直接返回错误；空协议保留旧版自动降级。
func (m *Manager) OpenStream(ctx context.Context, hop Hop) (DataStream, string, error) {
	switch hop.Transport {
	case "quic":
		if hop.QUICAddr == "" {
			return nil, "", fmt.Errorf("hop %s: no quic address", hop.Code)
		}
		cctx, cancel := context.WithTimeout(ctx, m.handshakeTimeout)
		defer cancel()
		st, err := m.pool.OpenStream(cctx, hop.QUICAddr)
		if err != nil {
			return nil, "", fmt.Errorf("hop %s: quic %s: %w", hop.Code, hop.QUICAddr, err)
		}
		return st, "quic", nil
	case "tls":
		if hop.TLSAddr == "" {
			return nil, "", fmt.Errorf("hop %s: no tls address", hop.Code)
		}
		cctx, cancel := context.WithTimeout(ctx, m.handshakeTimeout)
		defer cancel()
		st, err := m.fallback.OpenStream(cctx, hop.TLSAddr)
		if err != nil {
			return nil, "", fmt.Errorf("hop %s: tls %s: %w", hop.Code, hop.TLSAddr, err)
		}
		return st, "tls-yamux", nil
	}

	// 未指定传输协议的存量规则保留 QUIC 优先、失败降级 TLS 的旧行为。
	useQUIC := !m.forceFallback && hop.QUICAddr != ""
	if useQUIC {
		cctx, cancel := context.WithTimeout(ctx, m.handshakeTimeout)
		st, err := m.pool.OpenStream(cctx, hop.QUICAddr)
		cancel()
		if err == nil {
			return st, "quic", nil
		}
		m.log.Warn("hop quic tunnel failed, fallback to tls",
			zap.String("hop", hop.Code), zap.String("quic_addr", hop.QUICAddr), zap.Error(err))
	}

	if hop.TLSAddr == "" {
		return nil, "", fmt.Errorf("hop %s: no fallback address", hop.Code)
	}
	cctx, cancel := context.WithTimeout(ctx, m.handshakeTimeout)
	defer cancel()
	st, err := m.fallback.OpenStream(cctx, hop.TLSAddr)
	if err != nil {
		return nil, "", fmt.Errorf("hop %s: fallback %s: %w", hop.Code, hop.TLSAddr, err)
	}
	return st, "tls-yamux", nil
}
