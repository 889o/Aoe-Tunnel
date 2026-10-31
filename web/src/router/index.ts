import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { title: '登录' },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/pages/RegisterPage.vue'),
      meta: { title: '注册' },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/pages/DashboardPage.vue'),
          meta: { title: 'Dashboard', perm: 'dashboard:view' },
        },
        {
          path: 'nodes',
          name: 'nodes',
          component: () => import('@/pages/NodesPage.vue'),
          meta: { title: '节点设置', perm: 'node:read' },
        },
        {
          path: 'packages',
          name: 'packages',
          component: () => import('@/pages/PackagesPage.vue'),
          meta: { title: '套餐管理', perm: 'package:list' },
        },
        {
          path: 'store',
          name: 'store',
          component: () => import('@/pages/WalletStorePage.vue'),
          meta: { title: '钱包商城', perm: 'wallet:view' },
        },
        {
          path: 'ports',
          name: 'ports',
          component: () => import('@/pages/PortsPage.vue'),
          meta: { title: '转发规则', perm: 'port:list' },
        },
        {
          path: 'monitor',
          name: 'monitor',
          component: () => import('@/pages/MonitorPage.vue'),
          meta: { title: '服务器监控', perm: 'node:read' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/pages/UsersPage.vue'),
          meta: { title: '用户管理', perm: 'user:list' },
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('@/pages/ProfilePage.vue'),
          meta: { title: '个人设置', perm: 'profile:view' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/pages/SystemSettingsPage.vue'),
          meta: { title: '系统设置', perm: 'system:view' },
        },
        {
          path: 'forbidden',
          name: 'forbidden',
          component: () => import('@/pages/ForbiddenPage.vue'),
          meta: { title: '无访问权限' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.path === '/login' || to.path === '/register') {
    return auth.isAuthed ? { path: '/dashboard' } : true
  }
  if (!auth.isAuthed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (!auth.loaded) {
    try {
      await auth.ensureLoaded()
    } catch {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }
  const perm = to.meta.perm as string | undefined
  if (perm && !auth.hasPerm(perm)) {
    return { path: '/forbidden' }
  }
  return true
})

export default router
