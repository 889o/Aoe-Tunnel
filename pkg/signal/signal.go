package signal

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TopicPrefix  = "aoe/v1/node/"
	CmdSuffix    = "/cmd"
	ReportSuffix = "/report"
	StateSuffix  = "/state"

	ALPNQUIC     = "aoe-tunnel-v1"
	ALPNFallback = "aoe-tunnel-fallback-v1"
)

func CmdTopic(node string) string    { return TopicPrefix + node + CmdSuffix }
func ReportTopic(node string) string { return TopicPrefix + node + ReportSuffix }
func StateTopic(node string) string  { return TopicPrefix + node + StateSuffix }
func ReportFilter() string           { return TopicPrefix + "+" + ReportSuffix }
func StateFilter() string            { return TopicPrefix + "+" + StateSuffix }

// NodeFromTopic 从 aoe/v1/node/{code}/{suffix} 提取节点编码。
func NodeFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		return ""
	}
	return parts[3]
}

type Envelope struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	TS       int64           `json:"ts"`
	NodeCode string          `json:"node_code,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

func NewEnvelope(typ, node string, payload any) (Envelope, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Envelope{}, err
		}
		raw = b
	}
	return Envelope{
		ID:       uuid.NewString(),
		Type:     typ,
		TS:       time.Now().Unix(),
		NodeCode: node,
		Payload:  raw,
	}, nil
}

func (e Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

const (
	TypePortAdd         = "port.add"
	TypePortUpdate      = "port.update"
	TypePortRemove      = "port.remove"
	TypePortSync        = "port.sync"
	TypeNodeStatus      = "node.status"
	TypeNodeMetrics     = "node.metrics"
	TypeTrafficReport   = "traffic.report"
	TypePortEvent       = "port.event"
	TypeNodeSyncRequest = "node.sync.request"
	// TypeNodeKick 由面板下发给被抢占的入口实例：同一入口节点只允许最后一个部署的实例在线。
	TypeNodeKick = "node.kick"
)

// ExitTarget 是入口规则扇出到的一个出口规则及其数据面地址。
type ExitTarget struct {
	RuleID   uint   `json:"rule_id"`
	Node     string `json:"node"`
	QUICAddr string `json:"quic_addr"`
	TLSAddr  string `json:"tls_addr"`
}

// PortRule 同时承载入口规则与出口规则：入口使用 ListenPort 与 Exits，
// 出口使用 TargetAddr；Agent 以 RuleID 对应自身规则。
type PortRule struct {
	RuleID    uint   `json:"rule_id"`
	Revision  int    `json:"revision"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
	PackageID uint   `json:"package_id"`
	Node      string `json:"node"`
	Protocol  string `json:"protocol"`
	// Transport 为 quic 或 tls，入口 Agent 据此选择隧道链路。
	Transport   string       `json:"transport"`
	ListenPort  int          `json:"listen_port"`
	TargetAddr  string       `json:"target_addr"`
	Exits       []ExitTarget `json:"exits,omitempty"`
	RatioUp     float64      `json:"ratio_up"`
	RatioDown   float64      `json:"ratio_down"`
	RateUpBps   int64        `json:"rate_up_bps"`
	RateDownBps int64        `json:"rate_down_bps"`
	QuotaBytes  int64        `json:"quota_bytes"`
	UsedBytes   int64        `json:"used_bytes"`
	Enabled     bool         `json:"enabled"`
}

type PortSync struct {
	Rules []PortRule `json:"rules"`
}

type NodeStatus struct {
	Online      bool   `json:"online"`
	Version     string `json:"version"`
	UptimeSec   int64  `json:"uptime_sec"`
	TunnelMode  string `json:"tunnel_mode"`
	ActiveConns int64  `json:"active_conns"`
}

// SystemMetrics 是 Agent 每秒采集的主机资源快照，面板只做内存缓存，不落库。
type SystemMetrics struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemUsed     uint64  `json:"mem_used"`
	MemTotal    uint64  `json:"mem_total"`
	MemPercent  float64 `json:"mem_percent"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskPercent float64 `json:"disk_percent"`
	Load1       float64 `json:"load1"`
	Load5       float64 `json:"load5"`
	Load15      float64 `json:"load15"`
	UptimeSec   int64   `json:"uptime_sec"`
	NetInBps    int64   `json:"net_in_bps"`
	NetOutBps   int64   `json:"net_out_bps"`
	TS          int64   `json:"ts"`
}

type TrafficReport struct {
	IntervalSec int           `json:"interval_sec"`
	Items       []TrafficItem `json:"items"`
}

type TrafficItem struct {
	RuleID    uint  `json:"rule_id"`
	UpBytes   int64 `json:"up_bytes"`
	DownBytes int64 `json:"down_bytes"`
	BillBytes int64 `json:"bill_bytes"`
	UsedBytes int64 `json:"used_bytes"`
}

type PortEvent struct {
	RuleID uint   `json:"rule_id"`
	Event  string `json:"event"`
	Detail string `json:"detail"`
}

type StatePayload struct {
	Online     bool   `json:"online"`
	Version    string `json:"version"`
	TS         int64  `json:"ts"`
	InstanceID string `json:"instance_id,omitempty"`
}

// NodeKick 只携带目标实例；Agent 收到与自身实例 ID 匹配的指令后主动退出，让新实例独占节点。
type NodeKick struct {
	InstanceID string `json:"instance_id"`
}
