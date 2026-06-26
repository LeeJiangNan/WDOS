<template>
  <div>
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
      <h2>部门工单配置</h2>
      <el-button type="primary" @click="openAdd">新建规则</el-button>
    </div>

    <el-table :data="list" border stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="camera_group_pattern" label="匹配模式" width="140" />
      <el-table-column prop="area_name" label="区域名称" width="140" />
      <el-table-column label="部门" width="120">
        <template #default="{row}">{{ deptMap[row.department_id] || ('ID:'+row.department_id) }}</template>
      </el-table-column>
      <el-table-column label="处理班组" width="120">
        <template #default="{row}">{{ row.handler_group_id ? (groupMap[row.handler_group_id] || 'ID:'+row.handler_group_id) : '-' }}</template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column label="启用" width="70">
        <template #default="{row}">
          <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openEdit(row)">编辑</el-button>
          <el-popconfirm title="确定删除？" @confirm="doDelete(row.id)">
            <template #reference><el-button text type="danger" size="small">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- 表单弹窗 -->
    <el-dialog :title="isEdit ? '编辑规则' : '新建规则'" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="匹配模式" required>
          <el-input v-model="form.camera_group_pattern" placeholder="如 B1* 匹配 B1开头" />
          <span style="color:#999;font-size:12px">*前缀 / *后缀 / *包含*</span>
        </el-form-item>
        <el-form-item label="区域名称" required>
          <el-input v-model="form.area_name" placeholder="如 B1停车场/机房" />
        </el-form-item>
        <el-form-item label="分配部门" required>
          <el-select v-model="form.department_id" placeholder="选择部门" style="width:100%">
            <el-option v-for="d in departments" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理班组">
          <el-select v-model="form.handler_group_id" placeholder="可选" style="width:100%" clearable>
            <el-option v-for="g in userGroups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="0" :max="100" />
          <span style="color:#999;font-size:12px;margin-left:8px">越大越优先</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="doSave" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const list = ref([])
const departments = ref([])
const userGroups = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)

const form = ref({
  camera_group_pattern: '',
  area_name: '',
  department_id: null,
  handler_group_id: 0,
  priority: 10,
  is_active: true,
})

// 用 map 快速查名字，比 computed 更稳
const deptMap = computed(() => {
  const m = {}
  departments.value.forEach(d => { m[d.id] = d.name })
  return m
})
const groupMap = computed(() => {
  const m = {}
  userGroups.value.forEach(g => { m[g.id] = g.name })
  return m
})

async function fetchRules() {
  try {
    const res = await api.get('/area-routing-rules')
    list.value = res.data?.list || res.data || []
  } catch (e) {
    console.error('加载规则失败:', e)
    list.value = []
  }
}

async function fetchDepts() {
  try {
    const res = await api.get('/departments')
    departments.value = res.data?.list || res.data || []
  } catch (e) {
    console.error('加载部门失败:', e)
    departments.value = []
  }
}

async function fetchGroups() {
  try {
    const res = await api.get('/user-groups')
    userGroups.value = res.data?.list || res.data || []
  } catch (e) {
    console.error('加载用户组失败:', e)
    userGroups.value = []
  }
}

async function fetchAll() {
  loading.value = true
  await Promise.all([fetchRules(), fetchDepts(), fetchGroups()])
  loading.value = false
}

function openAdd() {
  isEdit.value = false
  editId.value = null
  form.value = { camera_group_pattern: '', area_name: '', department_id: null, handler_group_id: 0, priority: 10, is_active: true }
  dialogVisible.value = true
}

function openEdit(row) {
  isEdit.value = true
  editId.value = row.id
  // 只拷贝可编辑字段，排除 id、created_at 等
  form.value = {
    camera_group_pattern: row.camera_group_pattern || '',
    area_name: row.area_name || '',
    department_id: row.department_id,
    handler_group_id: row.handler_group_id || 0,
    priority: row.priority ?? 10,
    is_active: row.is_active ?? true,
  }
  dialogVisible.value = true
}

async function doSave() {
  saving.value = true
  try {
    if (isEdit.value) {
      await api.put(`/area-routing-rules/${editId.value}`, form.value)
      ElMessage.success('已更新')
    } else {
      await api.post('/area-routing-rules', form.value)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await fetchRules()
  } catch (e) {
    ElMessage.error(e?.message || '保存失败')
  }
  saving.value = false
}

async function doDelete(id) {
  try {
    await api.delete(`/area-routing-rules/${id}`)
    ElMessage.success('已删除')
    await fetchRules()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

onMounted(fetchAll)
</script>
