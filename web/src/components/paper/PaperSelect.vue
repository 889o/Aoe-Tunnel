<script setup lang="ts">
const model = defineModel<string>({ default: '' })

const props = withDefaults(
  defineProps<{
    label?: string
    options: Array<{ label: string; value: string }>
    placeholder?: string
    disabled?: boolean
  }>(),
  { disabled: false },
)

function onChange(event: Event) {
  model.value = (event.target as HTMLSelectElement).value
}
</script>

<template>
  <label class="block">
    <span v-if="props.label" class="paper-label">{{ props.label }}</span>
    <select class="paper-select" :value="model" :disabled="props.disabled" @change="onChange">
      <option v-if="props.placeholder" value="">{{ props.placeholder }}</option>
      <option v-for="opt in props.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
    </select>
  </label>
</template>
