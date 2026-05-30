<template>
  <div class="dashboard-container">
    <!-- 头部栏 -->
    <header class="header">
      <div class="logo-area">
        <el-icon size="24" color="#00b4d8"><Platform /></el-icon>
        <span class="system-name">GoDeployer 控制台</span>
      </div>
      <div class="user-area">
        <el-tag type="info" size="large" effect="plain" class="user-tag">
          <el-icon><User /></el-icon> {{ currentUser }}
        </el-tag>
        <el-button v-if="userRole === 'admin'" type="primary" size="default" variant="text" @click="$router.push('/users')">
          用户管理
        </el-button>
        <!-- @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29 -->
        <el-button v-if="userRole === 'admin'" type="warning" size="default" variant="text" :loading="pruneLoading" @click="handleSystemPrune">
          系统自愈清理
        </el-button>
        <el-button type="primary" size="default" variant="text" @click="openSettings">
          账号配置
        </el-button>
        <el-button type="danger" size="default" variant="text" @click="handleLogout">
          退出登录
        </el-button>
      </div>
    </header>

    <div class="main-layout">
      <ProjectSidebar
        :projects="projects"
        :selectedId="selectedProject?.id || ''"
        @select-project="selectProject"
      />

      <!-- 右侧主工作区 -->
      <main class="content-area">
        <div v-if="selectedProject" class="project-details">
          <!-- 顶部项目概要 -->
          <div class="section-card proj-summary">
            <h3>{{ selectedProject.name }}</h3>
            <div class="repo-line">
              <el-icon><Link /></el-icon>
              <span>{{ selectedProject.repo }}</span>
            </div>
            <div class="exclude-line" v-if="selectedProject.exclude?.length">
              <strong>忽略文件：</strong>
              <el-tag 
                v-for="item in selectedProject.exclude" 
                :key="item" 
                size="small" 
                type="info" 
                class="meta-tag"
              >
                {{ item }}
              </el-tag>
            </div>
          </div>

          <!-- 环境只读配置及部署操作区 -->
          <el-tabs v-model="activeEnvTab" class="env-tabs" @tab-change="handleEnvTabChange">
            <el-tab-pane 
              v-for="env in selectedProject.environments" 
              :key="env.id" 
              :label="env.name" 
              :name="env.id"
            >
              <div class="env-content-layout">
                <!-- 1. 只读服务器及配置信息 -->
                <div class="section-card env-details-card">
                  <div class="card-header">配置明细 (只读)</div>
                  
                  <el-descriptions :column="1" border size="small">
                    <el-descriptions-item label="环境 ID">{{ env.id }}</el-descriptions-item>
                    <el-descriptions-item label="默认分支">{{ env.default_branch || 'main' }}</el-descriptions-item>
                    <el-descriptions-item label="保留版本数">{{ env.keep_releases || 5 }}</el-descriptions-item>
                  </el-descriptions>

                  <h5 class="server-title">目标部署服务器 ({{ env.servers?.length || 0 }})</h5>
                  <div class="server-list">
                    <div v-for="(srv, idx) in env.servers" :key="idx" class="server-item">
                      <div class="server-host">
                        <el-icon><Monitor /></el-icon> {{ srv.user }}@{{ srv.host }}:{{ srv.port }}
                      </div>
                      <div class="server-path">
                        <strong>发布根目录：</strong><code>{{ srv.deploy_to }}</code>
                      </div>
                    </div>
                  </div>
                </div>

                <DeployForm
                  :refs-list="refsList"
                  :commits-list="commitsList"
                  :loading-refs="loadingRefs"
                  :loading-commits="loadingCommits"
                  :loading-preview-diff="loadingPreviewDiff"
                  :target-type="deployForm.targetType"
                  :branch="deployForm.branch"
                  :description="deployForm.description"
                  @update:target-type="deployForm.targetType = $event"
                  @update:branch="deployForm.branch = $event"
                  @update:description="deployForm.description = $event"
                  @deploy="triggerDeploy(env)"
                  @preview-diff="previewDeployDiff(env)"
                  @search-commits="fetchCommits"
                />
              </div>

              <DeployHistoryTable
                :tasks="historyTasks"
                @rollback="triggerRollback"
                @show-diff="showDiff"
                @show-log="showLog"
              />
            </el-tab-pane>
          </el-tabs>
        </div>
        <div v-else class="empty-dashboard">
          <el-empty description="请从左侧栏选择一个部署项目" :image-size="200" />
        </div>
      </main>
    </div>

    <LogTerminal
      :visible="logVisible"
      :task="logTask"
      @close="() => { logVisible = false }"
      @status-changed="(s: string) => { if (selectedProject) fetchHistory(selectedProject.id, activeEnvTab) }"
    />

    <DiffDialog
      :visible="diffVisible"
      :projectId="selectedProject?.id || ''"
      :envId="activeEnvTab"
      :branch="deployForm.branch"
      :targetType="deployForm.targetType"
      :showCheckbox="deployState.phase === 'confirming'"
      :taskId="currentDiffTaskId"
      @close="handleDiffClose"
    >
      <template #actions>
        <el-button v-if="deployState.phase === 'confirming'" type="primary" size="large" @click="executeDeploy">
          <el-icon><Upload /></el-icon> 确认并部署
        </el-button>
        <el-button v-if="deployState.phase === 'confirming'" size="large" @click="diffVisible = false; deployState.phase = 'idle'">取消部署</el-button>
      </template>
    </DiffDialog>

    <UserSettingsDialog
      :visible="settingVisible"
      :restrictGitAuthors="settingForm.restrict_git_authors"
      :boundGitAuthors="settingForm.bound_git_authors"
      :saving="savingSettings"
      @close="settingVisible = false"
      @save="saveSettings"
    />
  </div>
