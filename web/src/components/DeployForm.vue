<template>
  <div class="section-card deploy-action-card">
    <div class="card-header">触发部署</div>
    <el-form label-position="top">
      <el-form-item label="发布目标类型">
        <el-radio-group :model-value="targetType" @update:model-value="$emit('update:targetType', $event)" size="small">
          <el-radio-button value="branch">分支</el-radio-button>
          <el-radio-button value="tag">Tag</el-radio-button>
          <el-radio-button value="commit">历史 Commit</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="targetType === 'branch' || targetType === 'tag'" label="选择分支/Tag">
        <el-select :model-value="branch" @update:model-value="$emit('update:branch', $event)" filterable allow-create placeholder="选择..." :loading="loadingRefs" style="width:100%">
          <el-option v-for="item in refsList.filter(r => r.type === targetType)" :key="item.name" :label="item.name" :value="item.name">
            <span style="float:left">{{ item.name }}</span>
            <span style="float:right;color:var(--el-text-color-secondary);font-size:13px">{{ item.hash?.substring(0,7) }}</span>
          </el-option>
        </el-select>
      </el-form-item>
      <div v-if="targetType === 'commit'" class="commit-filters" style="margin-bottom:18px">
        <el-row :gutter="10">
          <el-col :span="6"><el-select v-model="commitFilters.ref" placeholder="分支/Tag" size="small" clearable filterable style="width:100%"><el-option v-for="item in refsList" :key="item.name" :label="item.name" :value="item.name" /></el-select></el-col>
          <el-col :span="6"><el-input v-model="commitFilters.keyword" placeholder="搜 Message" size="small" clearable /></el-col>
          <el-col :span="6"><el-input v-model="commitFilters.author" placeholder="搜提交人" size="small" clearable /></el-col>
          <el-col :span="6"><el-input v-model="commitFilters.file" placeholder="搜文件" size="small" clearable /></el-col>
        </el-row>
        <el-form-item label="选择 Commit" style="margin-top:10px">
          <el-select :model-value="branch" @update:model-value="$emit('update:branch', $event)" filterable remote :remote-method="onSearchCommits" :loading="loadingCommits" placeholder="选择 Commit..." style="width:100%">
            <el-option v-for="item in commitsList" :key="item.hash" :label="item.message" :value="item.hash">
              <div style="display:flex;justify-content:space-between;align-items:center">
                <span style="max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ item.message }}</span>
                <span style="font-size:12px;color:#888">{{ item.author }} - {{ item.hash?.substring(0,7) }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </div>
      <el-form-item label="发布备注/说明" required style="margin-top:15px">
        <el-input :model-value="description" @update:model-value="$emit('update:description', $event)" placeholder="请输入本次上线的备注说明" type="textarea" :rows="2" />
      </el-form-item>
      <div style="display:flex;gap:10px;margin-top:20px">
        <el-button type="primary" size="large" class="trigger-deploy-btn" @click="$emit('deploy')" style="flex:1">
          <el-icon><Upload /></el-icon> 触发上线
        </el-button>
        <el-button size="large" @click="$emit('preview-diff')" :loading="loadingPreviewDiff" style="flex:1;margin-left:0">
          <el-icon><View /></el-icon> 预览 Diff
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'

const props = defineProps<{
  refsList: any[]
  commitsList: any[]
  loadingRefs: boolean
  loadingCommits: boolean
  loadingPreviewDiff: boolean
  targetType: string
  branch: string
  description: string
}>()

const emit = defineEmits<{
  (e: 'deploy'): void
  (e: 'preview-diff'): void
  (e: 'search-commits'): void
  (e: 'update:targetType', val: string): void
  (e: 'update:branch', val: string): void
  (e: 'update:description', val: string): void
}>()

const commitFilters = reactive({
  keyword: '', author: '', file: '', ref: ''
})

function onSearchCommits(query: string) {
  commitFilters.keyword = query
  emit('search-commits')
}
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
}
.trigger-deploy-btn {
  border-radius: 8px;
  background: linear-gradient(135deg, #0077b6 0%, #0096c7 100%);
  border: none;
  font-weight: 600;
}
</style>
