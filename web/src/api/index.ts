import { request } from './http'
import type {
  CreateNodeInput,
  CreatePackageInput,
  CreatePortRuleInput,
  CreateRoleInput,
  CreateUserInput,
  CreatePaymentOrderInput,
  CreateRechargeOrderInput,
  DashboardSummary,
  InstallBundle,
  InviteInfo,
  LoginResult,
  MeResponse,
  MonitorNode,
  Node,
  Package,
  PaymentChannel,
  PaymentMethod,
  PaymentOrderStatus,
  PaymentOrderView,
  Permission,
  PortRule,
  PortRuleStats,
  ProfileResponse,
  PublicSiteConfig,
  PurchaseResult,
  RechargeOptions,
  RegisterResult,
  Role,
  RoleView,
  RotateNodeTokenResult,
  SystemMetrics,
  SystemSettings,
  UpdateNodeInput,
  UpdateNodeRateInput,
  UpdatePackageInput,
  UpdatePaymentChannelInput,
  UpdatePortRuleInput,
  UpdateRoleInput,
  UpdateSystemSettingsInput,
  UpdateUserInput,
  User,
  UserView,
  WalletView,
} from '@/types'

// 认证
export const login = (username: string, password: string) =>
  request<LoginResult>('POST', '/auth/login', { username, password })
export const register = (username: string, password: string, inviteCode?: string) =>
  request<RegisterResult>('POST', '/auth/register', {
    username,
    password,
    ...(inviteCode ? { invite_code: inviteCode } : {}),
  })
export const getInviteInfo = () => request<InviteInfo>('GET', '/profile/invite')
export const getPublicSiteConfig = () => request<PublicSiteConfig>('GET', '/site/config')
export const logout = () => request<{ logged_out: boolean }>('POST', '/auth/logout')
export const getMe = () => request<MeResponse>('GET', '/auth/me')
export const getDashboard = () => request<DashboardSummary>('GET', '/dashboard/summary')

// 节点
export const listNodes = () => request<Node[]>('GET', '/nodes')
export const createNode = (input: CreateNodeInput) => request<Node>('POST', '/nodes', input)
export const updateNode = (code: string, input: UpdateNodeInput) =>
  request<Node>('PATCH', `/nodes/${encodeURIComponent(code)}`, input)
export const updateNodeRate = (code: string, input: UpdateNodeRateInput) =>
  request<Node>('PATCH', `/nodes/${encodeURIComponent(code)}/rate-limit`, input)
export const deleteNode = (code: string) =>
  request<{ deleted: string }>('DELETE', `/nodes/${encodeURIComponent(code)}`)
export const getNodeInstall = (code: string) =>
  request<InstallBundle>('GET', `/nodes/${encodeURIComponent(code)}/install`)
export const rotateNodeToken = (code: string) =>
  request<RotateNodeTokenResult>('POST', `/nodes/${encodeURIComponent(code)}/rotate-token`)

// 服务器监控：仅内存时序，1s 一点，最近 60 点
export const getMonitorOverview = () => request<MonitorNode[]>('GET', '/monitor/overview')
export const getNodeStatHistory = (code: string) =>
  request<SystemMetrics[]>('GET', `/nodes/${encodeURIComponent(code)}/stat-history`)

// 套餐：绑定入口/出口节点，入口规则自动向套餐内出口扇出
export const listPackages = () => request<Package[]>('GET', '/packages')
export const createPackage = (input: CreatePackageInput) => request<Package>('POST', '/packages', input)
export const updatePackage = (id: number, input: UpdatePackageInput) =>
  request<Package>('PATCH', `/packages/${id}`, input)
export const deletePackage = (id: number) => request<{ deleted: number }>('DELETE', `/packages/${id}`)

// 端口
export interface PortRuleQuery {
  node?: string
  direction?: '' | 'entry' | 'exit'
  package_id?: number
}
export const listPortRules = (q: PortRuleQuery = {}) => {
  const params: Record<string, string> = {}
  if (q.node) params.node = q.node
  if (q.direction) params.direction = q.direction
  if (q.package_id) params.package_id = String(q.package_id)
  return request<PortRule[]>('GET', '/port-rules', undefined, params)
}
export const getPortRule = (id: number) => request<PortRule>('GET', `/port-rules/${id}`)
export const createPortRule = (input: CreatePortRuleInput) => request<PortRule>('POST', '/port-rules', input)
export const updatePortRule = (id: number, input: UpdatePortRuleInput) =>
  request<PortRule>('PATCH', `/port-rules/${id}`, input)
export const deletePortRule = (id: number) => request<{ deleted: number }>('DELETE', `/port-rules/${id}`)
export const getPortRuleStats = (id: number, from?: string, to?: string) => {
  const params: Record<string, string> = {}
  if (from) params.from = from
  if (to) params.to = to
  return request<PortRuleStats>('GET', `/port-rules/${id}/stats`, undefined, params)
}

// 用户
export const listUsers = () => request<UserView[]>('GET', '/users')
export const createUser = (input: CreateUserInput) => request<User>('POST', '/users', input)
export const updateUser = (id: number, input: UpdateUserInput) => request<User>('PATCH', `/users/${id}`, input)
export const deleteUser = (id: number) => request<{ deleted: number }>('DELETE', `/users/${id}`)

// 角色与权限
export const listRoles = () => request<RoleView[]>('GET', '/roles')
export const createRole = (input: CreateRoleInput) => request<Role>('POST', '/roles', input)
export const updateRole = (id: number, input: UpdateRoleInput) => request<Role>('PATCH', `/roles/${id}`, input)
export const deleteRole = (id: number) => request<{ deleted: number }>('DELETE', `/roles/${id}`)
export const listPermissions = () => request<Permission[]>('GET', '/permissions')

// 个人设置与系统设置
export const getProfile = () => request<ProfileResponse>('GET', '/profile')
export const updateProfile = (input: { display_name?: string; old_password?: string; new_password?: string }) =>
  request<User>('PATCH', '/profile', input)
export const getSystemSettings = () => request<SystemSettings>('GET', '/system/settings')
export const updateSystemSettings = (input: UpdateSystemSettingsInput) =>
  request<SystemSettings>('PATCH', '/system/settings', input)
export const getPaymentChannels = () => request<PaymentChannel[]>('GET', '/system/payment-channels')
export const updatePaymentChannel = (code: string, input: UpdatePaymentChannelInput) =>
  request<PaymentChannel>('PATCH', `/system/payment-channels/${code}`, input)

// 钱包商城
export const getWallet = () => request<WalletView>('GET', '/wallet')
export const purchasePackage = (packageID: number, period: string) =>
  request<PurchaseResult>('POST', '/wallet/purchase', { package_id: packageID, period })
export const getPaymentMethods = () => request<PaymentMethod[]>('GET', '/wallet/payment-methods')
export const createPaymentOrder = (input: CreatePaymentOrderInput) =>
  request<PaymentOrderView>('POST', '/wallet/payment-orders', input)
export const getRechargeOptions = () => request<RechargeOptions>('GET', '/wallet/recharge-options')
export const createRechargeOrder = (input: CreateRechargeOrderInput) =>
  request<PaymentOrderView>('POST', '/wallet/recharge-orders', input)
export const getPaymentOrder = (outTradeNo: string) =>
  request<PaymentOrderStatus>('GET', `/wallet/payment-orders/${encodeURIComponent(outTradeNo)}`)