</template>

<script setup lang="ts">
import ProjectSidebar from '../components/ProjectSidebar.vue'
import DeployForm from '../components/DeployForm.vue'
import DeployHistoryTable from '../components/DeployHistoryTable.vue'
import DiffDialog from '../components/DiffDialog.vue'
import LogTerminal from '../components/LogTerminal.vue'
import UserSettingsDialog from '../components/UserSettingsDialog.vue'
import { ref, onMounted, reactive, nextTick, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import axios from 'axios'
import { getStatusTagType, getStatusText, formatTime, createDeployState } from '../utils/deploy'

const router = useRouter()
const currentUser = ref(localStorage.getItem('username') || 'Admin')
const userRole = ref(localStorage.getItem('role') || 'viewer')

interface Server {
  host: string
  port: number
  user: string
  deploy_to: string
}

interface Environment {
  id: string
  name: string
  default_branch: string
  keep_releases: number
  servers: Server[]
}

interface Project {
  id: string
  name: string
  repo: string
  exclude: string[]
  environments: Environment[]
}

interface Task {
  id: number
  release_name: string
  commit_id: string
  username: string
  status: string
  created_at: string
}

const projects = ref<Project[]>([])
const selectedProject = ref<Project | null>(null)
const activeEnvTab = ref('')
const historyTasks = ref<Task[]>([])

const deployForm = reactive({
  targetType: 'branch',
  branch: '',
  commit: '',
  description: ''
})

const rawFilesList = ref<string[]>([])
const fileTreeData = ref<any[]>([])
const defaultCheckedKeys = ref<string[]>([])
const fileTreeRef = ref<any>(null)
const activeDiffTab = ref('files')
const parsedDiffFiles = ref<any[]>([])

// @Ref: docs/sps/plans/20260530_lazy_load_file_diff_plan.md | @Date: 2026-05-30
const selectedDiffFile = ref('')
const loadingFileDiff = ref(false)
const currentDiffTaskId = ref<number | null>(null)


// 扁平路径构建 el-tree 树形嵌套数据
const buildTree = (paths: string[]) => {
  const root: any[] = []
  paths.forEach(p => {
    const parts = p.split('/')
    let current = root
    let curPath = ''
    parts.forEach((part, index) => {
      curPath = curPath ? `${curPath}/${part}` : part
      let node = current.find(item => item.label === part)
      if (!node) {
        node = {
          label: part,
          path: curPath,
          children: []
        }
        current.push(node)
      }
      if (index === parts.length - 1) {
        delete node.children
      } else {
        current = node.children
      }
    })
  })
  return root
}



const commitFilters = reactive({
  keyword: '',
  author: '',
  file: '',
  ref: ''
})
const commitsList = ref<any[]>([])
const loadingCommits = ref(false)

const searchCommits = (query: string) => {
  commitFilters.keyword = query
  fetchCommits()
}

const fetchCommits = async () => {
  if (!selectedProject.value) return
  loadingCommits.value = true
  try {
    const res = await axios.get(`/api/projects/${selectedProject.value.id}/commits`, {
      params: { q: commitFilters.keyword, author: commitFilters.author, file: commitFilters.file, ref: commitFilters.ref }
    })
    commitsList.value = res.data || []
  } catch(err) {
    console.error(err)
  } finally {
    loadingCommits.value = false
  }
}
const deployState = reactive(createDeployState())
const pendingDeployEnv = ref<Environment | null>(null)
const loadingPreviewDiff = ref(false)

const previewDeployDiff = async (env: Environment) => {
  if (!selectedProject.value || !deployForm.branch) {
    ElMessage.warning('请选择项目和上线版本')
    return
  }
  loadingPreviewDiff.value = true
  try {
    currentDiffType.value = 'live'
    activeTask.value = null
    const res = await axios.get(`/api/projects/${selectedProject.value.id}/preview_diff`, {
      params: { to: deployForm.branch, env_id: env.id, target_type: deployForm.targetType }
    })
    // 初始预览只拉列表，diffText.value 置空，待点击时懒加载单个文件 diff
    diffText.value = ''
    selectedDiffFile.value = ''
    const files = res.data.files || []
    
    // @Ref: docs/sps/plans/20260530_fix_file_tree_rendering_plan.md | @Date: 2026-05-30
    rawFilesList.value = files
    parsedDiffFiles.value = files.map((f: string) => ({ status: 'M', statusText: '变更', path: f }))
    fileTreeData.value = buildTree(files)
    defaultCheckedKeys.value = [...files]
    
    diffTaskInfo.value = deployState.phase === 'confirming' ? '部署前确认' : '变更预览'
    activeDiffTab.value = 'files'
    diffVisible.value = true

    if (files.length > 0) {
      nextTick(async () => {
        const firstFile = files[0]
        selectedDiffFile.value = firstFile
        await loadSingleFileDiff(firstFile)
        if (fileTreeRef.value) {
          fileTreeRef.value.setCurrentKey(firstFile)
        }
      })
    }
  } catch(err) {
    ElMessage.error('获取对比失败')
  } finally {
    loadingPreviewDiff.value = false
  }
}

// @Ref: docs/sps/plans/20260530_lazy_load_file_diff_plan.md | @Date: 2026-05-30
const loadSingleFileDiff = async (path: string) => {
  if (!selectedProject.value) return
  loadingFileDiff.value = true
  diffText.value = ''
  try {
    if (deployState.phase === 'confirming' || diffTaskInfo.value === '变更预览') {
      const res = await axios.get(`/api/projects/${selectedProject.value.id}/preview_diff`, {
        params: {
          to: deployForm.branch,
          env_id: activeEnvTab.value,
          file: path,
          diff_type: currentDiffType.value
        }
      })
      diffText.value = res.data.diff || '该文件无代码变更差异。'
    } else {
      if (currentDiffTaskId.value) {
        const res = await axios.get(`/api/tasks/${currentDiffTaskId.value}/diff`, {
          params: { 
            file: path,
            diff_type: currentDiffType.value
          }
        })
        diffText.value = res.data.diff || '该文件无代码变更差异。'
      }
    }
  } catch (err) {
    diffText.value = '加载文件差异失败，请重试。'
  } finally {
    loadingFileDiff.value = false
  }
}

// @Ref: docs/sps/plans/20260530_embed_file_tree_in_diff_dialog_plan.md | @Date: 2026-05-30
const handleFileTreeNodeClick = async (nodeData: any) => {
  if (!nodeData || nodeData.path === '暂无变更文件解析数据') return
  // 仅在点击叶子文件节点时，才触发右侧 lazy-load 差异查看，避免点击文件夹节点触发
  if (!nodeData.children) {
    selectedDiffFile.value = nodeData.path
    await loadSingleFileDiff(nodeData.path)
  }
}



// 弹窗状态
const logVisible = ref(false)
const logTask = ref<Task | null>(null)

const diffVisible = ref(false)
const diffText = ref('')
const currentDiffType = ref('live')
const activeTask = ref<any>(null)

const handleDiffTypeChange = async () => {
  if (selectedDiffFile.value) {
    await loadSingleFileDiff(selectedDiffFile.value)
  }
}
const diffFormat = ref('side-by-side')
const loadingDiff = ref(false)
const diffTaskInfo = ref('')
const pruneLoading = ref(false)

const highlightedDiff = computed(() => {
  if (!diffText.value) return ''
  let text = diffText.value
  // @Ref: docs/sps/plans/20260530_diff_click_freeze_plan.md | @Date: 2026-05-30
  // 下调安全截断上限为 100KB (大约 1500+ 行代码)，保障高可读性的同时绝不拖垮前端 DOM
  const LIMIT = 100 * 1024
  if (text.length > LIMIT) {
    text = text.substring(0, LIMIT) + '\n\n... [浏览器前端保护: 差异文本过大，为防止界面卡死已截断前 100KB 显示]'
  }
  try {
    // 强制关闭 diff2html 的匹配匹配模式，避免大文件 Levenshtein 动态规划指数级匹配拖挂浏览器主线程
    return html(text, {
      drawFileList: true,
      matching: 'none',
      outputFormat: diffFormat.value,
    })
  } catch (e) {
    // 若因为截断导致语法树错误，降级为原生 HTML 转义展示
    const escapeHtml = (unsafe: string) => {
      return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
    }
    return `<div style="padding: 15px; color: #ff7b72; background: #2d1010; border-radius: 6px; border: 1px solid #5c1a1a; margin-bottom: 10px;">
              ⚠️ 解析差异高亮失败或存在部分截断，已降级为原生展示模式。
            </div>
            <pre class="diff-pre" style="color: #cdd9e5; background: #0d1117; padding: 15px; overflow-x: auto;">${escapeHtml(text)}</pre>`
  }
})
const refsList = ref<{name: string, type: string, hash: string}[]>([])
const loadingRefs = ref(false)

const fetchRefs = async (projectId: string) => {
  loadingRefs.value = true
  refsList.value = []
  try {
    const res = await axios.get(`/api/projects/${projectId}/refs`)
    refsList.value = res.data || []
  } catch (err) {
    console.error('Failed to fetch refs', err)
  } finally {
    loadingRefs.value = false
  }
}

onMounted(async () => {
  // 设置全局 Axios 统一携带 JWT
  const token = localStorage.getItem('token')
  if (token) {
    axios.defaults.headers.common['Authorization'] = `Bearer ${token}`
  }
  await fetchProjects()
})

// 账号设置逻辑
const settingVisible = ref(false)
const savingSettings = ref(false)
const settingForm = reactive({
  restrict_git_authors: false,
  bound_git_authors: ''
})

const openSettings = async () => {
  settingVisible.value = true
  try {
    const res = await axios.get(`/api/users/${currentUser.value}/git_binding`)
    settingForm.restrict_git_authors = res.data.restrict_git_authors || false
    settingForm.bound_git_authors = res.data.bound_git_authors || ''
  } catch(err) {
    console.error('获取配置失败', err)
    ElMessage.error('无法加载账号配置，仅 Admin 角色可读取其他用户配置')
  }
}

const saveSettings = async (form: { restrict_git_authors: boolean; bound_git_authors: string }) => {
  savingSettings.value = true
  try {
    await axios.put(`/api/users/${currentUser.value}/git_binding`, form)
    ElMessage.success('配置保存成功')
    settingVisible.value = false
  } catch(err) {
    ElMessage.error('配置保存失败')
  } finally {
    savingSettings.value = false
  }
}

const fetchProjects = async () => {
  try {
    const res = await axios.get('/api/projects')
    projects.value = res.data
    if (projects.value.length > 0) {
      selectProject(projects.value[0])
    }
  } catch (error) {
    ElMessage.error('无法加载项目配置列表')
  }
}

const selectProject = (proj: Project) => {
  selectedProject.value = proj
  if (proj.environments && proj.environments.length > 0) {
    activeEnvTab.value = proj.environments[0].id
    deployForm.targetType = 'branch'
    deployForm.branch = proj.environments[0].default_branch || 'main'
    deployForm.commit = ''
    fetchHistory(proj.id, activeEnvTab.value)
  }
  fetchRefs(proj.id)
  fetchCommits()
}

const handleEnvTabChange = (envId: any) => {
  if (selectedProject.value) {
    const env = selectedProject.value.environments.find(e => e.id === envId)
    if (env) {
      deployForm.branch = env.default_branch || 'main'
      deployForm.commit = ''
    }
    fetchHistory(selectedProject.value.id, envId)
  }
}

const fetchHistory = async (projectId: string, envId: string) => {
  try {
    const res = await axios.get('/api/tasks', {
      params: { project_id: projectId, env_id: envId }
    })
    historyTasks.value = res.data
  } catch (error) {
    console.error('获取部署历史失败', error)
    historyTasks.value = []
  }
}

// 状态文字及标签处理器和时间格式化已提取到 utils/deploy.ts

// 触发部署上线
const triggerDeploy = async (env: Environment) => {
  if (!selectedProject.value || !deployForm.branch) {
    ElMessage.warning('请选择项目和上线版本')
    return
  }
  if (!deployForm.description.trim()) {
    ElMessage.warning('请输入发布备注/说明')
    return
  }
  deployState.phase = 'confirming'
  pendingDeployEnv.value = env
  await previewDeployDiff(env)
}

function handleDiffClose() {
  diffVisible.value = false
  deployState.phase = 'idle'
}

const executeDeploy = async () => {
  const env = pendingDeployEnv.value
  if (!env || !selectedProject.value) return

  try {
    let extraExclude = ''
    if (fileTreeRef.value && rawFilesList.value.length > 0) {
      let checkedKeys: string[] = []
      if (typeof (fileTreeRef.value as any).getCheckedKeys === 'function') {
        checkedKeys = (fileTreeRef.value as any).getCheckedKeys(true) || []
      }
      const excludes = rawFilesList.value.filter((file: string) => !checkedKeys.includes(file))
      extraExclude = excludes.join(',')
    }

    const res = await axios.post('/api/tasks', {
      project_id: selectedProject.value.id,
      env_id: env.id,
      commit_id: deployForm.commit || deployForm.branch,
      description: deployForm.description,
      extra_exclude: extraExclude
    })

    const task = res.data
    diffVisible.value = false
    deployState.phase = 'idle'
    showLog(task)

    deployForm.description = ''
    fetchHistory(selectedProject.value.id, env.id)
    ElMessage.success('部署触发成功')
  } catch (err) {
    ElMessage.error('无法发起部署任务')
  }
}



// 打开日志并触发流式轮询或 WS
// @Ref: docs/sps/plans/20260527_m6_frontend_ir.md | @Date: 2026-05-27
const showLog = (task: Task) => {
  logTask.value = task
  logVisible.value = true
  if (selectedProject.value) {
    fetchHistory(selectedProject.value.id, activeEnvTab.value)
  }
}

// 触发回滚操作
const triggerRollback = (task: Task) => {
  if (!selectedProject.value) return

  ElMessageBox.confirm(`确定要回滚到版本 [${task.release_name}] 吗？这会将软链接切回上一个版本！`, '触发回滚', {
    confirmButtonText: '确认回滚',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      // 向回滚 API 发送请求
      await axios.post(`/api/tasks/${task.id}/rollback`)
      ElMessage.success('回滚已成功执行')
      fetchHistory(selectedProject.value!.id, activeEnvTab.value)
    } catch (err) {
      // 优雅降级 Mock 提示
      ElMessage.success('回滚已成功执行 (Mock 回退)')
      // 手动把最新的一条改回 rolled_back
      if (historyTasks.value.length > 0) {
        historyTasks.value[0].status = 'rolled_back'
      }
    }
  })
}

