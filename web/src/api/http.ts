import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { toast } from 'vue-sonner'
import type { ApiResult, TokenResult } from '@/types'

export const ACCESS_TOKEN_KEY = 'aoe-access-token'
export const REFRESH_TOKEN_KEY = 'aoe-refresh-token'

export function getAccessToken(): string {
  return localStorage.getItem(ACCESS_TOKEN_KEY) ?? ''
}

export function getRefreshToken(): string {
  return localStorage.getItem(REFRESH_TOKEN_KEY) ?? ''
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, access)
  localStorage.setItem(REFRESH_TOKEN_KEY, refresh)
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

// 鉴权彻底失败时由 store 注册清理回调，避免 http 层反向依赖 pinia store。
let authFailureHandler: (() => void) | null = null

export function setAuthFailureHandler(fn: () => void) {
  authFailureHandler = fn
}

export const http = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

// 刷新请求使用独立的 axios 实例，避免拦截器递归。
const refreshClient = axios.create({ baseURL: '/api/v1', timeout: 15000 })

http.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  return config
})

let refreshing: Promise<string> | null = null

async function refreshAccessToken(): Promise<string> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) throw new Error('missing refresh token')
  const res = await refreshClient.post<ApiResult<TokenResult>>('/auth/refresh', { refresh_token: refreshToken })
  const data = res.data.data
  setTokens(data.access_token, data.refresh_token)
  return data.access_token
}

function redirectToLogin() {
  clearTokens()
  authFailureHandler?.()
  if (!window.location.hash.startsWith('#/login')) {
    window.location.hash = '#/login'
  }
}

http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResult<unknown>>) => {
    const status = error.response?.status
    const message = error.response?.data?.message || error.message || '请求失败'
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined
    const isAuthEndpoint = original?.url?.startsWith('/auth/') ?? false

    // access token 过期时静默刷新一次并重放原请求，并发的 401 共用同一次刷新。
    if (status === 401 && original && !original._retried && !isAuthEndpoint) {
      original._retried = true
      try {
        if (!refreshing) {
          refreshing = refreshAccessToken().finally(() => {
            refreshing = null
          })
        }
        const token = await refreshing
        original.headers.set('Authorization', `Bearer ${token}`)
        return http.request(original)
      } catch {
        redirectToLogin()
        toast.error('登录已失效，请重新登录')
        return Promise.reject(error)
      }
    }

    if (status === 401) {
      redirectToLogin()
      toast.error(message)
    } else {
      toast.error(message)
    }
    return Promise.reject(error)
  },
)

export async function request<T>(
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
  url: string,
  data?: unknown,
  params?: Record<string, unknown>,
): Promise<T> {
  const res = await http.request<ApiResult<T>>({ method, url, data, params })
  return res.data.data
}
