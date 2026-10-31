<script setup lang="ts">
const model = defineModel<string | number>({ default: '' })

const props = withDefaults(
  defineProps<{
    label?: string
    hint?: string
    type?: string
    placeholder?: string
    disabled?: boolean
    min?: number
    max?: number
    step?: number
    autocomplete?: string
  }>(),
  { type: 'text', disabled: false },
)

function onInput(event: Event) {
  const el = event.target as HTMLInputElement
  model.value = props.type === 'number' ? (el.value === '' ? '' : Number(el.value)) : el.value
}
</script>

<template>
  <label class="block">
    <span v-if="props.label" class="paper-label">{{ props.label }}</span>
    <input
      :type="props.type"
      :value="model"
      :placeholder="props.placeholder"
      :disabled="props.disabled"
      :min="props.min"
      :max="props.max"
      :step="props.step"
      :autocomplete="props.autocomplete"
      class="paper-input"
      @input="onInput"
    />
    <span v-if="props.hint" class="mt-1 block text-[11px] text-faint">{{ props.hint }}</span>
  </label>
</template>
