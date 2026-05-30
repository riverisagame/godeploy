<template>
  <el-dialog
    v-model="localVisible"
    :title="title"
    fullscreen
    destroy-on-close
    @close="handleClose"
  >
    <div style="margin-bottom:15px;display:flex;justify-content:space-between;align-items:center">
      <div class="diff-actions" style="display:flex;align-items:center;gap:12px">
        <slot name="actions" />
        <el-radio-group v-model="currentDiffType" size="default" :disabled="loadingDiff">
          <el-radio-button value="live">与线上对比 (Live Diff)</el-radio-button>
          <el-radio-button value="git_log">本地变更 (Git Log Diff)</el-radio-button>
        </el-radio-group>
      </div>
      <el-radio-group v-model="diffFormat" size="small" :disabled="loadingDiff">
        <el-radio-button value="line-by-line">竖向对比</el-radio-button>
        <el-radio-button value="side-by-side">横向对比</el-radio-button>
      </el-radio-group>
    </div>

    <div v-if="loadingDiff" class="diff-loading-skeleton">
      <div class="skeleton-bar wide" /><div class="skeleton-bar medium" /><div class="skeleton-bar narrow" />
      <div style="text-align:center;margin-top:32px;color:#484f58;font-size:14px">
        <el-icon class="is-loading" style="margin-right:6px"><Loading /></el-icon> 正在获取代码差异...
      </div>
    </div>

    <div v-else class="diff-split-layout">
      <div class="diff-left-sidebar" style="max-height:calc(100vh - 240px);overflow-y:auto">
        <div style="font-size:12px;color:#8a99ad;margin-bottom:10px;font-weight:600">变更文件列表</div>
        <el-tree
          :key="fileTreeData.length"
          ref="fileTreeRef"
          :data="fileTreeData"
          :show-checkbox="showCheckbox"
          node-key="path"
          default-expand-all
          :default-checked-keys="defaultCheckedKeys"
          :props="{ label: 'label', children: 'children' }"
          @node-click="handleFileClick"
          style="background:transparent;color:#e0e0e0"
        />
      </div>

      <div class="diff-right-content" v-loading="loadingFileDiff" element-loading-background="rgba(11, 14, 20, 0.8)">
        <div v-if="selectedFile" class="dark-diff" style="height:100%;overflow-y:auto" v-html="htmlDiff" />
        <div v-else class="diff-empty-placeholder">
          <el-icon size="48" color="#30363d"><Document /></el-icon>
          <p>请在左侧文件列表中选择要查看差异的文件</p>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { html } from 'diff2html'
import 'diff2html/bundles/css/diff2html.min.css'
import axios from 'axios'

