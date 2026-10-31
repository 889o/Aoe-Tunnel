<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Activity,
  Boxes,
  Cpu,
  LayoutDashboard,
  LogOut,
  Network,
  Package,
  Settings,
  ShieldCheck,
  UserCog,
  Users,
  Wallet,
} from 'lucide-vue-next'
import PaperBadge from '@/components/paper/PaperBadge.vue'
import ThemeToggle from '@/components/paper/ThemeToggle.vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

// 后端只下发 icon 名称，前端负责映射 lucide 组件，缺省用 Activity 兜底。
const iconMap: Record<string, Component> = {
  LayoutDashboard,
  Network,
  Boxes,
  Package,
  Users,
  ShieldCheck,
  Cpu,
  UserCog,
  Settings,
  Wallet,
  Activity,
}

const title = computed(() => (route.meta.title as string) ?? '控制台')
const roleLabel = computed(() => auth.role?.name ?? '—')
const displayName = computed(() => auth.user?.display_name || auth.user?.username || '未登录')

async function logout() {
  await auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="mx-auto flex min-h-screen w-full max-w-[1400px] flex-col gap-5 px-4 py-6 md:flex-row md:px-8">
    <aside class="md:w-60 md:shrink-0">
      <div class="paper-card sticky top-6 p-4 md:p-5">
        <div class="flex items-center gap-3">
          <div class="paper-sink flex size-10 items-center justify-center rounded-xl" aria-hidden="true">
            <span class="font-paper text-lg font-bold text-accent">A</span>
          </div>
          <div>
            <p class="font-paper text-base font-semibold leading-tight">Aoe-tunnel</p>
            <p class="text-[11px] text-faint">多端口托管中转</p>
          </div>
        </div>

        <hr class="paper-divider my-4" />

        <nav class="flex flex-wrap gap-1 md:flex-col">
          <RouterLink
            v-for="item in auth.menus"
            :key="item.key"
            :to="item.path"
            class="flex flex-1 items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm transition-all duration-200 md:flex-none"
            :style="
              route.path === item.path
                ? 'background: var(--accent-soft); color: var(--ink-1); box-shadow: inset 2px 2px 6px var(--sink-hi), inset -2px -2px 6px var(--sink-lo)'
                : 'color: var(--ink-2)'
            "
          >
            <component :is="iconMap[item.icon] ?? Activity" class="size-4 shrink-0" />
            <span class="truncate">{{ item.title }}</span>
          </RouterLink>
        </nav>

        <hr class="paper-divider my-4" />

        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate text-sm font-medium" style="color: var(--ink-1)">{{ displayName }}</p>
            <PaperBadge class="mt-1" :tone="auth.role?.code === 'super_admin' ? 'warn' : 'default'" dot>
              {{ roleLabel }}
            </PaperBadge>
          </div>
          <button type="button" class="paper-btn paper-btn-ghost paper-btn-sm" title="退出登录" @click="logout">
            <LogOut class="size-4" />
          </button>
        </div>
      </div>
    </aside>

    <main class="flex min-w-0 flex-1 flex-col gap-5">
      <header class="flex items-center justify-between gap-3">
        <div>
          <h1 class="font-paper text-2xl font-semibold">{{ title }}</h1>
        </div>
        <div class="flex items-center gap-2">
          <ThemeToggle />
          <button type="button" class="paper-btn paper-btn-sm" title="退出登录" @click="logout">
            <LogOut class="size-4" />
            <span class="hidden sm:inline">退出</span>
          </button>
        </div>
      </header>

      <RouterView />
    </main>
  </div>
</template>
