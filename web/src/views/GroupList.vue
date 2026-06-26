<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <span style="font-size:14px;color:#666">管理各部门下的用户班组</span>
      <el-button type="primary" @click="openDialog()">+ 新增用户组</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="组名称" min-width="200" />
      <el-table-column prop="department_id" label="所属部门ID" width="120" />
      <el-table-column label="操作" width="150">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openDialog(row)">编辑</el-button>
          <el-popconfirm title="确定删除该用户组？" @confirm="del(row.id)">
            <template #reference><el-button text type="danger" size="small">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="editing?.id ? '编辑用户组' : '新增用户组'" v-model="dialogVisible" width="400px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="组名称" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="所属部门" required>
          <el-select v-model="form.department_id" style="width:100%">
            <el-option v-for="d in depts" :key="d.id" :label="d.name" :value="d.id" />
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
import api from '../api'

const list = ref([])
const depts = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref(null)
const form = reactive({ name: '', department_id: 0 })

async function fetch() {
  loading.value = true
  try {
    const [grpRes, deptRes] = await Promise.all([
      api.get('/user-groups'),
      api.get('/departments')
    ])
    list.value = grpRes.data?.list || []
    depts.value = deptRes.data?.list || []
  } finally { loading.value = false }
}

function openDialog(row) {
  editing.value = row
  if (row) Object.assign(form, { name: row.name, department_id: row.department_id })
  else Object.assign(form, { name: '', department_id: depts.value[0]?.id || 0 })
  dialogVisible.value = true
}

async function save() {
  try {
    if (editing.value?.id) {
      await api.put(`/user-groups/${editing.value.id}`, form)
    } else {
      await api.post('/user-groups', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetch()
  } catch (e) { ElMessage.error('保存失败') }
}

async function del(id) {
  try {
    await api.delete(`/user-groups/${id}`)
    ElMessage.success('已删除')
    fetch()
  } catch (e) { ElMessage.error('删除失败') }
}

onMounted(fetch)
</script>
