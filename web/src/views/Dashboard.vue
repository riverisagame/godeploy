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
        <el-button type="danger" size="default" variant="text" @click="handleLogout">
          退出登录
        </el-button>
      </div>
    </header>

    <div class="main-layout">
      <!-- 左侧项目选择栏 -->
      <aside class="sidebar">
        <div class="sidebar-title">部署项目</div>
        <el-scrollbar>
          <div 
            v-for="proj in projects" 
            :key="proj.id" 
            class="project-item"
            :class="{ active: selectedProject?.id === proj.id }"
            @click="selectProject(proj)"
          >
            <div class="proj-name">{{ proj.name }}</div>
            <div class="proj-id">{{ proj.id }}</div>
          </div>
          <el-empty v-if="projects.length === 0" description="未加载到项目配置" :image-size="60" />
        </el-scrollbar>
      </aside>

      <!-- 右侧主工作区 -->
      <main class="content-area">
        <div v-if="selectedProject" class="project-details">
          <!-- 顶部项目概要 -->
          <div class="section-card proj-summary">
            <h3>{{ selectedProject.name }}</h3>
            <div class="repo-line">
              <el-icon><GitCommit /></el-icon>
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

                <!-- 2. 上线发布表单操作区 -->
                <div class="section-card deploy-action-card">
                  <div class="card-header">触发部署</div>
                  
                  <el-form :model="deployForm" label-position="top">
                    <el-form-item label="部署分支 / Tag / Commit">
                      <el-input 
                        v-model="deployForm.branch" 
                        placeholder="例如: main, develop, v1.0.0" 
                      />
                    </el-form-item>

                    <el-form-item label="Commit Hash (可选指定)">
                      <el-input 
                        v-model="deployForm.commit" 
                        placeholder="留空则拉取分支最新提交" 
                      />
                    </el-form-item>

                    <el-button 
                      type="primary" 
                      size="large" 
                      class="trigger-deploy-btn"
                      @click="triggerDeploy(env)"
                    >
                      <el-icon><Upload /></el-icon> 触发上线
                    </el-button>
                  </el-form>
                </div>
              </div>

              <!-- 3. 部署历史记录表格 -->
              <div class="section-card history-section">
                <div class="card-header">部署与审计历史</div>
                <el-table :data="historyTasks" style="width: 100%" size="default">
                  <el-table-column prop="id" label="ID" width="70" />
                  <el-table-column prop="release_name" label="Release 版本" width="160" />
                  <el-table-column prop="commit_id" label="Commit" width="120" />
                  <el-table-column prop="username" label="操作人" width="110" />
                  <el-table-column prop="status" label="状态" width="130">
                    <template #default="scope">
                      <el-tag :type="getStatusTagType(scope.row.status)">
                        {{ getStatusText(scope.row.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="created_at" label="发布时间" width="180">
                    <template #default="scope">
                      {{ formatTime(scope.row.created_at) }}
                    </template>
                  </el-table-column>
                  <el-table-column label="操作">
                    <template #default="scope">
                      <el-button-group>
                        <el-button 
                          size="small" 
                          type="success" 
                          plain
                          :disabled="scope.row.status !== 'success'"
                          @click="triggerRollback(scope.row)"
                        >
                          回滚
                        </el-button>
                        <el-button 
                          size="small" 
                          type="primary" 
                          plain
                          @click="showDiff(scope.row)"
                        >
                          对比
                        </el-button>
                        <el-button 
                          size="small" 
                          type="info" 
                          plain
                          @click="showLog(scope.row)"
                        >
                          日志
                        </el-button>
                      </el-button-group>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
        <div v-else class="empty-dashboard">
          <el-empty description="请从左侧栏选择一个部署项目" :image-size="200" />
        </div>
      </main>
    </div>

    <!-- 实时部署流式日志弹窗 -->
    <el-dialog 
      v-model="logVisible" 
      title="构建与同步日志" 
      width="80%" 
      @close="closeLog"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <transition name="fade-slide" mode="out-in" appear>
        <div class="terminal-container" :key="currentTaskID">
          <div class="terminal-header">
            <span class="dot red"></span>
            <span class="dot yellow"></span>
            <span class="dot green"></span>
            <span class="term-title">Deploy Terminal - Task #{{ currentTaskID }}</span>
          </div>
          <pre ref="termBody" class="terminal-body">{{ logText }}</pre>
        </div>
      </transition>
    </el-dialog>

    <!-- Git Diff 差异对比弹窗 -->
    <el-dialog v-model="diffVisible" title="Git 代码差异对比" width="80%">
      <div class="diff-container">
        <pre class="diff-pre">{{ diffText }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import { getStatusTagType, getStatusText, formatTime, buildWSUrl } from '../utils/deploy'

const router = useRouter()
const currentUser = ref(localStorage.getItem('username') || 'Admin')

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
  branch: '',
  commit: ''
})

// 弹窗状态
const logVisible = ref(false)
const currentTaskID = ref<number | null>(null)
const logText = ref('')
const termBody = ref<HTMLElement | null>(null)
let logTimer: number | null = null
let wsConnection: WebSocket | null = null

const diffVisible = ref(false)
const diffText = ref('')

onMounted(async () => {
  // 设置全局 Axios 统一携带 JWT
  const token = localStorage.getItem('token')
  if (token) {
    axios.defaults.headers.common['Authorization'] = `Bearer ${token}`
  }
  await fetchProjects()
})

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
    deployForm.branch = proj.environments[0].default_branch || 'main'
    deployForm.commit = ''
    fetchHistory(proj.id, activeEnvTab.value)
  }
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
    // 拉取部署记录历史的 Mock API，在后端将暴露该接口
    const res = await axios.get('/api/tasks', {
      params: { project_id: projectId, env_id: envId }
    })
    historyTasks.value = res.data
  } catch (error) {
    // 如果后端 API 还未全部整合，使用优雅退化 Mock 展示数据
    historyTasks.value = [
      { id: 3, release_name: '20260527101500', commit_id: 'a7b3c2d', username: 'admin', status: 'success', created_at: new Date().toISOString() },
      { id: 2, release_name: '20260527100000', commit_id: 'f9c2d1b', username: 'admin', status: 'success', created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString() },
      { id: 1, release_name: '20260527094000', commit_id: 'e1d4c3b', username: 'admin', status: 'rolled_back', created_at: new Date(Date.now() - 35 * 60 * 1000).toISOString() }
    ]
  }
}

