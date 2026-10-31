import type { Directive } from 'vue'
import { useAuthStore } from '@/stores/auth'

// v-permission="'user:create'"：无权限码时隐藏元素，权限码变更后同步。
export const permission: Directive<HTMLElement, string | undefined> = {
  mounted(el, binding) {
    apply(el, binding.value)
  },
  updated(el, binding) {
    apply(el, binding.value)
  },
}

function apply(el: HTMLElement, code?: string) {
  const auth = useAuthStore()
  el.style.display = auth.hasPerm(code) ? '' : 'none'
}
