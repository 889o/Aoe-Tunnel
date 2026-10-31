import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { getMe, login as apiLogin, logout as apiLogout } from '@/api'
import { clearTokens, getAccessToken, setAuthFailureHandler, setTokens } from '@/api/http'
import type { LoginResult, MenuItem, RoleBrief, User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  // token 用 ref 镜像 localStorage，否则 computed 无响应依赖会永久缓存初始值。
  const token = ref(getAccessToken())
  const user = ref<User | null>(null)
  const role = ref<RoleBrief | null>(null)
  const permissions = ref<string[]>([])
  const menus = ref<MenuItem[]>([])
  const loaded = ref(false)

  const isAuthed = computed(() => token.value.length > 0)

  function apply(payload: { user: User; role: RoleBrief; permissions: string[]; menus: MenuItem[] }) {
    user.value = payload.user
    role.value = payload.role
    permissions.value = payload.permissions ?? []
    menus.value = payload.menus ?? []
    loaded.value = true
  }

  function hasPerm(code?: string | null): boolean {
    if (!code) return true
    return permissions.value.includes(code)
  }

  function reset() {
    user.value = null
    role.value = null
    permissions.value = []
    menus.value = []
    loaded.value = false
  }

  setAuthFailureHandler(() => {
    clearTokens()
    token.value = ''
    reset()
  })

  async function login(username: string, password: string) {
    const res: LoginResult = await apiLogin(username.trim(), password)
    setTokens(res.access_token, res.refresh_token)
    token.value = res.access_token
    apply(res)
  }

  // 刷新后页面只剩 token，需要重新拉取身份与菜单；失败时交由路由守卫跳登录。
  async function ensureLoaded() {
    if (loaded.value || !isAuthed.value) return
    apply(await getMe())
  }

  async function logout() {
    try {
      await apiLogout()
    } catch {
      // 登出失败也要清本地凭证，服务端会话可由管理员踢下线。
    }
    clearTokens()
    token.value = ''
    reset()
  }

  return { user, role, permissions, menus, loaded, isAuthed, hasPerm, login, ensureLoaded, logout, reset }
})
