<template>
  <div class="deploy-history">
    <div class="history-header">
      <h3>部署历史</h3>
      <el-button size="small" @click="fetchHistory" text>
        <el-icon><Refresh /></el-icon>
      </el-button>
    </div>
    <el-table :data="deployments" style="width: 100%" class="custom-table" max-height="300" empty-text="暂无记录">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="created_at" label="时间" width="180">
        <template #default="scope">
          {{ new Date(scope.row.created_at).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column prop="commit_hash" label="Commit/Branch" show-overflow-tooltip />
      <el-table-column prop="release_name" label="Release 目录" width="160" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.status === 'success' ? 'success' : (scope.row.status === 'failed' ? 'danger' : 'info')" size="small">
            {{ scope.row.status.toUpperCase() }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="scope">
          <el-button link type="primary" size="small" @click="viewLog(scope.row.id)">日志</el-button>
          <el-button link type="danger" size="small" v-if="admin && scope.row.status === 'success'" @click="rollback(scope.row)">
            回滚此版本
          </el-button>
          <el-popconfirm title="确定要取消该部署吗？" @confirm="handleCancel(scope.row.id)" v-if="scope.row.status === 'pending' || scope.row.status === 'running'" placement="left">
            <template #reference>
              <el-button link type="danger" size="small">取消</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { api } from '../api'
import { computed } from 'vue'
import { isAdmin } from '../utils/auth'

const router = useRouter()
const admin = computed(() => isAdmin())

import type { Environment, Deployment } from '../types'

const props = defineProps<{
  env: Environment
  deployments: Deployment[]
  projectId: string | number | string[]
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const fetchHistory = () => {
  emit('refresh')
}

const viewLog = (id: number) => {
  router.push(`/deployments/${id}`)
}

const rollback = async (row: Deployment) => {
  try {
    await ElMessageBox.confirm(`确定要回滚到版本 [${row.release_name}] 吗？\n时间: ${new Date(row.created_at).toLocaleString()}`, '回滚确认', {
      confirmButtonText: '确定回滚',
      cancelButtonText: '取消',
      type: 'error',
    })
    
    const res = await api.rollbackDeployment(row.id, props.projectId as string, props.env.name, row.release_name)
    
    const deployId = res.data.id
    ElMessage.success('已触发回滚，正在前往控制台...')
    router.push(`/deployments/${deployId}`)
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data || '回滚失败')
    }
  }
}

const handleCancel = async (deployId: number) => {
  try {
    await api.cancelDeployment(deployId)
    ElMessage.success('取消请求已发送')
    fetchHistory()
  } catch (e: any) {
    ElMessage.error(e.response?.data || '取消失败')
  }
}
</script>

<style scoped>
.deploy-history {
  margin-top: 32px;
}
.history-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.history-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 16px;
}
:deep(.custom-table) {
  background-color: #0d1117;
  border-radius: 8px;
  border: 1px solid var(--border-color);
}
:deep(.custom-table th.el-table__cell) {
  background-color: rgba(255, 255, 255, 0.02);
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
}
:deep(.custom-table td.el-table__cell) {
  border-bottom: 1px solid var(--border-color);
}
:deep(.custom-table .el-table__row:hover > td.el-table__cell) {
  background-color: rgba(255, 255, 255, 0.05);
}
</style>
