<template>
  <div class="server-list">
    <div class="header-actions">
      <div>
        <h2>服务器管理</h2>
        <p class="subtitle">管理所有可用于部署的物理机和虚拟机资源</p>
      </div>
      <el-button type="primary" @click="dialogVisible = true" size="large">新建服务器</el-button>
    </div>

    <el-card class="dense-table-card" shadow="never">
      <el-table :data="servers" style="width: 100%" class="custom-table" :empty-text="'暂无服务器数据'">
        <el-table-column prop="ID" label="ID" width="80" align="center" />
        <el-table-column label="服务器名称">
          <template #default="scope">
            <div class="server-name-cell">
              <el-icon class="server-icon"><Monitor /></el-icon>
              <span>{{ scope.row.Name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="连接信息">
          <template #default="scope">
            <div class="connection-info">
              <el-tag size="small" type="primary" effect="dark" class="mono-tag">{{ scope.row.User }}@{{ scope.row.IP }}</el-tag>
              <el-tag size="small" type="info" class="mono-tag">Port: {{ scope.row.Port }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="认证方式" width="120">
          <template #default="scope">
            <el-tag size="small" :type="scope.row.KeyPath ? 'success' : 'warning'" effect="plain">
              {{ scope.row.KeyPath ? 'Key Auth' : 'Password' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default>
            <el-button size="small" type="danger" text bg>删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create Server Dialog -->
    <el-dialog title="新建服务器" v-model="dialogVisible" width="480px" destroy-on-close class="custom-dialog">
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
          <el-button type="primary" @click="createServer" :loading="creating">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Monitor } from '@element-plus/icons-vue'

const servers = ref<any[]>([])
const dialogVisible = ref(false)
const creating = ref(false)

const form = ref({
  name: '',
  ip: '',
  port: 22,
  user: 'root',
  key_path: '~/.ssh/id_rsa'
})

const fetchServers = async () => {
  try {
    const res = await axios.get('/api/servers')
    servers.value = res.data
  } catch (e) {
    ElMessage.error('获取服务器列表失败')
  }
}

const createServer = async () => {
  if (!form.value.name || !form.value.ip) {
    ElMessage.warning('请填写名称和IP')
    return
  }
  
  creating.value = true
  try {
    const res = await axios.post('/api/servers', form.value)
    ElMessage.success('创建成功')
    servers.value.push(res.data)
    dialogVisible.value = false
    form.value.name = ''
    form.value.ip = ''
    form.value.port = 22
    form.value.user = 'root'
    form.value.key_path = '~/.ssh/id_rsa'
  } catch (e: any) {
    ElMessage.error(e.response?.data || '创建失败')
  } finally {
    creating.value = false
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
