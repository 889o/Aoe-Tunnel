package model

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	DisplayName  string `gorm:"size:64" json:"display_name"`
	// Role 是早期版本的字符串角色列，仅保留用于存量数据回填，鉴权一律走 RoleID。
	Role   string `gorm:"size:32;not null;default:admin" json:"-"`
	RoleID uint   `gorm:"not null;default:0;index" json:"role_id"`
	// PackageID 是用户绑定的套餐；创建端口规则时自动使用该套餐，无需手动选择。
	PackageID *uint `gorm:"column:package_id;index" json:"package_id"`
	// PackagePeriod 是该用户购买时选择的计费周期，用于钱包商城区分“已购买/续费”。
	PackagePeriod string `gorm:"column:package_period;size:16;not null;default:''" json:"package_period"`
	// InviteCode 是邀请码，注册分销使用；存量用户首次查看邀请信息时补发。
	InviteCode string `gorm:"column:invite_code;size:16;index" json:"invite_code"`
	// InviterID 是上级用户，仅开启分销且注册携带有效邀请码时写入。
	InviterID *uint `gorm:"column:inviter_id;index" json:"-"`
	// Balance 是用户余额，单位元；注册用户默认为 0。
	Balance     float64    `gorm:"column:balance;type:decimal(14,2);not null;default:0" json:"balance"`
	Status      string     `gorm:"size:16;not null;default:active;index" json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserStatusPending 表示注册后等待管理员审核，审核通过前无法登录。
const UserStatusPending = "pending"

// SiteConfig 是站点运行配置，采用单行持久化（ID 固定为 1），避免多 key 值表的序列化开销。
type SiteConfig struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	SiteTitle    string `gorm:"column:site_title;size:128;not null" json:"site_title"`
	SiteSubtitle string `gorm:"column:site_subtitle;size:255;not null" json:"site_subtitle"`
	// Announcement 是站点公告内容，为空表示不展示公告。
	Announcement      string    `gorm:"column:announcement;size:512;not null;default:''" json:"announcement"`
	RegisterOpen      bool      `gorm:"column:register_open;not null;default:false" json:"register_open"`
	RegisterReview    bool      `gorm:"column:register_review;not null;default:true" json:"register_review"`
	DistributionOpen  bool      `gorm:"column:distribution_open;not null;default:false" json:"distribution_open"`
	DistributionRatio float64   `gorm:"column:distribution_ratio;type:decimal(6,2);not null;default:0" json:"distribution_ratio"`
	// PublicURL 是站点公网地址，是远端 Agent 访问控制面的唯一入口；为空时回退环境变量/请求头。
	PublicURL string `gorm:"column:public_url;size:255;not null;default:''" json:"public_url"`
	// AgentDownloadURL 是 Agent 二进制与证书包的公网下载目录；为空时回退环境变量。
	AgentDownloadURL string `gorm:"column:agent_download_url;size:255;not null;default:''" json:"agent_download_url"`
	// AgentBroker 是下发给 Agent 的 MQTT 接入地址（远端可达）；为空时回退环境变量。
	AgentBroker string `gorm:"column:agent_broker;size:255;not null;default:''" json:"agent_broker"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Role 的权限以权限码数组持久化，避免额外关联表。
