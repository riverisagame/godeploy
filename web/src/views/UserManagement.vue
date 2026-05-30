<template>
  <div class="user-management-container">
    <header class="header">
      <div class="logo-area">
        <el-icon size="24" color="#00b4d8"><Platform /></el-icon>
        <span class="system-name" @click="router.push('/')" style="cursor:pointer;">GoDeployer 控制台</span>
        <span class="page-title">/ 用户管理</span>
      </div>
      <div class="user-area">
        <el-button @click="router.push('/')">返回控制台</el-button>
      </div>
    </header>

    <main class="main-content">
      <div class="table-header">
        <h3>系统用户列表</h3>
        <el-button type="primary" @click="openDialog()">新增用户</el-button>
      </div>

      <el-table :data="users" style="width: 100%" v-loading="loading" border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="role" label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="getRoleTag(row.role)">{{ row.role }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="permitted_projects" label="允许访问的项目" min-width="150" />
        <el-table-column prop="bound_git_authors" label="绑定 Git 提交人" min-width="150" />
        <el-table-column prop="restrict_git_authors" label="限制 Git 提交人" width="140">
          <template #default="{ row }">
            <el-switch v-model="row.restrict_git_authors" disabled />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteUser(row)" :disabled="row.username === 'admin'">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </main>

    <!-- 新增/编辑用户对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新增用户'" width="500px">
      <el-form :model="formData" label-width="120px" :rules="rules" ref="formRef">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="formData.username" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="密码" prop="password" :required="!isEdit">
          <el-input v-model="formData.password" type="password" show-password :placeholder="isEdit ? '留空表示不修改' : ''" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="formData.role">
            <el-option label="Admin" value="admin" />
            <el-option label="Deployer" value="deployer" />
            <el-option label="Viewer" value="viewer" />
          </el-select>
        </el-form-item>
        <el-form-item label="允许访问项目" prop="permitted_projects">
          <el-input v-model="formData.permitted_projects" placeholder="*表示全部，多个逗号分隔" />
        </el-form-item>
        <el-form-item label="绑定Git提交人" prop="bound_git_authors">
          <el-input v-model="formData.bound_git_authors" placeholder="多个邮箱逗号分隔" />
        </el-form-item>
        <el-form-item label="限制仅限该提交人" prop="restrict_git_authors">
          <el-switch v-model="formData.restrict_git_authors" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Platform } from '@element-plus/icons-vue'

const router = useRouter()

const users = ref<any[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()

const formData = ref({
  username: '',
  password: '',
  role: 'viewer',
  permitted_projects: '*',
  bound_git_authors: '',
  restrict_git_authors: false
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/users')
    users.value = res.data
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '获取用户失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchUsers()
})

const getRoleTag = (role: string) => {
  switch (role) {
    case 'admin': return 'danger'
    case 'deployer': return 'warning'
    default: return 'info'
  }
}

const openDialog = (row?: any) => {
  if (row) {
    isEdit.value = true
    formData.value = {
      username: row.username,
      password: '',
      role: row.role,
      permitted_projects: row.permitted_projects || '*',
      bound_git_authors: row.bound_git_authors || '',
      restrict_git_authors: row.restrict_git_authors || false
    }
  } else {
    isEdit.value = false
    formData.value = {
      username: '',
      password: '',
      role: 'viewer',
      permitted_projects: '*',
      bound_git_authors: '',
      restrict_git_authors: false
    }
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (valid) {
      if (!isEdit.value && !formData.value.password) {
        ElMessage.error('新建用户必须输入密码')
        return
      }
      try {
        if (isEdit.value) {
          await axios.put(`/api/users/${formData.value.username}`, formData.value)
          ElMessage.success('更新成功')
        } else {
          await axios.post('/api/users', formData.value)
          ElMessage.success('新建成功')
        }
        dialogVisible.value = false
        fetchUsers()
      } catch (e: any) {
        ElMessage.error(e.response?.data?.error || '操作失败')
      }
    }
  })
}

const deleteUser = (row: any) => {
  ElMessageBox.confirm(`确定删除用户 ${row.username} 吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    try {
      await axios.delete(`/api/users/${row.username}`)
      ElMessage.success('删除成功')
      fetchUsers()
    } catch (e: any) {
      ElMessage.error(e.response?.data?.error || '删除失败')
    }
  }).catch(() => {})
}
</script>

<style scoped>
.user-management-container {
  min-height: 100vh;
  background: #10121a;
  display: flex;
  flex-direction: column;
  color: #c9d1d9;
}

.header {
  height: 60px;
  background: #1a1f2c;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.logo-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.system-name {
  font-size: 18px;
  font-weight: 600;
  color: #fff;
}

.page-title {
  font-size: 16px;
  color: #8a99ad;
  margin-left: 8px;
}

.main-content {
  flex: 1;
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.table-header h3 {
  margin: 0;
  color: #ffffff;
}

:deep(.el-table) {
  --el-table-bg-color: #161a23;
  --el-table-tr-bg-color: #161a23;
  --el-table-header-bg-color: #1c2128;
  --el-table-border-color: rgba(255, 255, 255, 0.06);
  --el-table-text-color: #c9d1d9;
  --el-table-header-text-color: #8a99ad;
}

:deep(.el-input__wrapper) {
  background-color: #121824;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
}
</style>