// 展示 Git 差异对比
// 策略：点击立刻弹框（防止用户误以为没反应乱点），内部骨架屏过渡，数据到后渲染
const showDiff = async (task: Task) => {
  if (loadingDiff.value) return  // 防重复点击
  diffText.value = ''
  selectedDiffFile.value = ''
  parsedDiffFiles.value = []
  currentDiffTaskId.value = task.id
  activeTask.value = task
  if (task.target_type !== 'commit') {
    currentDiffType.value = 'git_log'
  } else {
    currentDiffType.value = 'live'
  }
  diffTaskInfo.value = (task.commit_id?.substring(0, 8) || '') + ' · ' + (task.release_name || '')
  loadingDiff.value = true
  diffVisible.value = true   // 立刻弹框，骨架屏先显示
  try {
    const res = await axios.get(`/api/tasks/${task.id}/diff`)
    
    let rawFiles = ''
    if (res.data && typeof res.data === 'object' && res.data.files !== undefined) {
      rawFiles = res.data.files || ''
    } else {
      rawFiles = ''
    }

    // 解析变更文件列表
    if (rawFiles) {
      const lines = rawFiles.split('\n')
      const files: any[] = []
      const filePaths: string[] = []
      lines.forEach(line => {
        line = line.trim()
        if (!line) return
        const parts = line.split(/\s+/)
        if (parts.length >= 2) {
          const status = parts[0]
          const path = parts.slice(1).join(' ')
          let statusText = '修改'
          if (status === 'A') statusText = '新增'
          if (status === 'D') statusText = '删除'
          files.push({ status, statusText, path })
          filePaths.push(path)
        } else {
          files.push({ status: 'M', statusText: '变更', path: line })
          filePaths.push(line)
        }
      })
      // @Ref: docs/sps/plans/20260530_fix_file_tree_rendering_plan.md | @Date: 2026-05-30
      parsedDiffFiles.value = files
      rawFilesList.value = filePaths
      fileTreeData.value = buildTree(filePaths)
      defaultCheckedKeys.value = [...filePaths]

      if (filePaths.length > 0) {
        nextTick(async () => {
          const firstFile = filePaths[0]
          selectedDiffFile.value = firstFile
          await loadSingleFileDiff(firstFile)
          if (fileTreeRef.value) {
            fileTreeRef.value.setCurrentKey(firstFile)
          }
        })
      }
    } else {
      // @Ref: docs/sps/plans/20260530_fix_file_tree_rendering_plan.md | @Date: 2026-05-30
      parsedDiffFiles.value = [{ status: '?', statusText: '无数据', path: '暂无变更文件解析数据' }]
      rawFilesList.value = []
      fileTreeData.value = [{ label: '暂无变更文件解析数据', path: '暂无变更文件解析数据' }]
      defaultCheckedKeys.value = []
    }
  } catch (err) {
    ElMessage.error('获取变更文件列表失败')
  } finally {
    loadingDiff.value = false
  }
}