type Role struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Code            string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name            string    `gorm:"size:64;not null" json:"name"`
	Description     string    `gorm:"size:255" json:"description"`
	Builtin         bool      `gorm:"not null;default:false" json:"builtin"`
	PermissionCodes []string  `gorm:"column:permission_codes;serializer:json;type:json" json:"permission_codes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Permission struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Code  string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name  string `gorm:"size:64;not null" json:"name"`
	Group string `gorm:"size:32;not null;index" json:"group"`
	Sort  int    `gorm:"not null;default:0" json:"sort"`
}

// Session 一条记录对应一次登录，refresh 轮换只更新 RefreshHash。
type Session struct {
	ID          uint       `gorm:"primaryKey" json:"-"`
	SID         string     `gorm:"column:sid;size:64;uniqueIndex;not null" json:"sid"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	RefreshHash string     `gorm:"size:64;not null" json:"-"`
	UA          string     `gorm:"size:255" json:"ua"`
	IP          string     `gorm:"size:64" json:"ip"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// 节点类型：0 为存量未标记（启动回填为入口），1 入口 Ingress，2 出口 Egress。
const (
	NodeTypeLegacy  = 0
	NodeTypeIngress = 1
	NodeTypeEgress  = 2
)

type Node struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Code     string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name     string `gorm:"size:128;not null" json:"name"`
	QUICAddr string `gorm:"column:quic_addr;size:128;not null" json:"quic_addr"`
	TLSAddr  string `gorm:"size:128;not null" json:"tls_addr"`
	// Type 仅作面板侧的角色标记（1 入口 / 2 出口），Agent 不感知，也不影响转发逻辑。
	Type int `gorm:"not null;default:0;index" json:"type"`
	// IP 是 Agent 上线时面板记录的外网地址，节点创建时无需填写。
	IP string `gorm:"column:ip;size:64;not null;default:''" json:"ip"`
	// PublicAddr 是创建节点时可选填写的对接地址，优先于 Agent 上报 IP 展示。
	PublicAddr string `gorm:"column:public_addr;size:255;not null;default:''" json:"public_addr"`
	// RateUpMBps/RateDownMBps 是入口节点级带宽上限，单位 MB/s，0 表示不限速。
	RateUpMBps   float64 `gorm:"column:rate_up_mbps;type:decimal(10,3);not null;default:0" json:"rate_up_mbps"`
	RateDownMBps float64 `gorm:"column:rate_down_mbps;type:decimal(10,3);not null;default:0" json:"rate_down_mbps"`
	// RatioUp/RatioDown 是入口节点级计费比，入口规则按该系数计费，规则侧不再单独配置。
	RatioUp   float64 `gorm:"column:ratio_up;type:decimal(6,3);not null;default:1" json:"ratio_up"`
	RatioDown float64 `gorm:"column:ratio_down;type:decimal(6,3);not null;default:1" json:"ratio_down"`
	// NodeTokenHash 只保存令牌的 sha256，明文可由 type/code/epoch 随时重算渲染安装脚本。
	NodeTokenHash  string     `gorm:"size:64;not null;default:''" json:"-"`
	NodeTokenEpoch int        `gorm:"not null;default:1" json:"node_token_epoch"`
	// ActiveInstance 仅入口节点使用：记录最后一个上线的实例，用于拒绝旧实例的晚到心跳。
	ActiveInstance string     `gorm:"column:active_instance;size:64;not null;default:''" json:"-"`
	Version        string     `gorm:"size:32" json:"version"`
	Status         string     `gorm:"size:32;not null;default:offline" json:"status"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// NodeInstance 记录同一节点在多台服务器上的部署实例：入口只保留最后一个活跃实例，
// 出口允许一个安装命令部署多台服务器，入口规则按在线实例轮询并故障转移。
type NodeInstance struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	NodeCode   string     `gorm:"column:node_code;size:64;not null;uniqueIndex:uk_node_instance;index" json:"node_code"`
	InstanceID string     `gorm:"column:instance_id;size:64;not null;uniqueIndex:uk_node_instance" json:"instance_id"`
	IP         string     `gorm:"column:ip;size:64;not null;default:''" json:"ip"`
	QUICAddr   string     `gorm:"column:quic_addr;size:128;not null;default:''" json:"quic_addr"`
	TLSAddr    string     `gorm:"column:tls_addr;size:128;not null;default:''" json:"tls_addr"`
	Version    string     `gorm:"size:32;not null;default:''" json:"version"`
	Online     bool       `gorm:"not null;default:false" json:"online"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// 规则方向：entry 入口规则负责监听与计费，exit 出口规则负责落地 Dial 目标地址。
const (
	RuleDirectionEntry = "entry"
	RuleDirectionExit  = "exit"
)

// PaymentChannel 是支付方式配置；内置渠道不可删除，仅允许修改开关与配置。
type PaymentChannel struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Code      string            `gorm:"size:32;uniqueIndex;not null" json:"code"`
	Name      string            `gorm:"size:64;not null" json:"name"`
	Enabled   bool              `gorm:"not null;default:false" json:"enabled"`
	Builtin   bool              `gorm:"not null;default:false" json:"builtin"`
	Config    map[string]string `gorm:"column:config;serializer:json;type:json" json:"config"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// PaymentOrder 是在线支付订单；Kind 区分购买套餐与余额充值，回调按类型分别绑定套餐或加余额。
