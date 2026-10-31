<script setup lang="ts">
const model = defineModel<boolean>({ default: false })

withDefaults(defineProps<{ label?: string; disabled?: boolean }>(), { disabled: false })
</script>

<template>
  <button
    type="button"
    class="inline-flex items-center gap-2.5 text-sm text-muted disabled:opacity-50"
    :disabled="$props.disabled"
    role="switch"
    :aria-checked="model"
    @click="model = !model"
  >
    <span
      class="relative inline-flex h-6 w-11 items-center rounded-full border transition-all duration-200"
      :style="{
        background: model ? 'var(--accent)' : 'var(--paper-2)',
        borderColor: 'var(--line)',
        boxShadow: model
          ? 'inset 2px 2px 5px var(--sink-hi), inset -2px -2px 5px var(--sink-lo)'
          : 'inset 3px 3px 7px var(--sink-hi), inset -3px -3px 7px var(--sink-lo)',
      }"
    >
      <span
        class="absolute size-4 rounded-full transition-all duration-200"
        :style="{
          left: model ? '26px' : '4px',
          background: 'var(--paper-1)',
          boxShadow: '-2px -2px 5px var(--raise-hi), 2px 2px 6px var(--raise-lo)',
        }"
      />
    </span>
    <span v-if="$props.label">{{ $props.label }}</span>
  </button>
</template>
