<script setup lang="ts">
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{ tone?: 'default' | 'ok' | 'warn' | 'err'; dot?: boolean }>(), {
  tone: 'default',
  dot: false,
})

const toneClass = {
  default: '',
  ok: 'paper-badge-ok',
  warn: 'paper-badge-warn',
  err: 'paper-badge-err',
}
const dotColor = {
  default: 'var(--ink-3)',
  ok: 'var(--ok)',
  warn: 'var(--warn)',
  err: 'var(--err)',
}
</script>

<template>
  <span :class="cn('paper-badge', toneClass[props.tone])">
    <span
      v-if="props.dot"
      class="inline-block size-1.5 rounded-full"
      :style="{ background: dotColor[props.tone] }"
    />
    <slot />
  </span>
</template>
