<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <div style="display:flex;gap:8px">
        <el-input v-model="keyword" placeholder="搜索工单编号/地点" style="width:220px" clearable />
        <el-select v-model="status" style="width:130px">
          <el-option label="全部状态" value="" />
          <el-option label="待接单" value="pending" />
          <el-option label="处理中" value="processing" />
          <el-option label="已完成" value="completed" />
        </el-select>
      </div>
      <el-button @click="exportExcel">📥 导出 Excel</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe max-height="calc(100vh - 200px)">
      <el-table-column prop="order_no" label="工单编号" width="180" />
      <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
      <el-table-column prop="camera_name" label="报警地点" width="160" />
      <el-table-column label="状态" width="100">
        <template #default="{row}">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="accepter_name" label="处理人" width="100" />
      <el-table-column prop="created_at" label="创建时间" width="160" />
      <el-table-column label="耗时" width="100">
        <template #default="{row}">
          <span v-if="row.status === 'completed'" style="color:#52c41a">{{ row.duplicate_count || '-' }}s</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="viewDetail(row)">详情</el-button>
          <el-button text size="small" @click="transfer(row)">转交</el-button>
          <el-button text type="danger" size="small">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../../api'

const list = ref([])
const loading = ref(false)
const status = ref('')
const keyword = ref('')

function statusType(s) {
  return { pending: 'warning', processing: '', completed: 'success' }[s] || 'info'
}
function statusLabel(s) {
  return { pending: '待接单', processing: '处理中', completed: '已完成' }[s] || s
}

async function fetch() {
  loading.value = true
  try {
    const ep = status.value ? `/work-orders/${status.value}` : '/work-orders/pending'
    const res = await api.get(ep)
    list.value = res.data?.list || []
  } finally { loading.value = false }
}

function viewDetail(row) { /* 弹窗显示工单详情 */ }
function transfer(row) { /* 转交逻辑 */ }
function exportExcel() { /* 导出 */ }

onMounted(fetch)
</script>