type PaymentOrder struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	OutTradeNo string     `gorm:"column:out_trade_no;size:64;uniqueIndex;not null" json:"out_trade_no"`
	UserID     uint       `gorm:"column:user_id;not null;index" json:"user_id"`
	Kind       string     `gorm:"size:16;not null;default:package;index" json:"kind"`
	PackageID  uint       `gorm:"column:package_id;not null;index" json:"package_id"`
	Period     string     `gorm:"size:16;not null" json:"period"`
	Amount     float64    `gorm:"type:decimal(14,2);not null;default:0" json:"amount"`
	PayType    string     `gorm:"column:pay_type;size:16;not null" json:"pay_type"`
	Status     string     `gorm:"size:16;not null;default:pending;index" json:"status"`
	TradeNo    string     `gorm:"column:trade_no;size:64;not null;default:''" json:"trade_no"`
	PaidAt     *time.Time `gorm:"column:paid_at" json:"paid_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

const (
	PaymentOrderKindPackage  = "package"
	PaymentOrderKindRecharge = "recharge"
)

const (
	PaymentOrderPending = "pending"
	PaymentOrderPaid    = "paid"
	PaymentOrderFailed  = "failed"
)

// CommissionRecord 是分销佣金流水，按套餐购买金额 × 比例结算给上级。
type CommissionRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	InviterID  uint      `gorm:"column:inviter_id;index;not null" json:"inviter_id"`
	BuyerID    uint      `gorm:"column:buyer_id;index;not null" json:"buyer_id"`
	PackageID  uint      `gorm:"column:package_id" json:"package_id"`
	Amount     float64   `gorm:"column:amount;type:decimal(14,2);not null;default:0" json:"amount"`
	Ratio      float64   `gorm:"column:ratio;type:decimal(6,2);not null;default:0" json:"ratio"`
	Commission float64   `gorm:"column:commission;type:decimal(14,2);not null;default:0" json:"commission"`
	CreatedAt  time.Time `json:"created_at"`
}

// Package 是入口与出口节点的绑定单元：同一套餐的入口规则自动向套餐内所有出口规则扇出。
type Package struct {
	ID           uint     `gorm:"primaryKey" json:"id"`
	Name         string   `gorm:"size:128;not null" json:"name"`
	Description  string   `gorm:"size:255" json:"description"`
	IngressNodes []string `gorm:"column:ingress_nodes;serializer:json;type:json" json:"ingress_nodes"`
	EgressNodes  []string `gorm:"column:egress_nodes;serializer:json;type:json" json:"egress_nodes"`
	// TrafficLimitGB 是套餐总流量上限，单位 GB，0 表示不限；用量按套餐内入口规则计费流量汇总。
	TrafficLimitGB float64 `gorm:"column:traffic_limit_gb;type:decimal(14,3);not null;default:0" json:"traffic_limit_gb"`
	// RuleLimit 是可创建的入口规则数量上限，0 表示不限制。
	RuleLimit int `gorm:"column:rule_limit;not null;default:0" json:"rule_limit"`
	// Prices 按周期保存售价，key 为 duration 枚举；未配置的周期不可购买。
	Prices map[string]float64 `gorm:"column:prices;serializer:json;type:json" json:"prices"`
	// Price/Duration 是旧版单周期字段，仅用于存量数据兜底迁移，新写入不再使用。
	Price     float64   `gorm:"column:price;type:decimal(14,2);not null;default:0" json:"-"`
	Duration  string    `gorm:"column:duration;size:16;not null;default:monthly" json:"-"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 套餐周期枚举，与前端选项保持一致。