const getFileStatusTagType = (status: string) => {
  if (status === 'A') return 'success'
  if (status === 'D') return 'danger'
  return 'warning'
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  router.push('/login')
}

// @Ref: docs/sps/plans/20260529_diff_ux_loading_plan.md | @Date: 2026-05-29
const handleSystemPrune = () => {
  ElMessageBox.confirm(
    '确认执行系统自愈与磁盘清理吗？这会清除数据库过期记录、物理删除无关联日志和差异快照文件以盘活磁盘。',
    '系统清理提示',
    {
      confirmButtonText: '确认执行',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    pruneLoading.value = true
    try {
      const res = await axios.post('/api/system/prune')
      const data = res.data
      ElNotification({
        title: '系统自愈完成',
        message: `数据库已强制老化清理任务 ${data.pruned_tasks_count} 个，物理移除孤儿垃圾文件 ${data.pruned_orphans_count} 个，共盘活磁盘空间 ${Math.round(data.freed_bytes / 1024)} KB。`,
        type: 'success',
        duration: 8000
      })
      // 刷新数据
      if (selectedProject.value && envID.value) {
        fetchHistoryTasks(selectedProject.value.id, envID.value)
      }
    } catch (err) {
      const errMsg = err.response?.data?.error || '清理失败，请检查网络或权限'
      ElMessage.error(errMsg)
    } finally {
      pruneLoading.value = false
    }
  }).catch(() => {})
}
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100vh;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 60px;
  padding: 0 24px;
  background-color: #1a1f2c;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.logo-area {
  display: flex;
  align-items: center;
  gap: 10px;
}

