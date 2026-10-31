<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Copy, KeyRound, Link2, RefreshCw, UserCog } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { useRouter } from 'vue-router'
import { getInviteInfo, getProfile, updateProfile } from '@/api'
import type { InviteInfo, ProfileResponse } from '@/types'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import PaperInput from '@/components/paper/PaperInput.vue'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/lib/utils'

const auth = useAuthStore()
const router = useRouter()
const profile = ref<ProfileResponse | null>(null)
const invite = ref<InviteInfo | null>(null)
const loading = ref(false)
const savingPassword = ref(false)

const passwordForm = reactive({ old_password: '', new_password: '' })

async function load() {
  loading.value = true
  try {
    profile.value = await getProfile()
    invite.value = await getInviteInfo()
  } finally {
    loading.value = false
  }
}

async function copyInvite() {
  const text = invite.value?.url
  if (!text) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      ta.remove()
    }
    toast.success('邀请链接已复制')
  } catch {
    toast.error('复制失败，请手动选中复制')
  }
}

async function savePassword() {
  if (!passwordForm.old_password || !passwordForm.new_password) {
    toast.error('请填写原密码与新密码')
    return
  }
  if (passwordForm.new_password.length < 6) {
    toast.error('新密码至少 6 位')
    return
  }
  savingPassword.value = true
  try {
    await updateProfile({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password,
    })
    toast.success('密码已修改，请重新登录')
    await auth.logout()
    router.push('/login')
  } finally {
    savingPassword.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="grid max-w-3xl gap-4">
    <PaperCard>
      <div class="mb-4 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <UserCog class="size-4" style="color: var(--accent)" />
          <h2 class="font-paper text-base font-semibold">账号信息</h2>
        </div>
        <PaperButton size="sm" :loading="loading" @click="load">
          <RefreshCw class="size-4" :class="loading && 'animate-spin'" />
          刷新
        </PaperButton>
      </div>

      <div class="paper-sink mb-4 flex items-center justify-between rounded-xl px-4 py-3">
        <div>
          <p class="font-mono text-sm" style="color: var(--ink-1)">{{ profile?.user.username ?? '—' }}</p>
          <p class="mt-0.5 text-[11px] text-faint">ID #{{ profile?.user.id }} · 最后登录 {{ formatTime(profile?.user.last_login_at ?? null) }}</p>
        </div>
        <div class="flex flex-col items-end gap-1">
          <PaperBadge :tone="profile?.role.code === 'super_admin' ? 'warn' : 'default'" dot>
            {{ profile?.role.name ?? '—' }}
          </PaperBadge>
          <span class="font-mono text-[10px] text-faint">{{ profile?.role.code }}</span>
        </div>
      </div>

      <hr class="paper-divider my-5" />

      <div class="mb-3 flex items-center gap-2">
        <KeyRound class="size-4" style="color: var(--warn)" />
        <h2 class="font-paper text-base font-semibold">修改密码</h2>
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <PaperInput
          v-model="passwordForm.old_password"
          type="password"
          label="原密码"
          autocomplete="current-password"
        />
        <PaperInput
          v-model="passwordForm.new_password"
          type="password"
          label="新密码"
          autocomplete="new-password"
          hint="至少 6 位，修改后需要重新登录"
        />
      </div>
      <PaperButton class="mt-4" size="sm" :loading="savingPassword" @click="savePassword">
        <KeyRound class="size-4" />
        更新密码
      </PaperButton>
    </PaperCard>

    <PaperCard>
      <div class="mb-3 flex items-center gap-2">
        <Link2 class="size-4" style="color: var(--accent)" />
        <h2 class="font-paper text-base font-semibold">邀请链接</h2>
      </div>
      <p class="mb-3 text-[11px] text-faint">
        <span v-if="invite?.enabled">好友通过该链接注册即绑定上下级关系，其购买套餐按 {{ invite.ratio }}% 返佣到你的余额。</span>
        <span v-else>当前未开启分销，通过该链接注册不会绑定上下级关系。</span>
      </p>
      <div class="paper-sink flex items-center gap-2 rounded-xl px-3 py-2">
        <input
          :value="invite?.url || '—'"
          readonly
          class="min-w-0 flex-1 bg-transparent font-mono text-[12px] outline-none"
          style="color: var(--ink-1)"
          @focus="($event.target as HTMLInputElement).select()"
        />
        <PaperButton size="sm" :disabled="!invite?.url" @click="copyInvite">
          <Copy class="size-4" />
          复制
        </PaperButton>
      </div>
      <div class="mt-3 grid grid-cols-3 gap-2">
        <div class="paper-sink rounded-xl py-2 text-center">
          <p class="font-mono text-sm" style="color: var(--ink-1)">{{ invite?.code || '—' }}</p>
          <p class="mt-0.5 text-[10px] text-faint">我的邀请码</p>
        </div>
        <div class="paper-sink rounded-xl py-2 text-center">
          <p class="font-mono text-sm" style="color: var(--ink-1)">{{ invite?.invited_count ?? 0 }}</p>
          <p class="mt-0.5 text-[10px] text-faint">已邀请人数</p>
        </div>
        <div class="paper-sink rounded-xl py-2 text-center">
          <p class="font-mono text-sm" style="color: var(--ink-1)">¥ {{ (invite?.total_commission ?? 0).toFixed(2) }}</p>
          <p class="mt-0.5 text-[10px] text-faint">累计佣金</p>
        </div>
      </div>
    </PaperCard>
  </div>
</template>
