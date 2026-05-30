<template>
  <div class="section-card history-section">
    <div class="card-header">部署与审计历史</div>
    <el-table :data="tasks" style="width:100%" size="default" :row-class-name="(row: any) => row.row.status === 'failed' ? 'row-failed' : ''">
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="release_name" label="Release 版本" width="155" />
      <el-table-column label="Commit" width="105">
        <template #default="scope"><code class="commit-hash">{{ scope.row.commit_id?.substring(0, 8) }}</code></template>
      </el-table-column>
      <el-table-column prop="username" label="操作人" width="100" />
      <el-table-column prop="description" label="发布备注" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="scope">
          <el-tag :type="getStatusTagType(scope.row.status)" effect="dark">{{ getStatusText(scope.row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="发布时间" width="180">
        <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作">
        <template #default="scope">
          <el-button-group>
            <el-button size="small" type="success" plain :disabled="scope.row.status !== 'success'" @click="$emit('rollback', scope.row)">回滚</el-button>
            <el-button size="small" type="primary" plain @click="$emit('show-diff', scope.row)">对比</el-button>
            <el-button size="small" type="info" plain @click="$emit('show-log', scope.row)">日志</el-button>
          </el-button-group>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="tasks.length === 0" description="暂无部署记录" :image-size="60" />
  </div>
</template>

<script setup lang="ts">
import { getStatusTagType, getStatusText, formatTime } from '../utils/deploy'

defineProps<{ tasks: any[] }>()
defineEmits<{
  (e: 'rollback', task: any): void
  (e: 'show-diff', task: any): void
  (e: 'show-log', task: any): void
}>()
</script>

<style scoped>
.card-header {
  font-size: 15px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 16px;
  border-left: 3px solid #00b4d8;
  padding-left: 8px;
}
.section-card {
  background-color: #161a23;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 20px;
  margin-top: 24px;
}
</style>
