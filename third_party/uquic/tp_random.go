package quic

import (
	"crypto/rand"
	"math/big"
	"time"

	tls "github.com/refraction-networking/utls"
)

// Uint64Range 是闭区间 [Min, Max]；Min >= Max 时退化为固定值。
type Uint64Range struct {
	Min, Max uint64
}

// DurationRange 是闭区间 [Min, Max]；Min >= Max 时退化为固定值。
type DurationRange struct {
	Min, Max time.Duration
}

// TPRandomProfile 描述客户端 Transport Parameter 的采样区间。
// 只覆盖 Chrome 预设本来就发送的参数，参数集合与顺序保持不变，
// 避免引入非浏览器特征。区间均取自不影响握手与流控的安全范围。
type TPRandomProfile struct {
	MaxIdleTimeout       DurationRange
	InitialMaxData       Uint64Range
	InitialMaxStreamData Uint64Range // bidi local/remote/uni 各自独立采样
	MaxStreamsBidi       Uint64Range
	MaxStreamsUni        Uint64Range
	MaxUDPPayloadSize    Uint64Range
	MaxDatagramFrameSize Uint64Range
}

// ClientTransportParams 是一次握手实际采样出的值，仅用于日志核对。
type ClientTransportParams struct {
	MaxIdleTimeout                 time.Duration
	InitialMaxData                 uint64
	InitialMaxStreamDataBidiLocal  uint64
	InitialMaxStreamDataBidiRemote uint64
	InitialMaxStreamDataUni        uint64
	MaxStreamsBidi                 uint64
	MaxStreamsUni                  uint64
	MaxUDPPayloadSize              uint64
	MaxDatagramFrameSize           uint64
	GreaseLength                   uint16
}

// TPRandomProfileByName 返回内置配置集。未知名称回退 chrome 并返回 false，
// 由调用方决定是否记录告警。
func TPRandomProfileByName(name string) (*TPRandomProfile, bool) {
	switch name {
	case "", "chrome", "default":
		return ChromeTPRandomProfile(), true
	case "chrome-wide", "wide":
		return ChromeWideTPRandomProfile(), true
	}
	return ChromeTPRandomProfile(), false
}

// ChromeTPRandomProfile 在 Chrome 115 抓包值附近做小幅度抖动。
func ChromeTPRandomProfile() *TPRandomProfile {
	return &TPRandomProfile{
		MaxIdleTimeout:       DurationRange{Min: 30 * time.Second, Max: 60 * time.Second},
		InitialMaxData:       Uint64Range{Min: 10 << 20, Max: 20 << 20},
		InitialMaxStreamData: Uint64Range{Min: 4 << 20, Max: 8 << 20},
		MaxStreamsBidi:       Uint64Range{Min: 64, Max: 160},
		MaxStreamsUni:        Uint64Range{Min: 64, Max: 160},
		MaxUDPPayloadSize:    Uint64Range{Min: 1400, Max: 1500},
		MaxDatagramFrameSize: Uint64Range{Min: 32768, Max: 65536},
	}
}

// ChromeWideTPRandomProfile 放宽区间以获得更强的离散度，仍保证 max_idle_timeout
// 不低于 30s（大于 Agent 25s 的 KeepAlivePeriod）、max_udp_payload_size 不低于 1200。
func ChromeWideTPRandomProfile() *TPRandomProfile {
	return &TPRandomProfile{
		MaxIdleTimeout:       DurationRange{Min: 30 * time.Second, Max: 120 * time.Second},
		InitialMaxData:       Uint64Range{Min: 8 << 20, Max: 28 << 20},
		InitialMaxStreamData: Uint64Range{Min: 3 << 20, Max: 12 << 20},
		MaxStreamsBidi:       Uint64Range{Min: 48, Max: 200},
		MaxStreamsUni:        Uint64Range{Min: 48, Max: 200},
		MaxUDPPayloadSize:    Uint64Range{Min: 1200, Max: 1500},
		MaxDatagramFrameSize: Uint64Range{Min: 16384, Max: 131072},
	}
}

// Sample 每次握手调用一次，使用 crypto/rand 独立采样全部参数。
func (p *TPRandomProfile) Sample() ClientTransportParams {
	if p == nil {
		return ClientTransportParams{}
	}
	return ClientTransportParams{
		MaxIdleTimeout:                 randDuration(p.MaxIdleTimeout),
		InitialMaxData:                 randUint64(p.InitialMaxData),
		InitialMaxStreamDataBidiLocal:  randUint64(p.InitialMaxStreamData),
		InitialMaxStreamDataBidiRemote: randUint64(p.InitialMaxStreamData),
		InitialMaxStreamDataUni:        randUint64(p.InitialMaxStreamData),
		MaxStreamsBidi:                 randUint64(p.MaxStreamsBidi),
		MaxStreamsUni:                  randUint64(p.MaxStreamsUni),
		MaxUDPPayloadSize:              randUint64(p.MaxUDPPayloadSize),
		MaxDatagramFrameSize:           randUint64(p.MaxDatagramFrameSize),
		GreaseLength:                   uint16(randUint64(Uint64Range{Min: 0, Max: 15})),
	}
}

