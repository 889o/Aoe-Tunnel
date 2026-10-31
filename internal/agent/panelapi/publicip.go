package panelapi

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// publicIPCacheTTL 让心跳复用同一个外网 IP，避免每 15s 拉起一次 curl。
const publicIPCacheTTL = 5 * time.Minute

const defaultPublicIPURL = "https://ipinfo.io/ip"

// PublicIPResolver 通过 curl 查询外网 IP，失败时回落到上一次成功结果。
// 出口/入口多机部署时依赖该地址生成 Agent 的可达数据面地址。
type PublicIPResolver struct {
	url string

	mu        sync.Mutex
	value     string
	fetchedAt time.Time
}

func NewPublicIPResolver(url string) *PublicIPResolver {
	url = strings.TrimSpace(url)
	if url == "" {
		url = defaultPublicIPURL
	}
	return &PublicIPResolver{url: url}
}

// Get 返回可用的外网 IP；查询失败且无历史结果时返回空串，面板会回落到请求来源 IP。
func (r *PublicIPResolver) Get(ctx context.Context) string {
	r.mu.Lock()
	if r.value != "" && time.Since(r.fetchedAt) < publicIPCacheTTL {
		v := r.value
		r.mu.Unlock()
		return v
	}
	r.mu.Unlock()

	if v := r.fetch(ctx); v != "" {
		r.mu.Lock()
		r.value = v
		r.fetchedAt = time.Now()
		r.mu.Unlock()
		return v
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.value
}

func (r *PublicIPResolver) fetch(ctx context.Context) string {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(cctx, "curl", "-fsS", "--max-time", "5", r.url).Output()
	if err != nil {
		return ""
	}
	ip := strings.TrimSpace(string(out))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
