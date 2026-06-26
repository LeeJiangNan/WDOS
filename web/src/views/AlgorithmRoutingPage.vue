<template>
  <div>
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
      <h2>算法工单配置</h2>
      <el-button type="primary" @click="openAdd">新建规则</el-button>
    </div>
    <el-table :data="list" border stripe v-loading="loading">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="algorithm_pattern" label="算法名称" width="150" />
      <el-table-column prop="display_name" label="显示名称" width="120" />
      <el-table-column prop="category" label="分类" width="100" />
      <el-table-column label="部门" width="120">
        <template #default="{row}">{{ deptMap[row.department_id] || ('ID:'+row.department_id) }}</template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column label="启用" width="70">
        <template #default="{row}"><el-tag :type="row.is_active?'success':'info'" size="small">{{ row.is_active?'是':'否' }}</el-tag></template>
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

    <el-dialog :title="isEdit ? '编辑规则' : '新建规则'" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="算法名称" required>
          <el-input v-model="form.algorithm_pattern" placeholder="如 消防通道堵塞" />
          <span style="color:#999;font-size:12px">与CRIP推送的algorithm_name匹配</span>
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="form.display_name" placeholder="如 通道堵塞" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" placeholder="选择分类" style="width:100%" clearable>
            <el-option label="消防类" value="消防" />
            <el-option label="安防类" value="安防" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="分配部门" required>
          <el-select v-model="form.department_id" placeholder="选择部门" style="width:100%">
            <el-option v-for="d in departments" :key="d.id" :label="d.name" :value="d.id" />
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
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const form = ref({ algorithm_pattern: '', display_name: '', category: '', department_id: null, priority: 10, is_active: true })

const deptMap = computed(() => { const m = {}; departments.value.forEach(d => { m[d.id] = d.name }); return m })

async function fetchRules() { try { const r = await api.get('/algorithm-routing-rules'); list.value = r.data?.list || r.data || [] } catch (_) { list.value = [] } }
async function fetchDepts() { try { const r = await api.get('/departments'); departments.value = r.data?.list || r.data || [] } catch (_) { departments.value = [] } }

async function fetchAll() { loading.value = true; await Promise.all([fetchRules(), fetchDepts()]); loading.value = false }

function openAdd() { isEdit.value = false; editId.value = null; form.value = { algorithm_pattern: '', display_name: '', category: '', department_id: null, priority: 10, is_active: true }; dialogVisible.value = true }
function openEdit(row) {
  isEdit.value = true; editId.value = row.id
  form.value = { algorithm_pattern: row.algorithm_pattern || '', display_name: row.display_name || '', category: row.category || '', department_id: row.department_id, priority: row.priority ?? 10, is_active: row.is_active ?? true }
  dialogVisible.value = true
}

async function doSave() {
  saving.value = true
  try {
    if (isEdit.value) { await api.put(`/algorithm-routing-rules/${editId.value}`, form.value); ElMessage.success('已更新') }
    else { await api.post('/algorithm-routing-rules', form.value); ElMessage.success('已创建') }
    dialogVisible.value = false; await fetchRules()
  } catch (e) { ElMessage.error(e?.message || '保存失败') }
  saving.value = false
}

async function doDelete(id) { try { await api.delete(`/algorithm-routing-rules/${id}`); ElMessage.success('已删除'); await fetchRules() } catch (_) { ElMessage.error('删除失败') } }

onMounted(fetchAll)
</script>
