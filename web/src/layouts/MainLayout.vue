<template>
  <el-container style="height:100vh">
    <!-- 侧边栏 -->
    <el-aside width="220px" style="background:#001529">
      <div class="logo">🏬 WDOS 管理后台</div>
      <el-menu
        :default-active="activeMenu"
        background-color="#001529"
        text-color="rgba(255,255,255,.65)"
        active-text-color="#fff"
        router
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataBoard /></el-icon> 首页大屏
        </el-menu-item>
        <el-menu-item index="/templates">
          <el-icon><Document /></el-icon> 工单模板
        </el-menu-item>
        <el-menu-item index="/work-orders">
          <el-icon><Tickets /></el-icon> 工单数据
        </el-menu-item>
        <el-menu-item index="/suppression">
          <el-icon><Bell /></el-icon> 抑制策略
        </el-menu-item>
        <el-menu-item index="/users">
          <el-icon><User /></el-icon> 用户管理
        </el-menu-item>
        <el-menu-item index="/permissions">
          <el-icon><Lock /></el-icon> 权限配置
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主区域 -->
    <el-container>
      <el-header style="height:56px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #f0f0f0">
        <span style="font-weight:600">{{ $route.meta.title }}</span>
        <div style="display:flex;align-items:center;gap:12px">
          <span>{{ user?.name || 'admin' }}</span>
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
