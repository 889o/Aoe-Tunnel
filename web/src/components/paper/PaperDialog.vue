<script setup lang="ts">
import { X } from 'lucide-vue-next'
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'

const open = defineModel<boolean>('open', { default: false })

withDefaults(defineProps<{ title?: string; description?: string; width?: string }>(), {
  width: '560px',
})
</script>

<template>
  <DialogRoot :open="open" @update:open="open = $event">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-40 bg-black/45 backdrop-blur-[2px]" />
      <DialogContent
        class="paper-card fixed left-1/2 top-1/2 z-50 max-h-[86vh] w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto p-6"
        :style="{ maxWidth: $props.width }"
      >
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <DialogTitle class="font-paper text-lg font-semibold tracking-wide">
              {{ $props.title }}
            </DialogTitle>
            <DialogDescription v-if="$props.description" class="mt-1 text-xs text-faint">
              {{ $props.description }}
            </DialogDescription>
          </div>
          <DialogClose class="paper-btn paper-btn-ghost paper-btn-sm" aria-label="关闭">
            <X class="size-4" />
          </DialogClose>
        </div>
        <slot />
        <div v-if="$slots.footer" class="mt-6 flex items-center justify-end gap-2">
          <slot name="footer" />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
