package tunnel

import (
	"io"
	"time"

	"github.com/hashicorp/yamux"
	quic "github.com/refraction-networking/uquic"
)

// DataStream 抹平 QUIC Stream 与 yamux Stream 的差异，桥接层不需要知道底层传输。
type DataStream interface {
	io.Reader
	io.Writer
	io.Closer
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	CancelRead()
}

type quicStream struct{ quic.Stream }

// CancelRead 关闭 QUIC 读方向；Close 只关闭写方向，必须显式取消读才能唤醒阻塞的 Read。
func (s quicStream) CancelRead() { s.Stream.CancelRead(0) }

func WrapQUICStream(st quic.Stream) DataStream { return quicStream{st} }

type yamuxStream struct{ *yamux.Stream }

// yamux 的 Close 会同时终止双向，CancelRead 无需额外动作。
func (yamuxStream) CancelRead() {}

func WrapYamuxStream(st *yamux.Stream) DataStream { return yamuxStream{st} }
