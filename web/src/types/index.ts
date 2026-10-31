export interface ApiResult<T> {
  code: number
  message: string
  data: T
}

export interface Node {
  id: number
  code: string
  name: string
  // 外网 IP 由 Agent 上线时面板自动记录
  ip: string
  // 入口节点可选填写的对接地址（自定义内容）；留空时端口规则页不展示对接地址
  public_addr: string
  // 数据面监听地址由 Agent 自报，仅后端用于拼装可达地址，前端不展示
  quic_addr: string
  tls_addr: string
  // 1 入口 Ingress / 2 出口 Egress，仅面板侧角色标记
  type: number
  // 入口节点级带宽上限，单位 MB/s，0 表示不限速
  rate_up_mbps: number
  rate_down_mbps: number
  // 入口节点级计费比，规则侧不再单独配置
  ratio_up: number
  ratio_down: number
  node_token_epoch: number
  version: string
  status: string
  last_seen_at: string | null
  created_at: string
  updated_at: string
}

// SystemMetrics 对应 Agent 每秒上报的主机资源快照。
export interface SystemMetrics {
  cpu_percent: number
  mem_used: number
  mem_total: number
  mem_percent: number
  disk_used: number
  disk_total: number
  disk_percent: number
  load1: number
  load5: number
  load15: number
  uptime_sec: number
  net_in_bps: number
  net_out_bps: number
  ts: number
}

export interface MonitorNode extends Node {
  metrics?: SystemMetrics
}

// InstallBundle 对应 GET /nodes/:code/install，脚本内容按节点类型生成。
export interface InstallBundle {
  code: string
  type: number
  role: 'ingress' | 'egress'
  filename: string
  command: string
  token: string
  script: string
  install_url: string
  online_command: string
}

export interface RotateNodeTokenResult {
  code: string
  epoch: number
  token: string
}

export interface Package {
  id: number
  name: string
  description: string
  ingress_nodes: string[]
  egress_nodes: string[]
  // 套餐总流量上限，单位 GB，0 表示不限
  traffic_limit_gb: number
  // 可创建的入口规则数量上限，0 表示不限制
  rule_limit: number
  // 各计费周期的售价，未配置的周期不可购买
  prices: Partial<Record<PackageDuration, number>>
  enabled: boolean
  created_at: string
  updated_at: string
}

export type PackageDuration = 'monthly' | 'quarterly' | 'half_year' | 'yearly' | 'permanent'

export interface CreatePackageInput {
  name: string
  description?: string
  ingress_nodes: string[]
  egress_nodes: string[]
  traffic_limit_gb?: number
  rule_limit?: number
  prices: Partial<Record<PackageDuration, number>>
  enabled?: boolean
}

export interface UpdatePackageInput {
  name?: string
  description?: string
  ingress_nodes?: string[]
  egress_nodes?: string[]
  traffic_limit_gb?: number
  rule_limit?: number
  prices?: Partial<Record<PackageDuration, number>>
  enabled?: boolean
}

export interface WalletView {
  balance: number
  package_id: number | null
  package_name: string
  package_period: PackageDuration | ''
}

export interface PurchaseResult {
  balance: number
  package_id: number
  package_name: string
  package_period: PackageDuration
}

export interface PortRule {
  id: number
  // entry 入口规则（监听 + 计费）/ exit 出口规则（落地目标）
  direction: 'entry' | 'exit'
  package_id: number
  node: string
  // 入口规则选定的出口节点；为空表示历史规则按套餐全部出口扇出
  egress_node: string
  name: string
  protocol: string
  // 固定生效：quic 只走 QUIC，tls 只走 TLS1.3+yamux，不做跨协议降级
  transport: string
  listen_port: number
  owner_user_id: number | null
  target_addr: string
  ratio_up: number
  ratio_down: number
  rate_up_bps: number
  rate_down_bps: number
  quota_bytes: number
  used_bytes: number
  enabled: boolean
  status: string
  revision: number
  created_at: string
  updated_at: string
}

export interface TrafficStat {
  id: number
  rule_id: number
  node_code: string
  up_bytes: number
  down_bytes: number
  bill_bytes: number
  recorded_at: string
}

export interface PortRuleStats {
  up_bytes: number
  down_bytes: number
  bill_bytes: number
  records: TrafficStat[]
}

export interface CreateNodeInput {
  name: string
  type: number
  // 可选，留空则由 Agent 上报外网 IP 回填
  public_addr?: string
}

export interface UpdateNodeInput {
  name?: string
}

export interface UpdateNodeRateInput {
  rate_up_mbps?: number
  rate_down_mbps?: number
  ratio_up?: number
  ratio_down?: number
}

export interface CreatePortRuleInput {
  name: string
  node: string
  // 套餐内选定的出口节点，必填
  egress_node: string
  protocol?: string
  // quic / tls，创建时选择隧道协议
  transport?: string
  // 留空/0 时由后端在 10000..60000 内自动分配
  listen_port?: number
  // 出口节点实际 Dial 的目标地址，必填
  target_addr: string
}

export interface UpdatePortRuleInput {
  name?: string
  transport?: string
  target_addr?: string
  egress_node?: string
  listen_port?: number
  enabled?: boolean
}

export interface CreatePackageInput {
  name: string
  description?: string
  ingress_nodes: string[]
  egress_nodes: string[]
  traffic_limit_gb?: number
  rule_limit?: number
  enabled?: boolean
}

