<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, UserPlus } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { getPublicSiteConfig, register } from '@/api'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import ThemeToggle from '@/components/paper/ThemeToggle.vue'
import type { PublicSiteConfig } from '@/types'

const router = useRouter()
const route = useRoute()
// 邀请链接携带的邀请码，注册时一并提交，是否生效由后端分销开关决定。
const inviteCode = computed(() => String(route.query.invite ?? '').trim())

const config = ref<PublicSiteConfig | null>(null)
const loading = ref(false)

const form = reactive({ username: '', password: '', confirm: '' })

async function loadConfig() {
  try {
    config.value = await getPublicSiteConfig()
  } catch {
    // 站点信息拉取失败不阻塞注册页，后端注册接口仍会校验注册开关。
  }
}

async function submit() {
  if (!form.username.trim() || !form.password) {
    toast.error('请输入用户名与密码')
    return
  }
  if (form.password !== form.confirm) {
    toast.error('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    const res = await register(form.username.trim(), form.password, inviteCode.value)
    if (res.status === 'pending') {
      toast.success('注册成功，请等待管理员审核')
    } else {
      toast.success('注册成功，请登录')
    }
    router.push('/login')
  } catch {
    // 错误提示由 http 拦截器统一弹出
  } finally {
    loading.value = false
  }
}

onMounted(loadConfig)
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
          <UserPlus class="size-6 text-accent" />
        </div>
        <h1 class="text-center font-paper text-xl font-semibold">注册账号</h1>
        <p class="mt-1 text-center text-xs text-faint">
          {{ config?.register_review ? '注册后需管理员审核通过才能登录' : '注册后即可使用账号登录' }}
        </p>
        <p v-if="inviteCode" class="mt-2 text-center text-[11px] text-faint">
          邀请码 <span class="font-mono">{{ inviteCode }}</span>
        </p>

        <div v-if="config && !config.register_open" class="paper-sink mt-6 rounded-xl px-4 py-4 text-center">
          <p class="text-sm font-medium" style="color: var(--ink-1)">当前站点已关闭注册</p>
          <p class="mt-1 text-[11px] text-faint">如需账号，请联系管理员开通</p>
          <PaperButton class="mt-3" size="sm" @click="router.push('/login')">
            <ArrowLeft class="size-3.5" />
            返回登录
          </PaperButton>
        </div>

        <form v-else class="mt-6 flex flex-col gap-4" @submit.prevent="submit">
          <PaperInput
            v-model="form.username"
            label="用户名"
            placeholder="3-64 位，不含空格"
            autocomplete="username"
          />
          <PaperInput
            v-model="form.password"
            label="密码"
            type="password"
            placeholder="至少 6 位"
            autocomplete="new-password"
          />
          <PaperInput
            v-model="form.confirm"
            label="确认密码"
            type="password"
            placeholder="再次输入密码"
            autocomplete="new-password"
          />
          <PaperButton variant="primary" type="submit" block :loading="loading">完成注册</PaperButton>
        </form>

        <div class="paper-divider my-5" />

        <p class="text-center text-[11px] text-faint">
          已有账号？
          <button type="button" class="text-accent" @click="router.push('/login')">去登录</button>
        </p>
      </div>
    </div>
  </div>
</template>
