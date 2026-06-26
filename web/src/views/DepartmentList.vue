<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <span style="font-size:14px;color:#666">管理部门和部门信息</span>
      <el-button type="primary" @click="openDialog()">+ 新增部门</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="部门名称" min-width="200" />
      <el-table-column label="操作" width="150">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openDialog(row)">编辑</el-button>
          <el-popconfirm title="确定删除该部门？" @confirm="del(row.id)">
            <template #reference><el-button text type="danger" size="small">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="editing?.id ? '编辑部门' : '新增部门'" v-model="dialogVisible" width="400px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="部门名称" required><el-input v-model="form.name" /></el-form-item>
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
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref(null)
const form = reactive({ name: '' })

async function fetch() {
  loading.value = true
  try {
    const res = await api.get('/departments')
    list.value = res.data?.list || []
  } finally { loading.value = false }
}

function openDialog(row) {
  editing.value = row
  form.name = row?.name || ''
  dialogVisible.value = true
}

async function save() {
  try {
    if (editing.value?.id) {
      await api.put(`/departments/${editing.value.id}`, form)
    } else {
      await api.post('/departments', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetch()
  } catch (e) { ElMessage.error('保存失败') }
}

async function del(id) {
  try {
    await api.delete(`/departments/${id}`)
    ElMessage.success('已删除')
    fetch()
  } catch (e) { ElMessage.error('删除失败') }
}

onMounted(fetch)
</script>
