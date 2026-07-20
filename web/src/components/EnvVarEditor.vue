<template>
  <div class="env-vars-section">
    <div class="section-header">
      <h3><el-icon><Setting /></el-icon> 环境变量 (Environment Variables)</h3>
      <el-button size="small" type="primary" plain @click="addEnvVar">+ 添加变量</el-button>
    </div>
    <el-table :data="modelValue" style="width: 100%" class="custom-table" max-height="300" empty-text="暂无环境变量">
      <el-table-column label="Key" width="300">
        <template #default="scope">
          <el-input v-model="scope.row.key" placeholder="例如: DB_HOST"></el-input>
        </template>
      </el-table-column>
      <el-table-column label="Value">
        <template #default="scope">
          <el-input v-model="scope.row.value" placeholder="例如: 127.0.0.1" show-password></el-input>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80" align="center">
        <template #default="scope">
          <el-button link type="danger" size="small" @click="removeEnvVar(scope.$index)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { Setting, Delete } from '@element-plus/icons-vue'
import type { EnvVar } from '../types'

const props = defineProps<{
  modelValue: EnvVar[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: EnvVar[]): void
}>()

const addEnvVar = () => {
  const newValue = [...(props.modelValue || [])]
  newValue.push({ key: '', value: '' })
  emit('update:modelValue', newValue)
}

const removeEnvVar = (index: number) => {
  const newValue = [...props.modelValue]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}
</script>

<style scoped>
.env-vars-section {
  margin-top: 24px;
}
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.section-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
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
