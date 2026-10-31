<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogIn } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getPublicSiteConfig } from '@/api'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import ThemeToggle from '@/components/paper/ThemeToggle.vue'
import { useAuthStore } from '@/stores/auth'
import type { PublicSiteConfig } from '@/types'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const config = ref<PublicSiteConfig | null>(null)
const form = reactive({ username: '', password: '' })
const loading = ref(false)

async function loadConfig() {
  try {
    config.value = await getPublicSiteConfig()
  } catch {
    // 站点信息拉取失败时沿用默认标题，不影响登录。
  }
}

onMounted(loadConfig)

async function submit() {
  if (!form.username.trim() || !form.password) {
    toast.error('请输入用户名与密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    toast.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    router.push(redirect)
  } catch {
    // 错误提示由 http 拦截器统一弹出
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4 py-10">
    <div class="w-full max-w-md">
      <div class="mb-5 flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <div class="paper-sink flex size-9 items-center justify-center rounded-xl">
            <span class="font-paper text-base font-bold text-accent">A</span>
          </div>
          <div>
            <p class="font-paper text-base font-semibold leading-tight">{{ config?.site_title || 'Aoe-tunnel' }}</p>
            <p class="text-[11px] text-faint">{{ config?.site_subtitle || '控制台入口' }}</p>
          </div>
        </div>
        <ThemeToggle />
      </div>

      <div class="paper-card p-7">
        <div class="paper-sink mx-auto mb-5 flex size-14 items-center justify-center rounded-2xl">
          <LogIn class="size-6 text-accent" />
        </div>
        <h1 class="text-center font-paper text-xl font-semibold">账号登录</h1>
        <p class="mt-1 text-center text-xs text-faint">Web 端使用账号会话，Agent 节点使用内置账号登录获取 JWT</p>

        <form class="mt-6 flex flex-col gap-4" @submit.prevent="submit">
          <PaperInput
            v-model="form.username"
            label="用户名"
            placeholder="admin"
            autocomplete="username"
            hint="默认超级管理员账号由 PANEL_ADMIN_USER 指定"
          />
          <PaperInput
            v-model="form.password"
            label="密码"
            type="password"
            placeholder="••••••"
            autocomplete="current-password"
          />
          <PaperButton variant="primary" type="submit" block :loading="loading">登录控制台</PaperButton>
        </form>

        <div class="paper-divider my-5" />

        <p v-if="config?.register_open" class="mb-2 text-center text-[11px]">
          <span class="text-faint">还没有账号？</span>
          <button type="button" class="text-accent" @click="router.push('/register')">注册账号</button>
        </p>

        <p class="text-center text-[11px] text-faint">
          Access Token 15 分钟有效，过期后自动使用 Refresh Token 续期
        </p>
      </div>
    </div>
  </div>
</template>
