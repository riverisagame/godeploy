<template>
  <el-dialog v-model="localVisible" title="账号部署权限配置" width="500px" @close="emit('close')">
    <el-form :model="form" label-width="120px" label-position="top">
      <el-form-item label="启用白名单限制">
        <el-switch v-model="form.restrict_git_authors" />
        <div style="font-size:12px;color:#888;margin-top:4px">开启后，你只能部署白名单作者提交的代码。</div>
      </el-form-item>
      <el-form-item label="Git 作者白名单" v-if="form.restrict_git_authors">
        <el-input v-model="form.bound_git_authors" placeholder="输入 Git Author 名称或邮箱，多个用逗号分隔" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button type="primary" @click="emit('save', form)" :loading="saving">{{ saving ? '保存中...' : '保存配置' }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  visible: boolean
  restrictGitAuthors: boolean
  boundGitAuthors: string
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', form: { restrict_git_authors: boolean; bound_git_authors: string }): void
}>()

const localVisible = ref(props.visible)
watch(() => props.visible, (v) => { localVisible.value = v })

const form = ref({
  restrict_git_authors: props.restrictGitAuthors,
  bound_git_authors: props.boundGitAuthors,
})

watch(() => [props.restrictGitAuthors, props.boundGitAuthors], () => {
  form.value = { restrict_git_authors: props.restrictGitAuthors, bound_git_authors: props.boundGitAuthors }
})
</script>
