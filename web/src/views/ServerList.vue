<template>
  <div class="server-list">
    <div class="header-actions">
      <div>
        <h2>服务器管理</h2>
        <p class="subtitle">管理所有可用于部署的物理机和虚拟机资源</p>
      </div>
      <el-button v-if="admin" type="primary" @click="dialogVisible = true" size="large">新建服务器</el-button>
    </div>

    <el-card class="dense-table-card" shadow="never">
      <el-table :data="servers" style="width: 100%" class="custom-table" :empty-text="'暂无服务器数据'">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column label="服务器名称">
          <template #default="scope">
            <div class="server-name-cell">
              <el-icon class="server-icon"><Monitor /></el-icon>
              <span>{{ scope.row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="连接信息">
          <template #default="scope">
            <div class="connection-info">
              <el-tag size="small" type="primary" effect="dark" class="mono-tag">{{ scope.row.user }}@{{ scope.row.ip }}</el-tag>
              <el-tag size="small" type="info" class="mono-tag">Port: {{ scope.row.port }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="认证方式" width="120">
          <template #default="scope">
            <el-tag size="small" :type="scope.row.key_path ? 'success' : 'warning'" effect="plain">
              {{ scope.row.key_path ? 'Key Auth' : 'Password' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="admin" label="操作" width="160" align="center">
          <template #default="scope">
            <el-button size="small" type="primary" text bg @click="openEdit(scope.row)">编辑</el-button>
            <el-button size="small" type="danger" text bg @click="handleDelete(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create / Edit Server Dialog -->
    <el-dialog :title="isEdit ? '编辑服务器' : '新建服务器'" v-model="dialogVisible" width="480px" destroy-on-close class="custom-dialog" @closed="resetForm">
      <el-form :model="form" label-position="top" class="custom-form">
        <el-form-item label="服务器名称">
          <el-input v-model="form.name" placeholder="例如: web-prod-01" size="large"></el-input>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="16">
            <el-form-item label="IP 地址">
              <el-input v-model="form.ip" placeholder="例如: 192.168.1.100" size="large"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="SSH 端口">
              <el-input-number v-model="form.port" :min="1" :max="65535" size="large" style="width: 100%"></el-input-number>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="SSH 用户名">
              <el-input v-model="form.user" placeholder="root" size="large"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="私钥路径">
              <el-input v-model="form.key_path" placeholder="~/.ssh/id_rsa" size="large"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="saving">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Monitor } from '@element-plus/icons-vue'
import type { Server } from '../types'
import { isAdmin } from '../utils/auth'

const admin = computed(() => isAdmin())

const servers = ref<Server[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const saving = ref(false)

const form = ref({
  name: '',
  ip: '',
  port: 22,
  user: 'root',
  key_path: '~/.ssh/id_rsa'
})

const fetchServers = async () => {
  try {
    const res = await api.getServers()
    servers.value = res.data
  } catch (e) {
    ElMessage.error('获取服务器列表失败')
  }
}

const resetForm = () => {
  isEdit.value = false
  editId.value = null
  form.value = {
    name: '',
    ip: '',
    port: 22,
    user: 'root',
    key_path: '~/.ssh/id_rsa'
  }
}

const openEdit = (server: Server) => {
  isEdit.value = true
  editId.value = server.id
  form.value = {
    name: server.name,
    ip: server.ip,
    port: server.port,
    user: server.user,
    key_path: server.key_path
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!form.value.name || !form.value.ip) {
    ElMessage.warning('请填写名称和IP')
    return
  }
  
  saving.value = true
  try {
    if (isEdit.value && editId.value) {
      await api.updateServer(editId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await api.createServer(form.value)
      ElMessage.success('创建成功')
    }
    fetchServers()
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.response?.data || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    saving.value = false
  }
}

const handleDelete = async (server: Server) => {
  try {
    await ElMessageBox.confirm(`确定要删除服务器 ${server.name} 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await api.deleteServer(server.id)
    ElMessage.success('删除成功')
    await fetchServers()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data || '删除失败')
    }
  }
}

onMounted(() => {
  fetchServers()
})
</script>

<style scoped>
.server-list {
  padding: 8px;
}
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}
h2 {
  margin: 0 0 8px 0;
  color: var(--text-primary);
  font-size: 28px;
  letter-spacing: -0.5px;
}
.subtitle {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.dense-table-card {
  border: 1px solid var(--border-color);
  background-color: var(--bg-card);
  border-radius: 8px;
}

.dense-table-card :deep(.el-card__body) {
  padding: 0 !important;
}

.custom-table {
  background: transparent;
}
.server-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}
.server-icon {
  color: var(--accent-blue);
  font-size: 18px;
}
.connection-info {
  display: flex;
  gap: 8px;
  align-items: center;
}
.mono-tag {
  font-family: var(--mono);
}

:deep(.custom-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.6);
}
:deep(.custom-dialog .el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
}
:deep(.custom-form .el-form-item__label) {
  color: var(--text-secondary);
  padding-bottom: 4px;
}
</style>