const (
	PackageDurationMonthly   = "monthly"
	PackageDurationQuarterly = "quarterly"
	PackageDurationHalfYear  = "half_year"
	PackageDurationYearly    = "yearly"
	PackageDurationPermanent = "permanent"
)

// PackageDurationOrder 是周期展示顺序。
var PackageDurationOrder = []string{
	PackageDurationMonthly, PackageDurationQuarterly, PackageDurationHalfYear,
	PackageDurationYearly, PackageDurationPermanent,
}

// NormalizePackagePrices 兼容旧版单周期数据：未配置 Prices 时回填旧 price/duration。
func (p *Package) NormalizePackagePrices() {
	if len(p.Prices) > 0 {
		return
	}
	duration := p.Duration
	if duration == "" {
		duration = PackageDurationMonthly
	}
	p.Prices = map[string]float64{duration: p.Price}
}

func ValidPackageDuration(v string) bool {
	switch v {
	case PackageDurationMonthly, PackageDurationQuarterly, PackageDurationHalfYear,
		PackageDurationYearly, PackageDurationPermanent:
		return true
	}
	return false
}

type PortRule struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Direction 决定规则形态：entry 有监听端口，exit 有目标地址。
	Direction string `gorm:"size:16;not null;default:entry;index" json:"direction"`
	PackageID uint   `gorm:"not null;default:0;index" json:"package_id"`
	Node      string `gorm:"size:64;not null;index" json:"node"`
	Name      string `gorm:"size:128;not null" json:"name"`
	Protocol  string `gorm:"size:8;not null;default:tcp" json:"protocol"`
	// Transport 是入口到出口的隧道传输选择，固定生效：quic 只走 QUIC，tls 只走 TLS1.3+yamux。
	Transport string `gorm:"size:8;not null;default:quic" json:"transport"`
	// ListenPort 仅入口规则使用；出口规则恒为 0。
	ListenPort int `gorm:"not null;default:0" json:"listen_port"`
	// TargetAddr 仅出口规则使用。
	TargetAddr string `gorm:"size:255;not null;default:''" json:"target_addr"`
	// EgressNode 是入口规则选定的出口节点；为空表示历史规则仍按套餐全部出口扇出。
	EgressNode string `gorm:"column:egress_node;size:64;not null;default:'';index" json:"egress_node"`
	// OwnerUserID 用于行级隔离：无 port:manage-all 权限的用户只能看到自己创建的规则。
	OwnerUserID *uint     `gorm:"index" json:"owner_user_id"`
	RatioUp     float64   `gorm:"type:decimal(6,3);not null;default:1" json:"ratio_up"`
	RatioDown   float64   `gorm:"type:decimal(6,3);not null;default:1" json:"ratio_down"`
	RateUpBps   int64     `gorm:"not null;default:0" json:"rate_up_bps"`
	RateDownBps int64     `gorm:"not null;default:0" json:"rate_down_bps"`
	QuotaBytes  int64     `gorm:"not null;default:0" json:"quota_bytes"`
	UsedBytes   int64     `gorm:"not null;default:0" json:"used_bytes"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	Status      string    `gorm:"size:32;not null;default:active" json:"status"`
	Revision    int       `gorm:"not null;default:1" json:"revision"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TrafficStat struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RuleID     uint      `gorm:"not null;index:idx_rule_time,priority:1" json:"rule_id"`
	NodeCode   string    `gorm:"size:64;not null" json:"node_code"`
	UpBytes    int64     `gorm:"not null" json:"up_bytes"`
	DownBytes  int64     `gorm:"not null" json:"down_bytes"`
	BillBytes  int64     `gorm:"not null" json:"bill_bytes"`
	RecordedAt time.Time `gorm:"not null;index:idx_rule_time,priority:2" json:"recorded_at"`
}
