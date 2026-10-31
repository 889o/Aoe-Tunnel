<script setup lang="ts">
import { ShieldAlert } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import PaperButton from '@/components/paper/PaperButton.vue'
import PaperCard from '@/components/paper/PaperCard.vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function goHome() {
  const first = auth.menus[0]?.path ?? '/login'
  router.push(first)
}
</script>

<template>
  <PaperCard pad="lg">
    <div class="flex flex-col items-center py-10 text-center">
      <div class="paper-sink mb-5 flex size-16 items-center justify-center rounded-2xl">
        <ShieldAlert class="size-7" style="color: var(--err)" />
      </div>
      <h2 class="font-paper text-xl font-semibold">没有访问权限</h2>
      <p class="mt-2 max-w-md text-xs text-faint">
        当前账号的角色 {{ auth.role?.name ?? '' }} 未包含该页面所需权限码，请联系超级管理员调整角色权限。
      </p>
      <PaperButton class="mt-6" size="sm" variant="primary" @click="goHome">返回可访问页面</PaperButton>
    </div>
  </PaperCard>
</template>
