<template>
  <div>
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
      <h2>排班管理</h2>
      <div>
        <el-date-picker v-model="viewDate" type="date" placeholder="选择日期" @change="fetch" format="YYYY-MM-DD" value-format="YYYY-MM-DD" style="margin-right:12px" />
        <el-upload :action="uploadUrl" :headers="authHeaders" :on-success="onUploadSuccess" :on-error="onUploadError" accept=".xlsx,.xls" style="display:inline-block">
          <el-button>📥 Excel导入</el-button>
        </el-upload>
      </div>
    </div>

    <el-card v-for="dept in deptGroups" :key="dept.name" style="margin-bottom:16px">
      <template #header>
        <span style="font-weight:600">🏢 {{ dept.name }}</span>
        <span style="margin-left:8px;color:#999;font-size:12px">{{ dept.members.length }} 人</span>
      </template>
      <el-table :data="dept.members" stripe size="small">
        <el-table-column prop="user_name" label="姓名" width="100" />
        <el-table-column label="班次" width="100">
          <template #default="{row}">
            <el-tag :type="row.shift_type==='day'?'warning':''" size="small">{{ row.shift_type==='day' ? '☀️ 白班' : '🌙 夜班' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="area" label="区域" min-width="140" />
        <el-table-column prop="shift_date" label="日期" width="110" />
        <el-table-column label="值班" width="80">
          <template #default="{row}">
            <el-tag v-if="row.is_on_call" type="danger" size="small">值班</el-tag>
            <span v-else style="color:#ccc">-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <div v-if="!loading && schedules.length===0" style="text-align:center;padding:40px;color:#999">当日暂无排班数据</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const schedules = ref([])
const users = ref([])
const loading = ref(false)
const token = localStorage.getItem('wdos_token')
const authHeaders = computed(() => ({ Authorization: `Bearer ${token}` }))
const uploadUrl = computed(() => api.defaults.baseURL + '/schedules/import-excel')

const today = new Date()
const localDate = `${today.getFullYear()}-${String(today.getMonth()+1).padStart(2,'0')}-${String(today.getDate()).padStart(2,'0')}`
const viewDate = ref(localDate)

async function loadUsers() {
  try {
    const res = await api.get('/users')
    users.value = res.data?.list || res.data || []
  } catch (_) {}
}

const deptGroups = computed(() => {
  const userMap = {}
  for (const u of users.value) {
    userMap[u.id] = { name: u.name || '未知', dept: u.department_name || '未分配部门', deptId: u.department_id || 0 }
  }
  const enriched = schedules.value.map(s => ({
    ...s,
    user_name: userMap[s.user_id]?.name || `用户${s.user_id}`,
    dept_name: userMap[s.user_id]?.dept || '未分配部门',
    dept_id: userMap[s.user_id]?.deptId || 0
  }))
  const groups = {}
  for (const s of enriched) {
    const key = `${s.dept_id}_${s.dept_name}`
    if (!groups[key]) groups[key] = { name: s.dept_name, members: [] }
    groups[key].members.push(s)
  }
  return Object.values(groups)
})

async function fetch() {
  loading.value = true
  try {
    const res = await api.get('/schedules', { params: { date: viewDate.value } })
    const list = []
    if (res.data) {
      for (const [, arr] of Object.entries(res.data)) {
        if (Array.isArray(arr)) list.push(...arr)
      }
    }
    schedules.value = list
  } finally { loading.value = false }
}

function onUploadSuccess(res) {
  ElMessage.success(res?.message || '导入成功')
  fetch()
}
function onUploadError() { ElMessage.error('导入失败') }

onMounted(() => { loadUsers(); fetch() })
</script>