const props = defineProps<{
  visible: boolean
  projectId: string
  envId: string
  branch: string
  targetType: string
  showCheckbox: boolean
  taskId?: number | null
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const localVisible = ref(props.visible)
watch(() => props.visible, (v) => { localVisible.value = v })

const currentDiffType = ref('git_log')
const diffFormat = ref('side-by-side')
const loadingDiff = ref(false)
const loadingFileDiff = ref(false)
const selectedFile = ref('')
const diffText = ref('')
const fileTreeData = ref<any[]>([])
const defaultCheckedKeys = ref<string[]>([])
const fileTreeRef = ref<any>(null)

const title = computed(() => props.taskId ? '代码对比' : '变更预览')

function buildTree(paths: string[]) {
  const root: any[] = []
  paths.forEach(p => {
    const parts = p.split('/')
    let current = root
    let curPath = ''
    parts.forEach((part, index) => {
      curPath = curPath ? `${curPath}/${part}` : part
      let node = current.find((item: any) => item.label === part)
      if (!node) { node = { label: part, path: curPath, children: [] }; current.push(node) }
      if (index === parts.length - 1) delete node.children
      else current = node.children
    })
  })
  return root
}

const htmlDiff = computed(() => {
  if (!diffText.value) return ''
  let text = diffText.value
  const LIMIT = 100 * 1024
  if (text.length > LIMIT) text = text.substring(0, LIMIT) + '\n\n... [截断: 超过100KB]'
  try {
    return html(text, { drawFileList: true, matching: 'none', outputFormat: diffFormat.value })
  } catch { return `<pre>${text}</pre>` }
})

async function loadFileDiff(path: string) {
  if (!props.projectId) return
  loadingFileDiff.value = true
  try {
    if (props.taskId) {
      const res = await axios.get(`/api/tasks/${props.taskId}/diff`, { params: { file: path, diff_type: currentDiffType.value } })
      diffText.value = res.data.diff || ''
    } else {
      const res = await axios.get(`/api/projects/${props.projectId}/preview_diff`, {
        params: { to: props.branch, env_id: props.envId, file: path, diff_type: currentDiffType.value }
      })
      diffText.value = res.data.diff || ''
    }
  } catch { diffText.value = '加载失败' }
  finally { loadingFileDiff.value = false }
}

function handleFileClick(node: any) {
  if (!node.children) {
    selectedFile.value = node.path
    loadFileDiff(node.path)
  }
}

watch(currentDiffType, () => {
  if (selectedFile.value) loadFileDiff(selectedFile.value)
})

watch(localVisible, async (visible) => {
  if (!visible || !props.projectId) return
  loadingDiff.value = true
  try {
    let files: string[] = []
    if (props.taskId) {
      const res = await axios.get(`/api/tasks/${props.taskId}/diff`)
      files = (res.data.files || '').split('\n').filter(Boolean).map((l: string) => l.split(/\s+/).slice(1).join(' '))
    } else {
      const res = await axios.get(`/api/projects/${props.projectId}/preview_diff`, {
        params: { to: props.branch, env_id: props.envId }
      })
      files = res.data.files || []
    }
    fileTreeData.value = buildTree(files)
    defaultCheckedKeys.value = [...files]
    if (files.length > 0) { selectedFile.value = files[0]; await loadFileDiff(files[0]) }
  } catch { fileTreeData.value = [{ label: '加载失败', path: '' }] }
  finally { loadingDiff.value = false }
})

function handleClose() {
  emit('close')
  selectedFile.value = ''
  diffText.value = ''
  fileTreeData.value = []
}
</script>

<style scoped>
.diff-split-layout { display:flex; gap:16px; height:calc(100vh - 200px); background:#0a0e13 }
.diff-left-sidebar { width:240px; flex-shrink:0; background:#0d1117; border:1px solid #21262d; border-radius:8px; padding:12px }
.diff-right-content { flex:1; background:#0d1117; border:1px solid #21262d; border-radius:8px; overflow:hidden }
.diff-empty-placeholder { display:flex; flex-direction:column; align-items:center; justify-content:center; height:100%; color:#484f58; gap:12px }
.diff-loading-skeleton { padding:20px }
.skeleton-bar { height:16px; background:rgba(48,54,61,0.5); border-radius:4px; margin-bottom:12px }
.skeleton-bar.wide { width:90% }
.skeleton-bar.medium { width:65% }
.skeleton-bar.narrow { width:40% }

.dark-diff :deep(.d2h-wrapper) { color:#c9d1d9; background:#0d1117 }
.dark-diff :deep(.d2h-file-header) { background:#161b22; border-bottom:1px solid #30363d; color:#8b949e }
.dark-diff :deep(.d2h-file-stats) { background:#161b22; color:#8b949e }
.dark-diff :deep(.d2h-lines-added) { color:#3fb950 }
.dark-diff :deep(.d2h-lines-deleted) { color:#ff7b72 }
.dark-diff :deep(.d2h-tag) { background:#161b22; border-color:#30363d; color:#8b949e }
.dark-diff :deep(.d2h-diff-table) { background:#0d1117; border:1px solid #30363d; border-radius:6px }
.dark-diff :deep(.d2h-diff-table td) { background:transparent }
.dark-diff :deep(.d2h-diff-table tr) { background:#0d1117 }
.dark-diff :deep(.d2h-diff-tbody tr:nth-child(even) td) { background:#0d1117 }
.dark-diff :deep(.d2h-diff-tbody tr:nth-child(odd) td) { background:#0d1117 }
.dark-diff :deep(.d2h-code-line) { color:#c9d1d9; background:#0d1117 }
.dark-diff :deep(.d2h-code-line-ctn) { color:#c9d1d9 }
.dark-diff :deep(.d2h-code-line-prefix) { color:#484f58 }
.dark-diff :deep(.d2h-code-side-linenumber) { background:#161b22 !important; color:#484f58 !important; border-right:1px solid #30363d !important }
.dark-diff :deep(.d2h-code-side-line) { color:#c9d1d9; background:#0d1117 }
.dark-diff :deep(.d2h-emptyplaceholder) { background:#161b22; border-color:#30363d }
.dark-diff :deep(.d2h-ins .d2h-code-line) { background:#0c2d1b !important; color:#3fb950 !important }
.dark-diff :deep(.d2h-ins .d2h-code-side-line) { background:#0c2d1b !important }
.dark-diff :deep(.d2h-ins .d2h-code-side-linenumber) { background:#133a22 !important; color:#3fb950 !important; border-right:1px solid #1a7f37 !important }
.dark-diff :deep(.d2h-ins .d2h-code-line-prefix) { color:#3fb950 }
.dark-diff :deep(.d2h-del .d2h-code-line) { background:#2d1010 !important; color:#ff7b72 !important }
.dark-diff :deep(.d2h-del .d2h-code-side-line) { background:#2d1010 !important }
.dark-diff :deep(.d2h-del .d2h-code-side-linenumber) { background:#3d1515 !important; color:#ff7b72 !important; border-right:1px solid #da3633 !important }
.dark-diff :deep(.d2h-del .d2h-code-line-prefix) { color:#ff7b72 }
.dark-diff :deep(.d2h-cntx .d2h-code-line) { color:#c9d1d9 }
.dark-diff :deep(.d2h-code-linenumber) { background:#161b22 !important; color:#484f58 !important; border-right:1px solid #30363d !important }
.dark-diff :deep(.d2h-code-linenumber-content) { color:#484f58 }
.dark-diff :deep(.d2h-file-wrapper) { border-color:#30363d; background:#0d1117; border-radius:8px; margin-bottom:12px }
.dark-diff :deep(.d2h-file-list) { background:#0d1117 }
.dark-diff :deep(.d2h-file-list-header) { background:#161b22; border-color:#30363d }
.dark-diff :deep(.d2h-file-list-line) { color:#c9d1d9; border-color:#30363d }
.dark-diff :deep(.d2h-file-list-line.active) { background:#1f2937 }
.dark-diff :deep(.d2h-icon) { fill:#8b949e }
.dark-diff :deep(code) { color:#c9d1d9; background:transparent }
.dark-diff :deep(td.d2h-info) { background:#0d1117 !important; color:#8b949e }
.dark-diff :deep(.d2h-code-line del), .dark-diff :deep(.d2h-code-line ins) { background:transparent }
</style>
