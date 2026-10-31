<script setup lang="ts">
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{ interactive?: boolean; pad?: 'none' | 'sm' | 'md' | 'lg' }>(), {
  interactive: false,
  pad: 'md',
})

const padClass: Record<'none' | 'sm' | 'md' | 'lg', string> = {
  none: '',
  sm: 'p-4',
  md: 'p-5',
  lg: 'p-7',
}
</script>

<template>
  <section :class="cn('paper-card', props.interactive && 'paper-card-interactive', padClass[props.pad])">
    <slot />
  </section>
</template>
