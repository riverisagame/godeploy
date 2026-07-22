<template>
  <div class="user-list">
    <div class="header-actions">
      <h2>用户管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>新建用户
      </el-button>
    </div>

    <el-card shadow="never" class="table-card">
      <el-table :data="users" v-loading="loading" style="width: 100%" class="custom-table">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" min-width="150" />
        <el-table-column prop="role" label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" effect="dark" size="small">
              {{ row.role === 'admin' ? '管理员' : '开发者' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showCreateDialog" title="新建用户" width="500px" custom-class="dark-dialog">
      <el-form :model="createForm" ref="formRef" label-width="80px">
        <el-form-item label="用户名" prop="username" :rules="[{ required: true, message: '请输入用户名', trigger: 'blur' }]">
          <el-input v-model="createForm.username" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password" :rules="[{ required: true, message: '请输入密码', trigger: 'blur' }]">
          <el-input v-model="createForm.password" type="password" placeholder="密码" show-password />
        </el-form-item>
        <el-form-item label="角色" prop="role" :rules="[{ required: true, message: '请选择角色', trigger: 'change' }]">
          <el-select v-model="createForm.role" placeholder="选择角色" style="width: 100%;">
            <el-option label="开发者" value="developer" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" :loading="creating" @click="handleCreate">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const users = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)
const formRef = ref()

const createForm = reactive({
  username: '',
  password: '',
  role: 'developer'
})

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await api.getUsers()
    const data = (res as any).data || res
    users.value = data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || err.message || '获取用户列表失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (valid) {
      creating.value = true
      try {
        await api.createUser({
          username: createForm.username,
          password: createForm.password,
          role: createForm.role
        })
        ElMessage.success('用户创建成功')
        showCreateDialog.value = false
        // reset form
        createForm.username = ''
        createForm.password = ''
        createForm.role = 'developer'
        fetchUsers()
      } catch (err: any) {
        ElMessage.error(err.response?.data?.message || err.message || '创建用户失败')
      } finally {
        creating.value = false
      }
    }
  })
}

onMounted(() => {
  fetchUsers()
})
</script>

<style scoped>
.user-list {
  padding: 24px;
}
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.header-actions h2 {
  margin: 0;
  color: var(--text-primary);
  font-weight: 500;
}
.table-card {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
}
.custom-table {
  --el-table-border-color: var(--border-color);
  --el-table-header-bg-color: rgba(255, 255, 255, 0.02);
  --el-table-row-hover-bg-color: rgba(255, 255, 255, 0.04);
  background-color: transparent;
}
:deep(.el-table th.el-table__cell) {
  background-color: var(--el-table-header-bg-color);
  color: #94A3B8;
  font-weight: 600;
}
:deep(.el-table tr) {
  background-color: transparent;
}
:deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid var(--border-color);
  color: var(--text-primary);
}
:deep(.el-dialog.dark-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
}
:deep(.el-dialog__title) {
  color: var(--text-primary);
}
:deep(.el-form-item__label) {
  color: var(--text-primary);
}
</style>
