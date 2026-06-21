<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <div style="display:flex;gap:8px">
        <el-input v-model="search" placeholder="搜索模板名称" style="width:200px" clearable />
        <el-select v-model="statusFilter" placeholder="状态" style="width:120px" clearable>
          <el-option label="全部" value="" />
          <el-option label="启用" value="active" />
          <el-option label="停用" value="inactive" />
        </el-select>
      </div>
      <el-button type="primary" @click="openDialog()">+ 创建模板</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="模板名称" min-width="180" />
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column prop="flow_id" label="关联流程" width="120" />
      <el-table-column label="状态" width="100">
        <template #default="{row}">
          <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openDialog(row)">编辑</el-button>
          <el-button text :type="row.is_active ? 'danger' : 'success'" size="small" @click="toggle(row)">
            {{ row.is_active ? '停用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建/编辑弹窗 -->
    <el-dialog :title="editing?.id ? '编辑模板' : '创建模板'" v-model="dialogVisible" width="600px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：行人闯入处理工单" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="关联流程">
          <el-select v-model="form.flow_id" style="width:100%">
            <el-option label="标准派单处理" :value="1" />
            <el-option label="派单+审核" :value="2" />
            <el-option label="三级分级处理" :value="3" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
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
const dialogVisible = ref(false)
const editing = ref(null)
const search = ref('')
const statusFilter = ref('')
const form = reactive({ name: '', description: '', flow_id: 1 })

async function fetch() {
  loading.value = true
  try {
    const res = await api.get('/templates', { params: { status: statusFilter.value } })
    list.value = res.data?.list || []
  } finally { loading.value = false }
}

function openDialog(row) {
  editing.value = row
  if (row) Object.assign(form, row)
  else { form.name = ''; form.description = ''; form.flow_id = 1 }
  dialogVisible.value = true
}

async function save() {
  try {
    if (editing.value?.id) {
      await api.put(`/templates/${editing.value.id}`, form)
    } else {
      await api.post('/templates', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetch()
  } catch (e) {}
}

async function toggle(row) {
  try {
    await api.post(`/templates/${row.id}/toggle`, { is_active: !row.is_active })
    ElMessage.success(row.is_active ? '已停用' : '已启用')
    fetch()
  } catch (e) {}
}

onMounted(fetch)
</script>