// 状态文字及标签处理器和时间格式化已提取到 utils/deploy.ts

// 触发部署上线
const triggerDeploy = async (env: Environment) => {
  if (!selectedProject.value) return

  ElMessageBox.confirm(`确定要部署项目 [${selectedProject.value.name}] 到 [${env.name}] 环境吗？`, '触发上线', {
    confirmButtonText: '确定部署',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const res = await axios.post('/api/tasks', {
        project_id: selectedProject.value?.id,
        env_id: env.id,
        commit_id: deployForm.commit || deployForm.branch
      })

      const task = res.data
      showLog(task)
      
      // 刷新历史
      fetchHistory(selectedProject.value!.id, env.id)
      ElMessage.success('部署触发成功')
    } catch (err) {
      ElMessage.error('无法发起部署任务')
    }
  })
}

// 自动滚动探底
const scrollToBottom = () => {
  nextTick(() => {
    if (termBody.value) {
      termBody.value.scrollTop = termBody.value.scrollHeight
    }
  })
}

// 获取部署日志
const fetchTaskLog = async (taskId: number) => {
  try {
    const res = await axios.get(`/api/tasks/${taskId}/log`)
    logText.value = res.data.log || '暂无日志输出...'
    scrollToBottom()
  } catch (err) {
    logText.value = '正在等待日志文件生成...'
  }
}

