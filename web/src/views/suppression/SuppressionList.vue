<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <el-input v-model="search" placeholder="搜索策略名称" style="width:220px" clearable />
      <el-button type="primary" @click="openDialog()">+ 创建策略</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="策略名称" min-width="180" />
      <el-table-column prop="camera_group_filter" label="适用区域" width="150" />
      <el-table-column label="锁定模式" width="120">
        <template #default="{row}">{{ row.lock_mode === 'algo_only' ? '同算法锁定' : '全点位锁定' }}</template>
      </el-table-column>
      <el-table-column prop="lock_after_seconds" label="锁定时长" width="100">
        <template #default="{row}">{{ row.lock_after_seconds }}s</template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{row}"><el-tag :type="row.is_active?'success':'info'" size="small">{{ row.is_active ? '启用' : '停用' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openDialog(row)">编辑</el-button>
          <el-button text type="danger" size="small">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="editing?.id ? '编辑策略' : '创建策略'" v-model="dialogVisible" width="600px">
      <el-form :model="form" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12"><el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="摄像头分组"><el-input v-model="form.camera_group_filter" placeholder="如 parking_*" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="锁定模式">
            <el-select v-model="form.lock_mode" style="width:100%"><el-option label="同算法锁定" value="algo_only" /><el-option label="全点位锁定" value="full_camera" /></el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="锁定时长(秒)"><el-input-number v-model="form.lock_after_seconds" :min="30" :max="7200" style="width:100%" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="最大锁定(秒)"><el-input-number v-model="form.max_lock_seconds" :min="60" :max="86400" style="width:100%" /></el-form-item></el-col>
        </el-row>
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
const form = reactive({ name: '', camera_group_filter: '', lock_mode: 'algo_only', lock_after_seconds: 300, max_lock_seconds: 3600 })

async function fetch() {
  loading.value = true
  try { const res = await api.get('/suppression-rules'); list.value = res.data?.list || [] } finally { loading.value = false }
}
function openDialog(row) {
  editing.value = row
  if (row) Object.assign(form, row)
  else { form.name = ''; form.camera_group_filter = ''; form.lock_mode = 'algo_only'; form.lock_after_seconds = 300; form.max_lock_seconds = 3600 }
  dialogVisible.value = true
}
async function save() {
  try {
    if (editing.value?.id) await api.put(`/suppression-rules/${editing.value.id}`, form)
    else await api.post('/suppression-rules', form)
    ElMessage.success('保存成功'); dialogVisible.value = false; fetch()
  } catch (e) {}
}
onMounted(fetch)
</script>
