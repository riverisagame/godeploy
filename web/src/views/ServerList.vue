<template>
  <div class="server-list">
    <div class="header-actions">
      <h2>服务器管理</h2>
      <el-button type="primary" @click="dialogVisible = true">新建服务器</el-button>
    </div>

    <el-table :data="servers" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="Name" label="服务器名称" />
      <el-table-column prop="IP" label="IP 地址" />
      <el-table-column prop="Port" label="SSH 端口" width="120" />
      <el-table-column label="操作" width="200">
        <template #default>
          <el-button size="small" type="danger" disabled>删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create Server Dialog -->
    <el-dialog title="新建服务器" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="服务器名称">
          <el-input v-model="form.name" placeholder="例如: web-prod-01"></el-input>
        </el-form-item>
        <el-form-item label="IP 地址">
          <el-input v-model="form.ip" placeholder="例如: 192.168.1.100"></el-input>
        </el-form-item>
        <el-form-item label="SSH 端口">
          <el-input-number v-model="form.port" :min="1" :max="65535"></el-input-number>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="createServer" :loading="creating">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const servers = ref<any[]>([])
const dialogVisible = ref(false)
const creating = ref(false)

const form = ref({
  name: '',
  ip: '',
  port: 22
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
  padding: 24px;
}
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
h2 {
  margin: 0;
  color: var(--text-primary);
  font-family: var(--mono);
  font-size: 24px;
}

:deep(.el-table) {
  background-color: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}
:deep(.el-table th.el-table__cell) {
  background-color: rgba(15, 23, 42, 0.6);
  color: var(--text-primary);
  font-weight: 600;
  border-bottom: 1px solid var(--border-color);
}
:deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-card);
}
:deep(.el-table--enable-row-hover .el-table__body tr:hover > td.el-table__cell) {
  background-color: rgba(59, 130, 246, 0.05);
}

:deep(.el-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
}
:deep(.el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
}
:deep(.el-input__wrapper) {
  background-color: var(--bg-dark);
  box-shadow: 0 0 0 1px var(--border-color) inset;
}
:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--accent-blue) inset !important;
}
</style>