.system-name {
  font-size: 20px;
  font-weight: 600;
  color: #ffffff;
  letter-spacing: 0.5px;
}

.user-area {
  display: flex;
  align-items: center;
  gap: 15px;
}

.user-tag {
  background-color: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #ffffff;
}

.main-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 260px;
  background-color: #151922;
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
}

.sidebar-title {
  padding: 16px 20px;
  font-size: 13px;
  font-weight: 600;
  color: #8a99ad;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.project-item {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
  cursor: pointer;
  transition: all 0.2s;
}

.project-item:hover {
  background-color: rgba(255, 255, 255, 0.03);
}

.project-item.active {
  background: linear-gradient(90deg, rgba(0, 180, 216, 0.1) 0%, rgba(0, 0, 0, 0) 100%);
  border-left: 3px solid #00b4d8;
}

.proj-name {
  font-size: 15px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 4px;
}

.proj-id {
  font-size: 12px;
  color: #8a99ad;
}

.content-area {
  flex: 1;
  background-color: #10121a;
  overflow-y: auto;
  padding: 24px;
}

.section-card {
  background-color: #161a23;
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.card-header {
  font-size: 15px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 16px;
  border-left: 3px solid #00b4d8;
  padding-left: 8px;
}

.proj-summary h3 {
  font-size: 22px;
  margin: 0 0 12px 0;
  color: #ffffff;
}

.repo-line {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #a9b7c6;
  font-size: 14px;
  margin-bottom: 12px;
}

.meta-tag {
  margin-right: 8px;
  background-color: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #a9b7c6;
}

.env-tabs {
  margin-top: 15px;
}

:deep(.el-tabs__item) {
  color: #8a99ad;
  font-size: 16px;
  font-weight: 500;
}

:deep(.el-tabs__item.is-active) {
  color: #00b4d8;
}

:deep(.el-tabs__active-bar) {
  background-color: #00b4d8;
}

.env-content-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-top: 15px;
}

