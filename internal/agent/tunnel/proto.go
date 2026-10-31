package tunnel

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	HeaderVersion = 1
	HeaderLen     = 8

	headerMagic0 = 0xA0
	headerMagic1 = 0xE1
)

// Header 是每条 QUIC Stream 的首部，出口节点据此定位规则与目标地址。
// Network 记录入口侧实际使用的传输协议，同端口 TCP+UDP 规则靠它决定 Dial 类型。
type Header struct {
	Version byte
	RuleID  uint32
	Network string
}

var ErrBadHeader = errors.New("tunnel: invalid stream header")

// headerNetworkByte 把协议名编码进首部保留字节，0 保持兼容旧客户端的 TCP 语义。
func headerNetworkByte(network string) byte {
	if network == "udp" {
		return 1
	}
	return 0
}

func WriteHeader(w io.Writer, h Header) error {
	buf := [HeaderLen]byte{
		headerMagic0, headerMagic1,
		h.Version, headerNetworkByte(h.Network),
		byte(h.RuleID >> 24), byte(h.RuleID >> 16), byte(h.RuleID >> 8), byte(h.RuleID),
	}
	_, err := w.Write(buf[:])
	return err
}

func ReadHeader(r io.Reader) (Header, error) {
	var buf [HeaderLen]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, err
	}
	if buf[0] != headerMagic0 || buf[1] != headerMagic1 {
		return Header{}, ErrBadHeader
	}
	if buf[2] != HeaderVersion {
		return Header{}, fmt.Errorf("tunnel: unsupported header version %d", buf[2])
	}
	network := "tcp"
	switch buf[3] {
	case 0:
	case 1:
		network = "udp"
	default:
		return Header{}, ErrBadHeader
	}
	return Header{
		Version: buf[2],
		RuleID:  binary.BigEndian.Uint32(buf[4:]),
		Network: network,
	}, nil
}
