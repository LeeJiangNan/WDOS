<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex;align-items:center;gap:16px">
          <span style="font-weight:600">🔐 角色权限配置</span>
          <el-select v-model="selectedRole" style="width:180px" @change="loadPerms">
            <el-option label="一线人员 (handler)" value="handler" />
            <el-option label="领班 (supervisor)" value="supervisor" />
            <el-option label="经理 (manager)" value="manager" />
            <el-option label="总监 (director)" value="director" />
          </el-select>
        </div>
      </template>

      <el-table :data="permList" stripe>
        <el-table-column prop="module" label="权限模块" width="180" />
        <el-table-column label="查看" width="80" align="center">
          <template #default="{row}"><el-checkbox v-model="row.view" /></template>
        </el-table-column>
        <el-table-column label="编辑" width="80" align="center">
          <template #default="{row}"><el-checkbox v-model="row.edit" /></template>
        </el-table-column>
        <el-table-column label="删除" width="80" align="center">
          <template #default="{row}"><el-checkbox v-model="row.delete" /></template>
        </el-table-column>
        <el-table-column label="数据范围" width="120">
          <template #default="{row}">
            <el-select v-model="row.scope" size="small">
              <el-option label="个人" value="self" />
              <el-option label="组内" value="group" />
              <el-option label="部门" value="department" />
              <el-option label="全局" value="global" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="备注" min-width="180">
          <template #default="{row}">
            <span v-if="row.note" style="color:#ff976a;font-size:12px">{{ row.note }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div style="margin-top:16px">
        <el-button type="primary" @click="savePerms">保存权限</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'

const selectedRole = ref('supervisor')
const permList = ref([
  { module: '工单模板管理', view: false, edit: false, delete: false, scope: 'group' },
  { module: '工单数据管理', view: true, edit: false, delete: false, scope: 'group' },
  { module: '抑制策略配置', view: false, edit: false, delete: false, scope: 'department' },
  { module: '区域路由配置', view: false, edit: false, delete: false, scope: 'department' },
  { module: 'SLA 策略配置', view: false, edit: false, delete: false, scope: 'global' },
  { module: '用户管理', view: true, edit: false, delete: false, scope: 'group' },
  { module: '排班管理', view: true, edit: true, delete: false, scope: 'group', note: '排班修改需经理级别' },
  { module: '排班导入', view: true, edit: false, delete: false, scope: 'group', note: '需排班权限' },
  { module: '统计报表', view: true, edit: false, delete: false, scope: 'group' },
])

async function loadPerms() {
  // 实际项目从 API 加载: api.get('/permissions/roles', { params: { role: selectedRole.value } })
}

async function savePerms() {
  try {
    await api.put(`/permissions/roles/${selectedRole.value}`, { permissions: permList.value })
    ElMessage.success('权限已保存')
  } catch (e) {}
}
</script>
