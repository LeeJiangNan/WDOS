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
        <el-popconfirm title="确定删除选中工单？" @confirm="batchDelete">
          <template #reference>
            <el-button type="danger" :disabled="selectedIds.length===0">🗑 批量删除 ({{ selectedIds.length }})</el-button>
          </template>
        </el-popconfirm>
      </div>
      <el-button @click="exportExcel">📥 导出 Excel</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe max-height="calc(100vh - 200px)" @selection-change="onSelectionChange" ref="tableRef">
      <el-table-column type="selection" width="40" />
      <el-table-column label="缩略图" width="80">
        <template #default="{row}">
          <el-image v-if="row.alarm_pic_url" :src="imgUrl(row.alarm_pic_url)" fit="cover" style="width:56px;height:42px;border-radius:4px;cursor:pointer" :preview-src-list="[imgUrl(row.alarm_pic_url)]" preview-teleported />
          <span v-else style="color:#ccc;font-size:20px">🖼</span>
        </template>
      </el-table-column>
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
          <span v-if="row.status === 'completed' && row.accepted_at && row.completed_at" style="color:#52c41a">{{ calcDuration(row) }}s</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="viewDetail(row)">详情</el-button>
          <el-button text size="small" @click="openTransfer(row)">转交</el-button>
          <el-popconfirm title="确定删除？" @confirm="doDelete(row)">
            <template #reference><el-button text type="danger" size="small">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- 详情弹窗 -->
    <el-dialog title="工单详情" v-model="detailVisible" width="700px">
      <el-descriptions v-if="detailOrder" :column="2" border>
        <el-descriptions-item label="工单编号">{{ detailOrder.order_no }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusType(detailOrder.status)">{{ statusLabel(detailOrder.status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="标题" :span="2">{{ detailOrder.title }}</el-descriptions-item>
        <el-descriptions-item label="报警地点">{{ detailOrder.camera_name }}</el-descriptions-item>
        <el-descriptions-item label="算法类型">{{ detailOrder.algorithm_name }}</el-descriptions-item>
        <el-descriptions-item label="处理人">{{ detailOrder.accepter_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="优先级">
          <el-tag size="small">{{ detailOrder.priority }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ detailOrder.created_at }}</el-descriptions-item>
        <el-descriptions-item label="完成时间">{{ detailOrder.completed_at || '-' }}</el-descriptions-item>
        <el-descriptions-item label="报警时间">{{ detailOrder.alarm_time }}</el-descriptions-item>
        <el-descriptions-item label="重复次数">{{ detailOrder.duplicate_count }}</el-descriptions-item>
        <el-descriptions-item label="处理结果" :span="2">{{ detailOrder.resolution || '暂无' }}</el-descriptions-item>
      </el-descriptions>
      <div v-if="detailOrder.alarm_pic_url" style="margin-top: 16px;">
        <el-divider content-position="left">报警图片</el-divider>
        <el-image
          :src="imgUrl(detailOrder.alarm_pic_url)"
          fit="contain"
          style="width: 100%; max-height: 500px;"
          :preview-src-list="[imgUrl(detailOrder.alarm_pic_url)]"
          preview-teleported
        />
      </div>
    </el-dialog>

    <!-- 转交弹窗 -->
    <el-dialog title="转交工单" v-model="transferVisible" width="400px">
      <el-form :model="transferForm" label-width="80px">
        <el-form-item label="目标用户ID" required><el-input-number v-model="transferForm.to_user_id" :min="1" style="width:100%" /></el-form-item>
        <el-form-item label="目标用户名"><el-input v-model="transferForm.to_user_name" /></el-form-item>
        <el-form-item label="转交原因"><el-input v-model="transferForm.reason" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="transferVisible = false">取消</el-button>
        <el-button type="primary" @click="doTransfer">确认转交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'

const list = ref([])
const loading = ref(false)
const selectedIds = ref([])
const tableRef = ref(null)
const status = ref('')
const keyword = ref('')

// 图片URL：拼接API地址（9090端口）
function imgUrl(path) {
  if (!path) return ''
  if (path.startsWith('http')) return path
  return location.protocol + '//' + location.hostname + ':9090/api/v1' + path
}

function statusType(s) {
  return { pending: 'warning', processing: '', completed: 'success' }[s] || 'info'
}
function statusLabel(s) {
  return { pending: '待接单', processing: '处理中', completed: '已完成' }[s] || s
}

async function fetch() {
  loading.value = true
  try {
    const ep = status.value ? `/work-orders/${status.value}` : '/work-orders'
    const res = await api.get(ep)
    list.value = res.data?.list || []
  } finally { loading.value = false }
}

// 详情弹窗
const detailVisible = ref(false)
const detailOrder = ref(null)

function viewDetail(row) {
  api.get(`/work-orders/${row.id}`).then(res => {
    detailOrder.value = res.data?.order || row
    detailVisible.value = true
  })
}

// 转交弹窗
const transferVisible = ref(false)
const transferOrder = ref(null)
const transferForm = reactive({ to_user_id: 0, to_user_name: '', reason: '' })

function openTransfer(row) {
  transferOrder.value = row
  transferForm.to_user_id = 0
  transferForm.to_user_name = ''
  transferForm.reason = ''
  transferVisible.value = true
}

async function doTransfer() {
  try {
    await api.post(`/work-orders/${transferOrder.value.id}/transfer`, transferForm)
    ElMessage.success('转交成功')
    transferVisible.value = false
    fetch()
  } catch (e) { ElMessage.error('转交失败') }
}

// 删除
async function doDelete(row) {
  try {
    await api.delete(`/work-orders/${row.id}`)
    ElMessage.success('已删除')
    fetch()
  } catch (e) { ElMessage.error('删除失败') }
}

function onSelectionChange(rows) { selectedIds.value = rows.map(r => r.id) }
function selectAll() { tableRef.value?.toggleAllSelection() }
async function batchDelete() {
  if (selectedIds.value.length === 0) return
  try {
    await api.post('/work-orders/batch-delete', { ids: selectedIds.value })
    ElMessage.success(`已删除 ${selectedIds.value.length} 条工单`)
    selectedIds.value = []
    fetch()
  } catch (e) { ElMessage.error('批量删除失败') }
}
function exportExcel() { ElMessage.info('导出功能开发中') }
function calcDuration(row) {
  if (!row.accepted_at || !row.completed_at) return '-'
  const a = new Date(row.accepted_at).getTime()
  const c = new Date(row.completed_at).getTime()
  return Math.round((c - a) / 1000)
}

onMounted(fetch)
</script>
