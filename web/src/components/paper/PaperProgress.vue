<script setup lang="ts">
import { computed } from 'vue'
import { formatBytes, percent } from '@/lib/utils'

const props = withDefaults(defineProps<{ used: number; total: number; showLabel?: boolean }>(), {
  showLabel: true,
})

const pct = computed(() => percent(props.used, props.total))
const unlimited = computed(() => props.total <= 0)
const tone = computed(() => (pct.value >= 100 ? 'var(--err)' : pct.value >= 80 ? 'var(--warn)' : 'var(--accent)'))
const label = computed(() =>
  unlimited.value ? '不限配额' : `${formatBytes(props.used)} / ${formatBytes(props.total)}`,
)
</script>

<template>
  <div class="w-full">
    <div
      class="paper-sink h-2 w-full overflow-hidden rounded-full p-0"
      role="progressbar"
      :aria-valuenow="Math.round(pct)"
      aria-valuemin="0"
      aria-valuemax="100"
    >
      <div
        v-if="!unlimited"
        class="h-full rounded-full transition-[width] duration-500 ease-out"
        :style="{ width: `${pct}%`, background: `linear-gradient(90deg, color-mix(in srgb, ${tone} 70%, transparent), ${tone})` }"
      />
    </div>
    <div v-if="showLabel" class="mt-1.5 flex items-center justify-between text-[11px] text-faint">
      <span>{{ label }}</span>
      <span v-if="!unlimited">{{ pct.toFixed(0) }}%</span>
    </div>
  </div>
</template>
