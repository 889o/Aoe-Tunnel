<script setup lang="ts">
import { Moon, Sun } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'

const theme = useThemeStore()
</script>

<template>
  <button
    type="button"
    class="paper-btn paper-btn-sm"
    :aria-label="theme.theme === 'dark' ? '切换到纸白主题' : '切换到石墨主题'"
    :title="theme.theme === 'dark' ? '纸白' : '石墨'"
    @click="theme.toggle()"
  >
    <Sun v-if="theme.theme === 'dark'" class="size-4" />
    <Moon v-else class="size-4" />
    <span class="hidden sm:inline">{{ theme.theme === 'dark' ? '纸白' : '石墨' }}</span>
  </button>
</template>