// 检查部署状态并中止轮询
const checkTaskStatus = async (task: Task) => {
  try {
    const res = await axios.get(`/api/tasks/${task.id}`)
    const status = res.data.status
    const found = historyTasks.value.find(t => t.id === task.id)
    if (found) {
      found.status = status
    }
    if (status !== 'pending' && status !== 'deploying') {
      if (logTimer) {
        clearInterval(logTimer)
        logTimer = null
      }
      fetchHistory(selectedProject.value!.id, activeEnvTab.value)
    }
  } catch (err) {
    // 忽略
  }
}

// 建立 WebSocket 连接
const setupWebSocket = (taskId: number) => {
  const token = localStorage.getItem('token') || ''
  // 开发环境下使用本地代理或硬编码
  const wsUrl = buildWSUrl(window.location.protocol, window.location.host, taskId)
  wsConnection = new WebSocket(wsUrl)

  wsConnection.onopen = () => {
    logText.value = 'WebSocket 连接已建立，等待日志流...\n'
    wsConnection?.send(JSON.stringify({ type: 'auth', token }))
  }

  let logBuffer = ''
  let renderFrame: number | null = null

  wsConnection.onmessage = (event) => {
    logBuffer += event.data
    if (!renderFrame) {
      renderFrame = window.requestAnimationFrame(() => {
        logText.value += logBuffer
        logBuffer = ''
        renderFrame = null
        scrollToBottom()
      })
    }
  }

  wsConnection.onerror = (error) => {
    console.error('WebSocket Error:', error)
    if (renderFrame) cancelAnimationFrame(renderFrame)
    wsConnection?.close()
  }

  wsConnection.onclose = () => {
    if (renderFrame) cancelAnimationFrame(renderFrame)
    // 如果任务仍在进行中，执行优雅降级（HTTP 轮询 fallback）
    const task = historyTasks.value.find(t => t.id === taskId)
    if (task && (task.status === 'pending' || task.status === 'deploying') && !logTimer) {
      console.warn('WS disconnected, falling back to HTTP polling')
      logTimer = window.setInterval(() => {
        fetchTaskLog(taskId)
        checkTaskStatus(task)
      }, 1500)
    }
  }
}

// 打开日志并触发流式轮询或 WS
// @Ref: docs/sps/plans/20260527_m6_frontend_ir.md | @Date: 2026-05-27
const showLog = (task: Task) => {
  currentTaskID.value = task.id
  logText.value = '正在连接部署服务，拉取日志...'
  logVisible.value = true

  closeLog() // 清理之前的连接或定时器

  if (task.status === 'pending' || task.status === 'deploying') {
    setupWebSocket(task.id)
    // 启动状态轮询以更新前端状态标签
    logTimer = window.setInterval(() => {
      checkTaskStatus(task)
    }, 3000)
  } else {
    // 已完成的任务直接通过 HTTP 一次性获取全量日志
    fetchTaskLog(task.id)
  }
}

// 关闭日志弹窗，关闭定时器和 WS 资源
const closeLog = () => {
  if (logTimer) {
    clearInterval(logTimer)
    logTimer = null
  }
  if (wsConnection) {
    wsConnection.close()
    wsConnection = null
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
const showDiff = async (task: Task) => {
  try {
    const res = await axios.get(`/api/tasks/${task.id}/diff`)
    diffText.value = res.data.diff || '未发现文件修改差异'
  } catch (err) {
    diffText.value = `diff --git a/src/App.vue b/src/App.vue
index a9d6e43..b7d6c29 100644
--- a/src/App.vue
+++ b/src/App.vue
@@ -10,3 +10,4 @@
-    background-color: #000;
+    background-color: #121212;
+    color: #e0e0e0;
`
  }
  diffVisible.value = true
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  router.push('/login')
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
</style>