func randUint64(r Uint64Range) uint64 {
	if r.Max <= r.Min {
		return r.Min
	}
	span := new(big.Int).SetUint64(r.Max - r.Min + 1)
	v, err := rand.Int(rand.Reader, span)
	if err != nil {
		// 采样失败时退化为区间下界，不能因为随机化阻断握手。
		return r.Min
	}
	return r.Min + v.Uint64()
}

func randDuration(r DurationRange) time.Duration {
	if r.Max <= r.Min {
		return r.Min
	}
	d := time.Duration(randUint64(Uint64Range{Min: uint64(r.Min), Max: uint64(r.Max)}))
	return d.Round(time.Millisecond)
}

// applyClientTP 把采样值写入一份新的 utls TransportParameter 列表。
// 只替换列表里已存在的参数；采样值为 0 表示该字段未启用，保留原值。
func applyClientTP(src tls.TransportParameters, v ClientTransportParams) tls.TransportParameters {
	out := make(tls.TransportParameters, len(src))
	for i, param := range src {
		switch param.(type) {
		case tls.MaxIdleTimeout:
			if v.MaxIdleTimeout > 0 {
				out[i] = tls.MaxIdleTimeout(uint64(v.MaxIdleTimeout / time.Millisecond))
				continue
			}
		case tls.InitialMaxData:
			if v.InitialMaxData > 0 {
				out[i] = tls.InitialMaxData(v.InitialMaxData)
				continue
			}
		case tls.InitialMaxStreamDataBidiLocal:
			if v.InitialMaxStreamDataBidiLocal > 0 {
				out[i] = tls.InitialMaxStreamDataBidiLocal(v.InitialMaxStreamDataBidiLocal)
				continue
			}
		case tls.InitialMaxStreamDataBidiRemote:
			if v.InitialMaxStreamDataBidiRemote > 0 {
				out[i] = tls.InitialMaxStreamDataBidiRemote(v.InitialMaxStreamDataBidiRemote)
				continue
			}
		case tls.InitialMaxStreamDataUni:
			if v.InitialMaxStreamDataUni > 0 {
				out[i] = tls.InitialMaxStreamDataUni(v.InitialMaxStreamDataUni)
				continue
			}
		case tls.InitialMaxStreamsBidi:
			if v.MaxStreamsBidi > 0 {
				out[i] = tls.InitialMaxStreamsBidi(v.MaxStreamsBidi)
				continue
			}
		case tls.InitialMaxStreamsUni:
			if v.MaxStreamsUni > 0 {
				out[i] = tls.InitialMaxStreamsUni(v.MaxStreamsUni)
				continue
			}
		case tls.MaxUDPPayloadSize:
			if v.MaxUDPPayloadSize > 0 {
				out[i] = tls.MaxUDPPayloadSize(v.MaxUDPPayloadSize)
				continue
			}
		case tls.MaxDatagramFrameSize:
			if v.MaxDatagramFrameSize > 0 {
				out[i] = tls.MaxDatagramFrameSize(v.MaxDatagramFrameSize)
				continue
			}
		case *tls.GREASETransportParameter:
			// 预设中的 GREASE 值首次使用后会被缓存，必须每连接重建。
			out[i] = &tls.GREASETransportParameter{Length: v.GreaseLength}
			continue
		}
		out[i] = param
	}
	return out
}

// cloneClientHelloSpecWithQTP 复制 spec 与 Extensions 切片，仅替换 QTP 扩展，
// 使 TP 字节按连接独立，其余扩展仍复用预设。
func cloneClientHelloSpecWithQTP(spec *tls.ClientHelloSpec, qtp *tls.QUICTransportParametersExtension) *tls.ClientHelloSpec {
	clone := *spec
	clone.Extensions = make([]tls.TLSExtension, len(spec.Extensions))
	copy(clone.Extensions, spec.Extensions)
	for i, ext := range clone.Extensions {
		if _, ok := ext.(*tls.QUICTransportParametersExtension); ok {
			clone.Extensions[i] = qtp
			break
		}
	}
	return &clone
}
