package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr   string
	MySQLDSN   string
	MQTTBroker string
	MQTTCACert string
	MQTTSNI    string
	MQTTUser   string
	MQTTPass   string
	LogLevel   string
	LogFormat  string

	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	AdminUser  string
	AdminPass  string

	// AgentUser/AgentPass 是内置 Agent 账号，供 Agent 登录换取 Bearer JWT。
	AgentUser string
	AgentPass string

	// NodeSecret 用于派生节点令牌；缺省由 MYSQL_DSN 派生，保证重启后历史令牌仍有效。
	NodeSecret string
	// AgentDownloadURL 是发布目录前缀，安装脚本据此拼接 aoe-agent-linux-<arch> 与 aoe-agent-certs.tar.gz。
	AgentDownloadURL string
	// PublicURL 是站点公网地址（如 https://panel.example.com）；缺省时安装脚本按请求头自动拼接。
	PublicURL string
	// WebDir 指向前端构建产物目录；配置后面板同时托管页面，未配置则只提供 API。
	WebDir string
	// AgentBroker 是下发给安装脚本的 MQTT 地址（需远端可达）；缺省与面板自身 MQTT_BROKER 相同。
	AgentBroker string
	// HTTPSAddr 与 TLSCert/TLSKey 开启 HTTPS：配置证书后面板在 HTTPSAddr 提供 TLS，
	// HTTPAddr 只做跳转，避免浏览器在非安全上下文禁用剪贴板等能力。
	HTTPSAddr string
	TLSCert   string
	TLSKey    string
	// AgentDistDir 是 Agent 二进制与证书包目录，面板通过 /downloads 对外提供。
	AgentDistDir string
}

func Load() (*Config, error) {
	accessTTL, err := time.ParseDuration(getEnv("PANEL_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("PANEL_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getEnv("PANEL_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("PANEL_REFRESH_TTL: %w", err)
	}
	c := &Config{
		HTTPAddr:   getEnv("PANEL_HTTP_ADDR", ":8080"),
		MySQLDSN:   os.Getenv("MYSQL_DSN"),
		MQTTBroker: getEnv("MQTT_BROKER", "ssl://127.0.0.1:8883"),
		MQTTCACert: os.Getenv("MQTT_CA"),
		MQTTSNI:    getEnv("MQTT_SNI", ""),
		MQTTUser:   os.Getenv("MQTT_USER"),
		MQTTPass:   os.Getenv("MQTT_PASS"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		LogFormat:  getEnv("LOG_FORMAT", "console"),
		// 未显式配置时由 MYSQL_DSN 派生，保证重启后 token 不失效，同时不新增必填项。
		JWTSecret:  os.Getenv("PANEL_JWT_SECRET"),
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
		AdminUser:  getEnv("PANEL_ADMIN_USER", "admin"),
		AdminPass:  getEnv("PANEL_ADMIN_PASS", "admin123"),
		AgentUser:  getEnv("PANEL_AGENT_USER", "agent"),
		AgentPass:  getEnv("PANEL_AGENT_PASS", "agent123"),

		NodeSecret:       os.Getenv("PANEL_NODE_SECRET"),
		AgentDownloadURL: os.Getenv("PANEL_AGENT_DOWNLOAD_URL"),
		PublicURL:        publicBaseURL(),
		WebDir:           strings.TrimSpace(os.Getenv("PANEL_WEB_DIR")),
		AgentBroker:      os.Getenv("PANEL_AGENT_BROKER"),
		HTTPSAddr:        getEnv("PANEL_HTTPS_ADDR", ":443"),
		TLSCert:          strings.TrimSpace(os.Getenv("PANEL_TLS_CERT")),
		TLSKey:           strings.TrimSpace(os.Getenv("PANEL_TLS_KEY")),
		AgentDistDir:     os.Getenv("PANEL_AGENT_DIST_DIR"),
	}
	if c.AgentBroker == "" {
		c.AgentBroker = c.MQTTBroker
	}

	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("PANEL_PUBLIC_URL 需为 http(s)://host 形式")
		}
	}

	if accessTTL <= 0 || refreshTTL <= accessTTL {
		return nil, fmt.Errorf("PANEL_ACCESS_TTL/PANEL_REFRESH_TTL invalid")
	}
	if c.MySQLDSN == "" {
		return nil, fmt.Errorf("MYSQL_DSN is required")
	}
	if c.MQTTBroker == "" {
		return nil, fmt.Errorf("MQTT_BROKER is required")
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return nil, fmt.Errorf("PANEL_TLS_CERT 与 PANEL_TLS_KEY 需同时配置")
	}
	return c, nil
}

// publicBaseURL 兼容两种写法：PANEL_SITE_BASE_URL 为主控安装脚本预置项，PANEL_PUBLIC_URL 为旧变量。
func publicBaseURL() string {
	for _, key := range []string{"PANEL_SITE_BASE_URL", "PANEL_PUBLIC_URL"} {
		if v := strings.TrimSuffix(strings.TrimSpace(os.Getenv(key)), "/"); v != "" {
			return v
		}
	}
	return ""
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
