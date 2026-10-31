<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    variant?: 'default' | 'primary' | 'ghost' | 'danger'
    size?: 'md' | 'sm'
    loading?: boolean
    disabled?: boolean
    block?: boolean
  }>(),
  { variant: 'default', size: 'md', loading: false, disabled: false, block: false },
)

const classes = {
  default: '',
  primary: 'paper-btn-primary',
  ghost: 'paper-btn-ghost',
  danger: 'paper-btn-danger',
}
</script>

<template>
  <button
    type="button"
    :disabled="props.disabled || props.loading"
    :class="cn('paper-btn', classes[props.variant], props.size === 'sm' && 'paper-btn-sm', props.block && 'w-full')"
  >
    <Loader2 v-if="props.loading" class="size-4 animate-spin" />
    <slot />
  </button>
</template>
