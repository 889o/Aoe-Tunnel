package stat

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// limitBurst 保证单次读取能拿到一个完整缓冲块，避免 WaitN 因超过 burst 直接失败。
const limitBurst = 64 << 10

// Limiters 是单个端口的上/下行独立令牌桶。
type Limiters struct {
	up   *rate.Limiter
	down *rate.Limiter
}

func NewLimiters(upBps, downBps int64) *Limiters {
	if upBps <= 0 && downBps <= 0 {
		return nil
	}
	l := &Limiters{}
	if upBps > 0 {
		l.up = newByteLimiter(upBps)
	}
	if downBps > 0 {
		l.down = newByteLimiter(downBps)
	}
	return l
}

func newByteLimiter(bps int64) *rate.Limiter {
	burst := int(bps)
	if burst < limitBurst {
		burst = limitBurst
	}
	return rate.NewLimiter(rate.Limit(bps), burst)
}

func (l *Limiters) WrapUp(r io.Reader) io.Reader {
	return wrap(r, l.up)
}

func (l *Limiters) WrapDown(r io.Reader) io.Reader {
	return wrap(r, l.down)
}

func wrap(r io.Reader, lim *rate.Limiter) io.Reader {
	if lim == nil {
		return r
	}
	return &throttledReader{r: r, lim: lim}
}

type throttledReader struct {
	r   io.Reader
	lim *rate.Limiter
}

func (t *throttledReader) Read(p []byte) (int, error) {
	if max := int(t.lim.Burst()); max > 0 && len(p) > max {
		p = p[:max]
	}
	n, err := t.r.Read(p)
	if n > 0 {
		// 读到的数据先占令牌再交给下游，保证限速在真实出口生效。
		if werr := t.lim.WaitN(context.Background(), n); werr != nil {
			return n, err
		}
	}
	return n, err
}