export interface UpdatePackageInput {
  name?: string
  description?: string
  ingress_nodes?: string[]
  egress_nodes?: string[]
  traffic_limit_gb?: number
  rule_limit?: number
  enabled?: boolean
}

export interface MenuItem {
  key: string
  path: string
  title: string
  icon: string
  perm: string
  children?: MenuItem[]
}

export interface RoleBrief {
  code: string
  name: string
}

export interface User {
  id: number
  username: string
  display_name: string
  role_id: number
  // 绑定的套餐，创建端口规则时自动使用
  package_id: number | null
  // 购买时选择的计费周期，用于区分已购买/续费
  package_period: PackageDuration | ''
  // 账户余额，单位元
  balance: number
  status: string
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export interface UserView extends User {
  role_code: string
  role_name: string
}

export interface Permission {
  id: number
  code: string
  name: string
  group: string
  sort: number
}

export interface Role {
  id: number
  code: string
  name: string
  description: string
  builtin: boolean
  permission_codes: string[]
  created_at: string
  updated_at: string
}

export interface RoleView extends Role {
  user_count: number
}

export interface TokenResult {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface LoginResult extends TokenResult {
  user: User
  role: RoleBrief
  permissions: string[]
  menus: MenuItem[]
}

export interface MeResponse {
  user: User
  role: RoleBrief
  permissions: string[]
  menus: MenuItem[]
}

export interface ProfileResponse {
  user: UserView
  role: Role
  permissions: string[]
}

export interface SystemSettings {
  id: number
  site_title: string
  site_subtitle: string
  // 站点公告内容，为空表示不展示
  announcement: string
  register_open: boolean
  register_review: boolean
  distribution_open: boolean
  // 分销比例，单位百分比 0-100
  distribution_ratio: number
  // 站点公网地址，远端 Agent 访问控制面的唯一入口
  public_url: string
  // Agent 二进制与证书包的下载目录
  agent_download_url: string
  // 下发给 Agent 的 MQTT 接入地址
  agent_broker: string
  created_at: string
  updated_at: string
}

// PublicSiteConfig 是登录/注册页可见的站点信息。
export interface PublicSiteConfig {
  site_title: string
  site_subtitle: string
  announcement: string
  register_open: boolean
  register_review: boolean
}

export interface UpdateSystemSettingsInput {
  site_title?: string
  site_subtitle?: string
  announcement?: string
  register_open?: boolean
  register_review?: boolean
  distribution_open?: boolean
  distribution_ratio?: number
  public_url?: string
  agent_download_url?: string
  agent_broker?: string
}

// 支付方式配置；内置渠道不可删除，仅支持开关与配置编辑。
export interface PaymentChannel {
  id: number
  code: string
  name: string
  enabled: boolean
  builtin: boolean
  config: Record<string, string>
  created_at: string
  updated_at: string
}

export interface UpdatePaymentChannelInput {
  enabled?: boolean
  config?: Record<string, string>
}

// 钱包商城可用的在线支付方式，由易支付配置中的微信/支付宝开关决定。
export interface PaymentMethod {
  type: 'wxpay' | 'alipay'
  name: string
}

export interface CreatePaymentOrderInput {
  package_id: number
  period: PackageDuration
  pay_type: 'wxpay' | 'alipay'
}

// 充值金额由用户自定义，快捷档位由后端配置返回。
export interface CreateRechargeOrderInput {
  amount: number
  pay_type: 'wxpay' | 'alipay'
}

export interface RechargeOptions {
  presets: number[]
}

export interface PaymentOrderView {
  out_trade_no: string
  amount: number
  pay_type: string
  status: 'pending' | 'paid' | 'failed'
  pay_url: string
  qrcode: string
}

export interface PaymentOrderStatus {
  id: number
  out_trade_no: string
  user_id: number
  kind: 'package' | 'recharge'
  package_id: number
  period: PackageDuration
  amount: number
  pay_type: string
  status: 'pending' | 'paid' | 'failed'
  trade_no: string
  paid_at: string | null
  created_at: string
  updated_at: string
}

export interface RegisterResult {
  username: string
  status: string
}

export interface TrafficTotals {
  up_bytes: number
  down_bytes: number
  bill_bytes: number
}

export interface TrafficBucket {
  bucket: string
  up_bytes: number
  down_bytes: number
  bill_bytes: number
}

export interface DashboardPackage {
  name: string
  period: string
  rule_limit: number
  rule_used: number
  // -1 表示不限
  rule_remaining: number
  traffic_limit_bytes: number
  traffic_used_bytes: number
  // -1 表示不限
  traffic_remaining_bytes: number
  traffic_limit_gb: number
}

export interface DashboardSummary {
  package: DashboardPackage
  traffic_24h: TrafficTotals
  series: TrafficBucket[]
}

export interface InviteInfo {
  enabled: boolean
  ratio: number
  code: string
  url: string
  invited_count: number
  total_commission: number
}

export interface CreateUserInput {
  username: string
  password: string
  display_name?: string
  role_code: string
  package_id?: number
  status?: string
}

export interface UpdateUserInput {
  display_name?: string
  role_code?: string
  package_id?: number
  status?: string
  password?: string
}

export interface CreateRoleInput {
  code: string
  name: string
  description?: string
  permission_codes: string[]
}

export interface UpdateRoleInput {
  name?: string
  description?: string
  permission_codes?: string[]
}
