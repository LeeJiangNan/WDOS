<template>
  <div>
    <!-- 统计卡片 -->
    <el-row :gutter="16" style="margin-bottom:20px">
      <el-col :span="6" v-for="card in cards" :key="card.label">
        <el-card shadow="hover">
          <div style="display:flex;justify-content:space-between;align-items:center">
            <div>
              <div style="font-size:28px;font-weight:700" :style="{color:card.color}">{{ card.value }}</div>
              <div style="font-size:13px;color:#999">{{ card.label }}</div>
            </div>
            <div style="font-size:36px;opacity:.3">{{ card.icon }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 表格 -->
    <el-row :gutter="16">
      <el-col :span="12">
        <el-card header="📈 近7日报警趋势" style="margin-bottom:16px">
          <div style="height:220px;display:flex;align-items:center;justify-content:center;color:#999;background:#fafafa;border-radius:8px">
            ECharts 折线图（接入后渲染）
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card header="🔒 当前锁定点位" style="margin-bottom:16px">
          <el-empty description="暂无锁定点位" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const cards = ref([
  { label: '今日报警总数', value: '-', color: '#ff4d4f', icon: '🔔' },
  { label: '待处理工单', value: '-', color: '#1890ff', icon: '📝' },
  { label: '工单完成率', value: '-', color: '#52c41a', icon: '✅' },
  { label: '平均响应时间', value: '-', color: '#fa8c16', icon: '⏱️' }
])

onMounted(async () => {
  try {
    const res = await api.get('/stats/my-overview')
    if (res.data?.today) {
      cards.value[1].value = res.data.today.pending_count
      cards.value[2].value = res.data.today.completed_count
    }
  } catch (e) { /* 接口未就绪 */ }
})
</script>
