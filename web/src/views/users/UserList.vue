<template>
  <div>
    <div style="display:flex;justify-content:space-between;margin-bottom:16px">
      <div style="display:flex;gap:8px">
        <el-input v-model="search" placeholder="搜索姓名/手机号" style="width:200px" clearable />
        <el-select v-model="roleFilter" placeholder="角色" style="width:130px" clearable>
          <el-option label="全部" value="" />
          <el-option label="一线人员" value="handler" />
          <el-option label="领班" value="supervisor" />
          <el-option label="经理" value="manager" />
          <el-option label="总监" value="director" />
        </el-select>
      </div>
      <el-button type="primary" @click="openDialog()">+ 新增用户</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="username" label="用户名" width="90" />
      <el-table-column prop="name" label="姓名" width="90" />
      <el-table-column prop="phone" label="手机号" width="120" />
      <el-table-column label="部门" width="100">
        <template #default="{row}">{{ deptName(row.department_id) }}</template>
      </el-table-column>
      <el-table-column label="角色" width="80">
        <template #default="{row}">
          <el-tag size="small">{{ roleLabel(row.role) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="password" label="密码" width="100" />
      <el-table-column label="微信绑定" width="100">
        <template #default="{row}">{{ row.open_id ? '✅ 已绑定' : '❌ 未绑定' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{row}"><el-tag :type="row.status==='active'?'success':'info'" size="small">{{ row.status === 'active' ? '在岗' : '禁用' }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="{row}">
          <el-button text type="primary" size="small" @click="openDialog(row)">编辑</el-button>
          <el-button text type="danger" size="small">禁用</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="editing?.id ? '编辑用户' : '新增用户'" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名" required><el-input v-model="form.username" placeholder="登录用，如 lizong" /></el-form-item>
        <el-form-item label="姓名" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="手机号" required><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="密码" required><el-input v-model="form.password" type="password" :placeholder="editing?.id ? '留空不修改' : '请输入密码'" /></el-form-item>
        <el-form-item label="角色"><el-select v-model="form.role" style="width:100%">
          <el-option label="一线人员" value="handler" />
          <el-option label="领班" value="supervisor" />
          <el-option label="经理" value="manager" />
          <el-option label="总监" value="director" />
          <el-option label="管理员" value="admin" />
        </el-select></el-form-item>
        <el-form-item label="部门">
          <el-select v-model="form.department_id" style="width:100%" clearable>
            <el-option v-for="d in deptOptions" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="用户组">
          <el-select v-model="form.group_id" style="width:100%" clearable>
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name + ' (部门' + g.department_id + ')'" :value="g.id" />
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
const deptOptions = ref([])
const groupOptions = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref(null)
const search = ref('')
const roleFilter = ref('')
const form = reactive({ username: '', name: '', phone: '', role: 'handler', password: '', department_id: 0, group_id: 0 })

function roleLabel(r) {
  return { handler: '一线', supervisor: '领班', manager: '经理', director: '总监', admin: '管理员' }[r] || r
}
function deptName(id) {
  const d = deptOptions.value.find(x => x.id === id)
  return d ? d.name : (id || '-')
}

async function fetch() {
  loading.value = true
  try {
    const [userRes, deptRes, grpRes] = await Promise.all([
      api.get('/users', { params: { role: roleFilter.value } }),
      api.get('/departments'),
      api.get('/user-groups')
    ])
    list.value = userRes.data?.list || []
    deptOptions.value = deptRes.data?.list || []
    groupOptions.value = grpRes.data?.list || []
  } finally { loading.value = false }
}

function openDialog(row) {
  editing.value = row
  if (row) Object.assign(form, {
    username: row.username || '', name: row.name, phone: row.phone, role: row.role, password: '',
    department_id: row.department_id || 0, group_id: row.group_id || 0
  })
  else {
    Object.assign(form, { username: '', name: '', phone: '', role: 'handler', password: '', department_id: 0, group_id: 0 })
  }
  dialogVisible.value = true
}

async function save() {
  try {
    if (editing.value?.id) {
      await api.put(`/users/${editing.value.id}`, form)
    } else {
      await api.post('/users', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetch()
  } catch (e) {}
}

onMounted(fetch)
</script>
