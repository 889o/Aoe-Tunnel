import { ref } from 'vue'
import { defineStore } from 'pinia'

const THEME_STORAGE = 'aoe-theme'
export type Theme = 'dark' | 'light'

function readTheme(): Theme {
  return localStorage.getItem(THEME_STORAGE) === 'light' ? 'light' : 'dark'
}

export const useThemeStore = defineStore('theme', () => {
  // 纯黑为默认主题，纯白为可切换的亮色；不再提供乳白配色
  const theme = ref<Theme>(readTheme())

  function apply(next: Theme) {
    theme.value = next
    document.documentElement.dataset.theme = next
    localStorage.setItem(THEME_STORAGE, next)
  }

  function toggle() {
    apply(theme.value === 'dark' ? 'light' : 'dark')
  }

  function init() {
    apply(readTheme())
  }

  return { theme, apply, toggle, init }
})