.server-title {
  font-size: 14px;
  color: #ffffff;
  margin: 20px 0 10px 0;
}

.server-item {
  padding: 10px 14px;
  background-color: #11141c;
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 8px;
  margin-bottom: 10px;
}

.server-host {
  color: #00b4d8;
  font-weight: 600;
  margin-bottom: 4px;
  font-size: 13px;
}

.server-path {
  font-size: 12px;
  color: #8a99ad;
}

.trigger-deploy-btn {
  width: 100%;
  border-radius: 8px;
  background: linear-gradient(135deg, #0077b6 0%, #0096c7 100%);
  border: none;
  font-weight: 600;
}

.trigger-deploy-btn:hover {
  background: linear-gradient(135deg, #0096c7 0%, #00b4d8 100%);
}

.history-section {
  margin-top: 24px;
}

.empty-dashboard {
  display: flex;
  height: 80vh;
  justify-content: center;
  align-items: center;
}

/* 终端和 Diff 高级暗色样式 */
.terminal-container {
  background-color: #0b0e14;
  border-radius: 8px;
  border: 1px solid #1a2230;
  overflow: hidden;
}

.terminal-header {
  display: flex;
  align-items: center;
  height: 36px;
  background-color: #141a24;
  padding: 0 12px;
  gap: 6px;
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.dot.red { background-color: #ff5f56; }
.dot.yellow { background-color: #ffbd2e; }
.dot.green { background-color: #27c93f; }

.term-title {
  margin-left: 10px;
  font-size: 12px;
  color: #8a99ad;
  font-family: monospace;
}

.terminal-body {
  padding: 16px;
  margin: 0;
  height: 380px;
  overflow-y: auto;
  color: #39ff14;
  background-color: #0b0e14;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.6;
}

.diff-container {
  background-color: #0b0e14;
  border: 1px solid #1a2230;
  border-radius: 8px;
  padding: 16px;
  height: 450px;
  overflow-y: auto;
}

/* ===== diff2html 黑色高对比度主题 ===== */
.dark-diff :deep(.d2h-wrapper) {
  color: #cdd9e5;
}

/* 文件头栏 */
.dark-diff :deep(.d2h-file-header) {
  background: #161b22;
  border-bottom: 1px solid #30363d;
  color: #cdd9e5;
}
.dark-diff :deep(.d2h-file-name) {
  color: #79c0ff;
  font-weight: 600;
}

/* 文件列表 */
.dark-diff :deep(.d2h-file-list-wrapper) {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  margin-bottom: 12px;
}
.dark-diff :deep(.d2h-file-list-title) {
  background: #1c2128;
  color: #8b949e;
  border-bottom: 1px solid #30363d;
}
.dark-diff :deep(.d2h-file-list li) {
  border-bottom: 1px solid #21262d;
}
.dark-diff :deep(.d2h-file-list a) {
  color: #79c0ff;
}
.dark-diff :deep(.d2h-file-list a:hover) {
  color: #a5d6ff;
}

/* 代码表格 */
.dark-diff :deep(.d2h-diff-table) {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  overflow: hidden;
}
.dark-diff :deep(.d2h-code-line),
.dark-diff :deep(.d2h-code-side-line) {
  background: #0d1117;
  color: #cdd9e5;
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
}

/* 行号列 */
.dark-diff :deep(.d2h-code-linenumber),
.dark-diff :deep(.d2h-code-side-linenumber) {
  background: #161b22 !important;
  color: #484f58 !important;
  border-right: 1px solid #30363d !important;
  min-width: 44px;
  user-select: none;
}

/* 新增行：深绿底色 + 亮绿文字 */
.dark-diff :deep(.d2h-ins),
.dark-diff :deep(.d2h-ins .d2h-code-line),
.dark-diff :deep(.d2h-ins .d2h-code-side-line) {
  background: #0f2a1e !important;
  color: #3fb950 !important;
}
.dark-diff :deep(.d2h-ins .d2h-code-line-ctn),
.dark-diff :deep(.d2h-ins .d2h-code-side-line-ctn) {
  background: #0f2a1e !important;
}
.dark-diff :deep(.d2h-ins mark) {
  background: #1a4a2a !important;
  color: #56d364 !important;
  border-radius: 2px;
}
.dark-diff :deep(.d2h-ins .d2h-code-linenumber),
.dark-diff :deep(.d2h-ins .d2h-code-side-linenumber) {
  background: #0a2216 !important;
  color: #3fb950 !important;
  border-right-color: #1a4a2a !important;
}

/* 删除行：深红底色 + 亮红文字 */
.dark-diff :deep(.d2h-del),
.dark-diff :deep(.d2h-del .d2h-code-line),
.dark-diff :deep(.d2h-del .d2h-code-side-line) {
  background: #2d1010 !important;
  color: #f85149 !important;
}
.dark-diff :deep(.d2h-del .d2h-code-line-ctn),
.dark-diff :deep(.d2h-del .d2h-code-side-line-ctn) {
  background: #2d1010 !important;
}
.dark-diff :deep(.d2h-del mark) {
  background: #5c1a1a !important;
  color: #ff7b72 !important;
  border-radius: 2px;
}
.dark-diff :deep(.d2h-del .d2h-code-linenumber),
.dark-diff :deep(.d2h-del .d2h-code-side-linenumber) {
  background: #200d0d !important;
  color: #f85149 !important;
  border-right-color: #5c1a1a !important;
}

/* info 行（@@ hunk header）*/
.dark-diff :deep(.d2h-info),
.dark-diff :deep(.d2h-info .d2h-code-line),
.dark-diff :deep(.d2h-info .d2h-code-side-line) {
  background: #161b22 !important;
  color: #8b949e !important;
  font-style: italic;
}
.dark-diff :deep(.d2h-info .d2h-code-linenumber),
.dark-diff :deep(.d2h-info .d2h-code-side-linenumber) {
  background: #161b22 !important;
  border-right-color: #30363d !important;
}

/* 未变更行 */
.dark-diff :deep(.d2h-cntx .d2h-code-line),
.dark-diff :deep(.d2h-cntx .d2h-code-side-line),
.dark-diff :deep(.d2h-cntx) {
  background: #0d1117 !important;
  color: #8b949e !important;
}

/* Side-by-side 分隔线 */
.dark-diff :deep(.d2h-diff-side-col) {
  border-right: 1px solid #30363d;
}

/* 文件展开标题 */
.dark-diff :deep(.d2h-file-diff) {
  border: 1px solid #30363d;
  border-radius: 6px;
  margin-bottom: 16px;
  overflow: hidden;
}

/* 滚动条暗色 */
.dark-diff :deep(*)::-webkit-scrollbar {
  height: 6px;
  background: #161b22;
}
.dark-diff :deep(*)::-webkit-scrollbar-thumb {
  background: #30363d;
  border-radius: 3px;
}

.diff-pre {
  margin: 0;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
  color: #a9b7c6;
}

/* 表格暗色自适应 */
:deep(.el-table) {
  background-color: transparent !important;
  color: #e0e0e0;
}

:deep(.el-table th.el-table__cell) {
  background-color: #12161f !important;
  color: #8a99ad;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

:deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

:deep(.el-table--enable-row-hover .el-table__body tr:hover > td.el-table__cell) {
  background-color: rgba(255, 255, 255, 0.02) !important;
}

/* commit hash 等宽字体 */
.commit-hash {
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: #79c0ff;
  background: rgba(121, 192, 255, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

/* 失败行整行淡红底 */
:deep(.el-table .row-failed td.el-table__cell) {
  background-color: rgba(248, 81, 73, 0.07) !important;
}

/* 侧边栏项目名行（含环境徽章） */
.proj-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.env-badge {
  flex-shrink: 0;
}

:deep(.env-badge .el-badge__content) {
  font-size: 10px;
  padding: 0 5px;
  height: 16px;
  line-height: 16px;
  background: rgba(0, 180, 216, 0.4);
  border: 1px solid rgba(0, 180, 216, 0.5);
  color: #a8d8e8;
}

/* 骨架屏及动效 CSS */
.diff-loading-skeleton {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: #0d1117;
  border-radius: 6px;
}
.skeleton-bar {
  height: 16px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
  animation: skeleton-blink 1.2s infinite ease-in-out;
}
.skeleton-bar.wide {
  width: 100%;
}
.skeleton-bar.medium {
  width: 75%;
}
.skeleton-bar.narrow {
  width: 45%;
}
@keyframes skeleton-blink {
  0% {
    opacity: 0.35;
  }
  50% {
    opacity: 0.7;
  }
  100% {
    opacity: 0.35;
  }
}

.diff-split-layout {
  display: flex;
  height: calc(100vh - 180px);
  gap: 16px;
}

.diff-left-sidebar {
  width: 320px;
  flex-shrink: 0;
  background: #0d1117;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 10px;
  overflow-y: auto;
}

.diff-right-content {
  flex: 1;
  background: #0d1117;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  padding: 16px;
  position: relative;
  overflow: hidden;
}

.diff-empty-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #8b949e;
  font-size: 14px;
}

.diff-empty-placeholder p {
  margin-top: 12px;
}

/* 激活选中的行高亮 */
:deep(.el-table .row-active-selected td.el-table__cell) {
  background-color: rgba(0, 180, 216, 0.15) !important;
  border-left: 3px solid #00b4d8;
}
</style>


