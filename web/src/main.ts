import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { permission } from './directives/permission'
import { useThemeStore } from './stores/theme'
import './styles/main.css'

const app = createApp(App)
app.use(createPinia())
useThemeStore().init()
app.directive('permission', permission)
app.use(router)
app.mount('#app')
