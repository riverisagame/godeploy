<template>
  <div class="audit-log-container">
    <h2>审计日志</h2>
    <el-table :data="logs" style="width: 100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" label="操作人" width="120" />
      <el-table-column prop="method" label="请求方式" width="100" />
      <el-table-column prop="path" label="路径" />
      <el-table-column prop="created_at" label="操作时间" width="200">
        <template #default="scope">
          {{ new Date(scope.row.created_at).toLocaleString() }}
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper" style="margin-top: 20px; display: flex; justify-content: flex-end;">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, sizes, prev, pager, next"
        :page-sizes="[10, 20, 50, 100]"
        @size-change="fetchLogs"
        @current-change="fetchLogs"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

interface AuditLog {
  id: number;
  user_id: number;
  username: string;
  method: string;
  path: string;
  details: string;
  created_at: string;
}

const logs = ref<AuditLog[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await api.getAuditLogs(page.value, pageSize.value)
    logs.value = res.data.data
    total.value = res.data.total
  } catch (error) {
    console.error('Failed to fetch audit logs', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLogs()
})
</script>

<style scoped>
.audit-log-container {
  padding: 20px;
}
</style>
