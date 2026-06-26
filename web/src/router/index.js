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
        meta: { title: '首页大屏', roles: ['admin','director','manager','supervisor','handler'] }
      },
      {
        path: 'templates',
        name: 'Templates',
        component: () => import('../views/templates/TemplateList.vue'),
        meta: { title: '工单模板', roles: ['admin'] }
      },
      {
        path: 'work-orders',
        name: 'WorkOrders',
        component: () => import('../views/workorders/WorkOrderList.vue'),
        meta: { title: '工单数据', roles: ['admin'] }
      },
      {
        path: 'suppression',
        name: 'Suppression',
        component: () => import('../views/suppression/SuppressionList.vue'),
        meta: { title: '抑制策略', roles: ['admin'] }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('../views/users/UserList.vue'),
        meta: { title: '用户管理', roles: ['admin'] }
      },
      {
        path: 'departments',
        name: 'Departments',
        component: () => import('../views/DepartmentList.vue'),
        meta: { title: '部门管理', roles: ['admin'] }
      },
      {
        path: 'schedules',
        name: 'Schedules',
        component: () => import('../views/SchedulePage.vue'),
        meta: { title: '排班管理', roles: ['admin'] }
      },
      {
        path: 'area-routing',
        name: 'AreaRouting',
        component: () => import('../views/AreaRoutingPage.vue'),
        meta: { title: '部门工单配置', roles: ['admin'] }
      },
      {
        path: 'algorithm-routing',
        name: 'AlgorithmRouting',
        component: () => import('../views/AlgorithmRoutingPage.vue'),
        meta: { title: '算法工单配置', roles: ['admin'] }
      },
      {
        path: 'user-groups',
        name: 'UserGroups',
        component: () => import('../views/GroupList.vue'),
        meta: { title: '用户组管理', roles: ['admin'] }
      },
      {
        path: 'permissions',
        name: 'Permissions',
        component: () => import('../views/permissions/PermissionConfig.vue'),
        meta: { title: '权限配置', roles: ['admin'] }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = (to.meta.title || 'WDOS') + ' - WDOS 管理后台'
  const token = localStorage.getItem('wdos_token')

  // 未登录 → 跳转登录页
  if (to.path !== '/login' && !token) {
    next('/login')
    return
  }

  // 登录后检查角色权限
  if (token && to.meta.roles) {
    const user = JSON.parse(localStorage.getItem('wdos_user') || '{}')
    const role = user.role || 'handler'
    if (!to.meta.roles.includes(role)) {
      // 无权限 → 重定向到首页大屏
      next('/dashboard')
      return
    }
  }

  next()
})

export default router
