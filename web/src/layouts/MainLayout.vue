<template>
  <el-container style="height:100vh">
    <el-aside width="220px" style="background:#001529">
      <div class="logo">🏬 WDOS 管理后台</div>
      <el-menu
        :default-active="activeMenu"
        background-color="#001529"
        text-color="rgba(255,255,255,.65)"
        active-text-color="#fff"
        router
      >
        <template v-for="item in menuItems" :key="item.index">
          <el-menu-item :index="item.index">
            <el-icon><component :is="item.icon" /></el-icon> {{ item.label }}
          </el-menu-item>
        </template>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header style="height:56px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #f0f0f0">
        <span style="font-weight:600">{{ $route.meta.title }}</span>
        <div style="display:flex;align-items:center;gap:12px">
          <span>{{ user?.name || 'admin' }} ({{ roleLabel }})</span>
          <el-button text @click="logout">退出</el-button>
        </div>
      </el-header>
      <el-main style="background:#f0f2f5">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const activeMenu = computed(() => route.path)
const user = JSON.parse(localStorage.getItem('wdos_user') || 'null')
const role = user?.role || 'handler'

const roleLabel = { admin:'管理员', director:'总监', manager:'经理', supervisor:'领班', handler:'一线人员' }[role] || role

// 所有菜单项
const allMenus = [
  { index: '/dashboard', label: '首页大屏', icon: 'DataBoard', roles: ['admin','director','manager','supervisor','handler'] },
  { index: '/templates', label: '工单模板', icon: 'Document', roles: ['admin'] },
  { index: '/work-orders', label: '工单数据', icon: 'Tickets', roles: ['admin'] },
  { index: '/suppression', label: '抑制策略', icon: 'Bell', roles: ['admin'] },
  { index: '/departments', label: '部门管理', icon: 'OfficeBuilding', roles: ['admin'] },
  { index: '/schedules', label: '排班管理', icon: 'Calendar', roles: ['admin'] },
  { index: '/area-routing', label: '部门工单配置', icon: 'Connection', roles: ['admin'] },
  { index: '/algorithm-routing', label: '算法工单配置', icon: 'Guide', roles: ['admin'] },
  { index: '/user-groups', label: '用户组管理', icon: 'Grid', roles: ['admin'] },
  { index: '/users', label: '用户管理', icon: 'User', roles: ['admin'] },
  { index: '/permissions', label: '权限配置', icon: 'Lock', roles: ['admin'] },
]

// 当前角色可见菜单
const menuItems = computed(() => allMenus.filter(m => m.roles.includes(role)))

function logout() {
  localStorage.removeItem('wdos_token')
  localStorage.removeItem('wdos_user')
  router.push('/login')
}
</script>

<style scoped>
.logo {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  padding: 18px 16px;
  border-bottom: 1px solid rgba(255,255,255,.1);
}
</style>
