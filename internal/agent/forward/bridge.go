package forward

import (
	"io"
	"net"
	"sync"

	"aoe-tunnel/internal/agent/stat"
	"aoe-tunnel/internal/agent/tunnel"
)

const bridgeBufferSize = 32 << 10

// Bridge 双向转发 TCP 与隧道 Stream。任一方向结束时关闭两端，防止 fd 泄漏。
// up 为 TCP→Stream 字节数，down 为 Stream→TCP 字节数；limits 为 nil 表示不限速。
func Bridge(tc net.Conn, st tunnel.DataStream, limits *stat.Limiters) (up, down int64) {
	var wg sync.WaitGroup
	var once sync.Once
	finish := func() {
		_ = tc.Close()
		_ = st.Close()
		st.CancelRead()
	}

	upSrc := io.Reader(tc)
	downSrc := io.Reader(st)
	if limits != nil {
		upSrc = limits.WrapUp(tc)
		downSrc = limits.WrapDown(st)
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		up, _ = io.CopyBuffer(st, upSrc, make([]byte, bridgeBufferSize))
		once.Do(finish)
	}()
	go func() {
		defer wg.Done()
		down, _ = io.CopyBuffer(tc, downSrc, make([]byte, bridgeBufferSize))
		once.Do(finish)
	}()
	wg.Wait()
	once.Do(finish)
	return up, down
}

// BridgeStream 双向桥接两条隧道 Stream，用于中转节点逐跳嵌套转发。
func BridgeStream(a, b tunnel.DataStream) (ab, ba int64) {
	var wg sync.WaitGroup
	var once sync.Once
	finish := func() {
		_ = a.Close()
		_ = b.Close()
		a.CancelRead()
		b.CancelRead()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		ab, _ = io.CopyBuffer(b, a, make([]byte, bridgeBufferSize))
		once.Do(finish)
	}()
	go func() {
		defer wg.Done()
		ba, _ = io.CopyBuffer(a, b, make([]byte, bridgeBufferSize))
		once.Do(finish)
	}()
	wg.Wait()
	once.Do(finish)
	return ab, ba
}
