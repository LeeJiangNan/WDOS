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

    <!-- 趋势图 + 锁定点位 -->
    <el-row :gutter="16">
      <el-col :span="14">
        <el-card header="📈 近7日报警趋势" style="margin-bottom:16px">
          <div ref="chartRef" style="height:260px"></div>
          <div v-if="trendError" style="text-align:center;color:#999;padding:40px">{{ trendError }}</div>
        </el-card>
      </el-col>
      <el-col :span="10">
        <el-card header="🔒 当前锁定点位" style="margin-bottom:16px">
          <el-table v-if="locks.length > 0" :data="locks" size="small" max-height="260px">
            <el-table-column prop="camera_name" label="摄像头" show-overflow-tooltip />
            <el-table-column prop="algorithm_name" label="算法" width="100" />
            <el-table-column label="锁定模式" width="90">
              <template #default="{row}">
                <el-tag size="small" type="danger">{{ row.lock_mode === 'algo_only' ? '仅算法' : row.lock_mode }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="locked_at" label="锁定时间" width="160" />
          </el-table>
          <el-empty v-else description="暂无锁定点位" />
          <div v-if="locks.length > 0" style="margin-top:8px;font-size:12px;color:#999">
            锁定由 SLA 二级上报自动触发，转交工单后自动解锁
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, onUnmounted } from 'vue'
import api from '../api'
import * as echarts from 'echarts'

const chartRef = ref(null)
const trendError = ref('')
const locks = ref([])
let chart = null

const cards = ref([
  { label: '今日报警总数', value: '-', color: '#ff4d4f', icon: '🔔' },
  { label: '待处理工单', value: '-', color: '#1890ff', icon: '📝' },
  { label: '工单完成率', value: '-', color: '#52c41a', icon: '✅' },
  { label: '平均响应时间', value: '-', color: '#fa8c16', icon: '⏱️' }
])

onMounted(async () => {
  try {
    const res = await api.get('/stats/my-overview')
    const d = res.data
    if (d) {
      cards.value[0].value = d.total_alarms ?? d.pending_orders ?? 0
      cards.value[1].value = (d.total_orders ?? 0) - (d.completed_orders ?? 0)
      cards.value[2].value = Math.round((d.completion_rate ?? 0) * 100) + '%'
      cards.value[3].value = (d.overtime_orders ?? 0) + ' 超时'
    }
  } catch (e) { /* ignore */ }

  // 加载趋势
  try {
    const tRes = await api.get('/stats/trend', { params: { days: 7 } })
    const trend = tRes.data?.list || tRes.data || []
    if (trend.length > 0) {
      await nextTick()
      renderChart(trend)
    } else {
      trendError.value = '暂无趋势数据'
    }
  } catch (e) {
    trendError.value = '趋势数据加载失败'
  }

  // 加载锁定点位
  try {
    const lRes = await api.get('/work-orders', { params: { locked: 'true' } })
    const list = lRes.data?.list || lRes.data || []
    locks.value = list.filter(o => o.is_locked)
  } catch (_) {}
})

function renderChart(data) {
  if (!chartRef.value) return
  if (chart) chart.dispose()
  chart = echarts.init(chartRef.value)
  chart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: data.map(d => d.date?.substring(5) || d.date), axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{
      name: '报警数', type: 'line', data: data.map(d => d.alarm_count || 0),
      smooth: true, symbol: 'circle', symbolSize: 8,
      itemStyle: { color: '#ff4d4f' },
      areaStyle: { color: 'rgba(255,77,79,0.1)' }
    }]
  })
  window.addEventListener('resize', () => chart?.resize())
}

onUnmounted(() => { chart?.dispose() })
</script>
