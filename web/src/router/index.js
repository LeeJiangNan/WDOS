import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '首页大屏' }
      },
      {
        path: 'templates',
        name: 'Templates',
        component: () => import('../views/templates/TemplateList.vue'),
        meta: { title: '工单模板' }
      },
      {
        path: 'work-orders',
        name: 'WorkOrders',
        component: () => import('../views/workorders/WorkOrderList.vue'),
        meta: { title: '工单数据' }
      },
      {
        path: 'suppression',
        name: 'Suppression',
        component: () => import('../views/suppression/SuppressionList.vue'),
        meta: { title: '抑制策略' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('../views/users/UserList.vue'),
        meta: { title: '用户管理' }
      },
      {
        path: 'permissions',
        name: 'Permissions',
        component: () => import('../views/permissions/PermissionConfig.vue'),
        meta: { title: '权限配置' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  document.title = (to.meta.title || 'WDOS') + ' - WDOS 管理后台'
  const token = localStorage.getItem('wdos_token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
