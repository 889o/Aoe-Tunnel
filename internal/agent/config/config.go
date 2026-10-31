package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	NodeCode        string
	MQTTBroker      string
	MQTTCACert      string
	MQTTSNI         string
	MQTTUser        string
	MQTTPass        string
	QUICAddr        string
	TLSAddr         string
	ListenHost      string
	CertFile        string
	KeyFile         string
	ClientCertFile  string
	ClientKeyFile   string
	CAFile          string
	SNI             string
	DataDir         string
	ForceFallback   bool
	ReportInterval  time.Duration
	MetricsInterval time.Duration
	PanelAPIURL     string
	PanelUser       string
	PanelPass       string
	NodeToken       string
	// PublicIPURL 是 curl 查询外网 IP 的地址，多机部署时面板依赖该地址生成可达数据面地址。
	PublicIPURL string
	LogLevel    string
	LogFormat   string
}

func Load() (*Config, error) {
	interval, err := time.ParseDuration(getEnv("AGENT_REPORT_INTERVAL", "2s"))
	if err != nil {
		return nil, fmt.Errorf("invalid AGENT_REPORT_INTERVAL: %w", err)
	}
	forceFallback, err := strconv.ParseBool(getEnv("AGENT_FORCE_FALLBACK", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid AGENT_FORCE_FALLBACK: %w", err)
	}
	metricsInterval, err := time.ParseDuration(getEnv("AGENT_METRICS_INTERVAL", "1s"))
	if err != nil {
		return nil, fmt.Errorf("invalid AGENT_METRICS_INTERVAL: %w", err)
	}

	c := &Config{
		NodeCode:        os.Getenv("AGENT_NODE_CODE"),
		MQTTBroker:      os.Getenv("MQTT_BROKER"),
		MQTTCACert:      os.Getenv("MQTT_CA"),
		MQTTSNI:         getEnv("MQTT_SNI", ""),
		MQTTUser:        os.Getenv("MQTT_USER"),
		MQTTPass:        os.Getenv("MQTT_PASS"),
		QUICAddr:        getEnv("AGENT_QUIC_ADDR", ":9613"),
		TLSAddr:         getEnv("AGENT_TLS_ADDR", ":9614"),
		ListenHost:      getEnv("AGENT_LISTEN_HOST", "0.0.0.0"),
		CertFile:        getEnv("AGENT_CERT_FILE", "deploy/certs/server.crt"),
		KeyFile:         getEnv("AGENT_KEY_FILE", "deploy/certs/server.key"),
		ClientCertFile:  getEnv("AGENT_CLIENT_CERT_FILE", "deploy/certs/client.crt"),
		ClientKeyFile:   getEnv("AGENT_CLIENT_KEY_FILE", "deploy/certs/client.key"),
		CAFile:          getEnv("AGENT_CA_FILE", "deploy/certs/ca.crt"),
		SNI:             getEnv("AGENT_SNI", "aoe-node"),
		DataDir:         getEnv("AGENT_DATA_DIR", "data"),
		ForceFallback:   forceFallback,
		ReportInterval:  interval,
		MetricsInterval: metricsInterval,
		PanelAPIURL:     getEnv("PANEL_API_URL", ""),
		PanelUser:       getEnv("PANEL_AGENT_USER", "agent"),
		PanelPass:       getEnv("PANEL_AGENT_PASS", "agent123"),
		NodeToken:       os.Getenv("AGENT_NODE_TOKEN"),
		PublicIPURL:     getEnv("AGENT_PUBLIC_IP_URL", "https://ipinfo.io/ip"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "console"),
	}

	// 控制面地址必须显式下发，避免 Agent 误连本机；一键安装脚本会写入这两个值。
	if c.PanelAPIURL == "" {
		return nil, fmt.Errorf("PANEL_API_URL is required")
	}
	if c.MQTTBroker == "" {
		return nil, fmt.Errorf("MQTT_BROKER is required")
	}
	return c, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
