package tunnel

import (
	"os"
	"strings"
	"time"

	quic "github.com/refraction-networking/uquic"
	tls "github.com/refraction-networking/utls"
	"go.uber.org/zap"
)

const (
	envTPRandom  = "AGENT_TP_RANDOM"
	envTPProfile = "AGENT_TP_PROFILE"
)

// QUICConfig 同时用于客户端拨号和出口节点监听，保持服务端 TP 稳定。
// 本地接收窗口必须不小于 ClientHello 里随机化后的宣告窗口（宽区间上限 12MB 流 / 28MB 连接），
// 否则慢速读取时窗口更新滞后，对端按宣告窗口发送会触发 FLOW_CONTROL_ERROR 断链。
func QUICConfig() *quic.Config {
	return &quic.Config{
		HandshakeIdleTimeout:           3 * time.Second,
		MaxIdleTimeout:                 60 * time.Second,
		KeepAlivePeriod:                25 * time.Second,
		MaxIncomingStreams:             512,
		InitialStreamReceiveWindow:     16 << 20,
		MaxStreamReceiveWindow:         24 << 20,
		InitialConnectionReceiveWindow: 32 << 20,
		MaxConnectionReceiveWindow:     48 << 20,
	}
}

// ClientQUICConfig 只在客户端拨号时使用：注入 TP 随机化 profile 与摘要日志回调。
// 每次拨号都会调用，uQUIC 在新建连接握手前重新采样，连接池复用不会重复采样。
func ClientQUICConfig(log *zap.Logger, addr string) *quic.Config {
	cfg := QUICConfig()
	if !tpRandomEnabled() {
		return cfg
	}
	name := strings.TrimSpace(os.Getenv(envTPProfile))
	profile, ok := quic.TPRandomProfileByName(name)
	if !ok && log != nil {
		log.Warn("unknown tp profile, fallback to chrome", zap.String("env", envTPProfile), zap.String("value", name))
	}
	cfg.ClientTransportParamsProfile = profile
	cfg.OnClientTransportParams = func(tp quic.ClientTransportParams) {
		if log == nil {
			return
		}
		log.Debug("quic transport params sampled",
			zap.String("addr", addr),
			zap.Int64("max_idle_timeout_ms", tp.MaxIdleTimeout.Milliseconds()),
			zap.Uint64("initial_max_data", tp.InitialMaxData),
			zap.Uint64("initial_max_stream_data_bidi_local", tp.InitialMaxStreamDataBidiLocal),
			zap.Uint64("initial_max_stream_data_bidi_remote", tp.InitialMaxStreamDataBidiRemote),
			zap.Uint64("initial_max_stream_data_uni", tp.InitialMaxStreamDataUni),
			zap.Uint64("initial_max_streams_bidi", tp.MaxStreamsBidi),
			zap.Uint64("initial_max_streams_uni", tp.MaxStreamsUni),
			zap.Uint64("max_udp_payload_size", tp.MaxUDPPayloadSize),
			zap.Uint64("max_datagram_frame_size", tp.MaxDatagramFrameSize),
			zap.Uint16("grease_len", tp.GreaseLength),
		)
	}
	return cfg
}

// tpRandomEnabled 默认开启；显式传 0/false/off 等才关闭。
func tpRandomEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envTPRandom))) {
	case "", "1", "true", "on", "yes", "enable", "enabled":
		return true
	}
	return false
}

// ChromeSpec 取 Chrome 的 QUIC 预设做 JA4 伪装，只把 ALPN/ALPS 改写为项目自有的 aoe-tunnel-v1。
func ChromeSpec(alpn string) (*quic.QUICSpec, error) {
	spec, err := quic.QUICID2Spec(quic.QUICChrome_115_IPv4)
	if err != nil {
		return nil, err
	}
	if spec.ClientHelloSpec != nil {
		for _, ext := range spec.ClientHelloSpec.Extensions {
			switch e := ext.(type) {
			case *tls.ALPNExtension:
				e.AlpnProtocols = []string{alpn}
			case *tls.ApplicationSettingsExtension:
				e.SupportedProtocols = []string{alpn}
			}
		}
	}
	return &spec, nil
}
