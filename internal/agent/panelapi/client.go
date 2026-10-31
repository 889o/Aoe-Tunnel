// Package panelapi 是 Agent 访问面板的唯一入口：
// 业务调用统一用内置账号登录换取 Bearer JWT，节点令牌只用于上线校验。
package panelapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"aoe-tunnel/pkg/signal"
)

type Config struct {
	BaseURL   string
	Username  string
	Password  string
	NodeToken string
	Timeout   time.Duration
}

type Client struct {
	baseURL   string
	username  string
	password  string
	nodeToken string
	http      *http.Client
	log       *zap.Logger

	mu     sync.Mutex
	token  string
	expiry time.Time
}

func New(cfg Config, log *zap.Logger) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		baseURL:   strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		username:  cfg.Username,
		password:  cfg.Password,
		nodeToken: cfg.NodeToken,
		http:      &http.Client{Timeout: timeout},
		log:       log,
	}
}

func (c *Client) Enabled() bool { return c.baseURL != "" }

// OnlineInfo 是上线时自报的版本、外网 IP 与数据面监听地址；IP 由 curl 查询，面板据此拼出可达地址。
type OnlineInfo struct {
	Version  string `json:"version,omitempty"`
	IP       string `json:"ip,omitempty"`
	QUICAddr string `json:"quic_addr,omitempty"`
	TLSAddr  string `json:"tls_addr,omitempty"`
	// InstanceID 标识这台服务器上的部署实例；入口节点靠它判断是否被新部署抢占。
	InstanceID string `json:"instance_id,omitempty"`
}

// Online 只用节点令牌调用上线接口，令牌与节点编码绑定；心跳重复调用可刷新外网 IP。
func (c *Client) Online(ctx context.Context, nodeCode string, info OnlineInfo) error {
	if !c.Enabled() {
		return fmt.Errorf("panelapi: PANEL_API_URL not configured")
	}
	raw, err := json.Marshal(info)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/nodes/"+nodeCode+"/online", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Token", c.nodeToken)
	return c.execute(req, nil, false)
}

// Report 用 JWT 上报业务信令；401 时自动重新登录并重试一次。
func (c *Client) Report(ctx context.Context, nodeCode string, env signal.Envelope) error {
	if !c.Enabled() {
		return fmt.Errorf("panelapi: PANEL_API_URL not configured")
	}
	raw, err := env.Marshal()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/nodes/"+nodeCode+"/report", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.execute(req, nil, true)
}

// Nodes 拉取节点目录，供链式路由解析地址。
func (c *Client) Nodes(ctx context.Context, out any) error {
	if !c.Enabled() {
		return fmt.Errorf("panelapi: PANEL_API_URL not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/nodes", nil)
	if err != nil {
		return err
	}
	return c.execute(req, out, true)
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) execute(req *http.Request, out any, auth bool) error {
	if auth {
		header, err := c.authorization(req.Context())
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", header)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusUnauthorized && auth {
		c.invalidate()
		header, err := c.authorization(req.Context())
		if err != nil {
			return err
		}
		retry := req.Clone(req.Context())
		retry.Header.Set("Authorization", header)
		resp2, err := c.http.Do(retry)
		if err != nil {
			return err
		}
		defer resp2.Body.Close()
		return c.decode(resp2, out)
	}
	return c.decode(resp, out)
}

func (c *Client) decode(resp *http.Response, out any) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("panelapi: http status %d", resp.StatusCode)
	}
	var body apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("panelapi: decode: %w", err)
	}
	if body.Code != 0 {
		return fmt.Errorf("panelapi: panel code %d: %s", body.Code, body.Message)
	}
	if out != nil && len(body.Data) > 0 {
		if err := json.Unmarshal(body.Data, out); err != nil {
			return fmt.Errorf("panelapi: decode data: %w", err)
		}
	}
	return nil
}

func (c *Client) invalidate() {
	c.mu.Lock()
	c.token = ""
	c.expiry = time.Time{}
	c.mu.Unlock()
}

// authorization 返回可用 access token，过期前 60s 主动续签。
func (c *Client) authorization(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.expiry.Add(-60*time.Second)) {
		tok := c.token
		c.mu.Unlock()
		return "Bearer " + tok, nil
	}
	c.mu.Unlock()

	if err := c.login(ctx); err != nil {
		return "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" {
		return "", fmt.Errorf("panelapi: login returned empty token")
	}
	return "Bearer " + c.token, nil
}

type loginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginOutput struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (c *Client) login(ctx context.Context) error {
	raw, err := json.Marshal(loginInput{Username: c.username, Password: c.password})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/auth/login", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("panelapi: login request: %w", err)
	}
	defer resp.Body.Close()

	var body apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("panelapi: login decode: %w", err)
	}
	if resp.StatusCode != http.StatusOK || body.Code != 0 {
		return fmt.Errorf("panelapi: login failed status=%d code=%d msg=%s", resp.StatusCode, body.Code, body.Message)
	}
	var out loginOutput
	if err := json.Unmarshal(body.Data, &out); err != nil {
		return fmt.Errorf("panelapi: login data: %w", err)
	}
	if out.AccessToken == "" {
		return fmt.Errorf("panelapi: login returned empty token")
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = 900
	}

	c.mu.Lock()
	c.token = out.AccessToken
	c.expiry = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	c.mu.Unlock()
	c.log.Info("panel login ok", zap.String("user", c.username), zap.Int64("expires_in", out.ExpiresIn))
	return nil
}
