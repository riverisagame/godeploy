<template>
  <div class="env-config">
    <div class="header-actions">
      <div>
        <h2>环境配置</h2>
        <p class="subtitle">管理项目 (ID: {{ projectId }}) 的部署环境、分支及 Hook 脚本</p>
      </div>
      <el-button type="primary" @click="dialogVisible = true" size="large">新建环境</el-button>
    </div>

    <!-- Environments List -->
    <el-collapse v-model="activeNames" class="custom-collapse" v-if="environments.length > 0">
      <el-collapse-item v-for="env in environments" :key="env.name" :name="env.name">
        <template #title>
          <div class="env-title">
            <span class="env-name">{{ env.name }}</span>
            <el-tag size="small" type="info" class="mono-tag" effect="plain"><el-icon><CopyDocument /></el-icon> {{ env.branch }}</el-tag>
            <el-tag size="small" type="success" effect="dark" class="deploy-type-tag">{{ env.deploy_type.toUpperCase() }}</el-tag>
          </div>
        </template>
        <div class="env-content-wrapper">
          <el-form label-position="top" class="hook-form">
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="前置脚本 (Pre-Deploy)">
                  <template #label>
                    <div class="label-with-icon">
                      <el-icon><VideoPause /></el-icon> 前置脚本 (Pre-Deploy)
                    </div>
                  </template>
                  <el-input 
                    v-model="env.pre_deploy" 
                    type="textarea" 
                    :rows="6"
                    placeholder="部署前执行的命令，如 npm run build"
                    class="code-textarea">
                  </el-input>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="后置脚本 (Post-Deploy)">
                  <template #label>
                    <div class="label-with-icon">
                      <el-icon><VideoPlay /></el-icon> 后置脚本 (Post-Deploy)
                    </div>
                  </template>
                  <el-input 
                    v-model="env.post_deploy" 
                    type="textarea" 
                    :rows="6"
                    placeholder="部署后执行的命令，如 systemctl restart xxx"
                    class="code-textarea">
                  </el-input>
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="24" style="margin-top: 16px;">
              <el-col :span="12">
                <el-form-item label="部署目标路径 (Deploy Path)">
                  <el-input v-model="env.deploy_path" placeholder="例如: /var/www/html/myapp"></el-input>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="关联服务器">
                  <el-select v-model="env.server_ids" multiple placeholder="请选择要部署的服务器" style="width: 100%;">
                    <el-option
                      v-for="server in allServers"
                      :key="server.id"
                      :label="server.name + ' (' + server.ip + ')'"
                      :value="server.id">
                    </el-option>
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <!-- @Ref: docs/sps/plans/20260720_system_health_ir.md | @Date: 2026-07-20 -->
            <EnvVarEditor v-model="env.env_vars" />

            <div class="action-footer">
              <el-button type="primary" @click="saveConfig(env)">保存配置</el-button>
              <el-button type="success" @click="startDeploy(env)">
                <el-icon class="el-icon--left"><Promotion /></el-icon> 立即部署
              </el-button>
            </div>

            <!-- @Ref: docs/sps/plans/20260720_system_health_ir.md | @Date: 2026-07-20 -->
            <DeployHistory 
              :env="env" 
              :deployments="env.deployments || []" 
              :projectId="projectId" 
              @refresh="fetchDeployHistory(env)" 
            />
          </el-form>
        </div>
      </el-collapse-item>
    </el-collapse>
    
    <el-empty v-else description="暂无环境配置" :image-size="200"></el-empty>

    <!-- Create Environment Dialog -->
    <el-dialog title="新建环境" v-model="dialogVisible" width="480px" destroy-on-close class="custom-dialog">
      <el-form :model="form" label-position="top" class="custom-form">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="环境名称">
              <el-input v-model="form.name" placeholder="例如: test / prod" size="large"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="分支名称">
              <el-input v-model="form.branch" placeholder="例如: main" size="large"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="部署类型">
          <el-select v-model="form.deploy_type" placeholder="请选择" size="large" style="width: 100%">
            <el-option label="Symlink (软链接, 推荐)" value="symlink"></el-option>
            <el-option label="Docker (容器化)" value="docker"></el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="addEnvironment" :loading="creating">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Pre-Deploy Diff Dialog -->
    <el-dialog title="部署前确认 (Commit Diff)" v-model="diffDialogVisible" width="640px" destroy-on-close class="custom-dialog">
      <div v-if="loadingDiff" style="text-align: center; padding: 40px; color: var(--text-secondary);">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        <p>正在获取变更记录...</p>
      </div>
      <div v-else>
        <div v-if="diffCommits.length > 0">
          <p style="color: var(--text-secondary); margin-bottom: 16px;">以下是自上次成功部署以来的代码变更：</p>
          <div class="commit-list">
            <div v-for="commit in diffCommits" :key="commit.hash" class="commit-item">
              <div class="commit-header">
                <span class="commit-hash"><el-icon><CopyDocument /></el-icon> {{ commit.hash.substring(0, 7) }}</span>
                <span class="commit-author"><el-icon><User /></el-icon> {{ commit.author }}</span>
                <span class="commit-time"><el-icon><Clock /></el-icon> {{ new Date(commit.date).toLocaleString() }}</span>
              </div>
              <div class="commit-msg">{{ commit.message }}</div>
            </div>
          </div>
        </div>
        <div v-else>
          <el-alert title="太棒了！或者...等等？" description="没有发现新的代码变更。您可以强制触发部署，或者取消操作。" type="info" show-icon :closable="false" style="margin-bottom: 16px; background-color: rgba(255, 255, 255, 0.05); border: 1px solid var(--border-color); color: var(--text-primary);" />
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="diffDialogVisible = false">取消</el-button>
          <el-button type="success" @click="confirmDeploy" :loading="triggeringDeploy">
            <el-icon class="el-icon--left"><Promotion /></el-icon> 确定部署 (Latest)
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { ElMessage } from 'element-plus'
import { VideoPlay, VideoPause, Promotion, CopyDocument, Loading, User, Clock } from '@element-plus/icons-vue'
import EnvVarEditor from '../components/EnvVarEditor.vue'
import DeployHistory from '../components/DeployHistory.vue'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id

import type { Environment, CommitInfo, Project, Server } from '../types'

const environments = ref<Environment[]>([])
const allServers = ref<Server[]>([])
const dialogVisible = ref(false)
const creating = ref(false)
const activeNames = ref<string[]>([])

const diffDialogVisible = ref(false)
const loadingDiff = ref(false)
const diffCommits = ref<CommitInfo[]>([])
const triggeringDeploy = ref(false)
const currentDeployEnv = ref<Environment | null>(null)

const form = ref({
  name: '',
  branch: 'main',
  deploy_type: 'symlink'
})

const fetchEnvironments = async () => {
  try {
    const res = await api.getProjects()
    const currentProject = res.data.find((p: Project) => p.id === parseInt(projectId as string))
    if (currentProject && currentProject.environments) {
      environments.value = currentProject.environments
      environments.value.forEach(env => fetchDeployHistory(env))
      if (environments.value.length > 0) {
        activeNames.value = [environments.value[0].name]
      }
    }
  } catch (e) {
    ElMessage.error('获取环境列表失败')
  }
}

const fetchAllServers = async () => {
  try {
    const res = await api.getServers()
    allServers.value = res.data
  } catch (e) {
    console.error('Failed to load servers', e)
  }
}

const addEnvironment = async () => {
  if (!form.value.name || !form.value.branch) {
    ElMessage.warning('请填写环境和分支名称')
    return
  }
  
  creating.value = true
  try {
    const res = await api.createEnvironment(projectId as string, form.value)
    ElMessage.success('创建环境成功')
    environments.value = res.data.environments || []
    dialogVisible.value = false
    activeNames.value.push(form.value.name)
    form.value.name = ''
  } catch (e: any) {
    ElMessage.error(e.response?.data || '创建失败')
  } finally {
    creating.value = false
  }
}



const saveConfig = async (env: Environment) => {
  try {
    await api.updateEnvironment(projectId as string, env.name, {
      pre_deploy: env.pre_deploy,
      post_deploy: env.post_deploy,
      deploy_path: env.deploy_path,
      server_ids: env.server_ids,
      env_vars: env.env_vars || []
    })
    ElMessage.success(`${env.name} 环境配置保存成功`)
  } catch (e: any) {
    ElMessage.error(e.response?.data || '保存失败')
  }
}
const startDeploy = async (env: Environment) => {
  currentDeployEnv.value = env
  diffDialogVisible.value = true
  loadingDiff.value = true
  diffCommits.value = []
  
  try {
    const res = await api.getEnvironmentDiff(projectId as string, env.name)
    diffCommits.value = res.data || []
  } catch (e: any) {
    ElMessage.warning('获取 Diff 失败，可能是首次部署，允许继续部署。')
  } finally {
    loadingDiff.value = false
  }
}

const confirmDeploy = async () => {
  if (!currentDeployEnv.value) return
  const env = currentDeployEnv.value
  
  triggeringDeploy.value = true
  try {
    const commitHash = "latest" 
    
    const res = await api.createDeployment(projectId as string, env.name, commitHash)
    
    const deployId = res.data.id
    ElMessage.success('已触发部署，正在前往控制台...')
    diffDialogVisible.value = false
    router.push(`/deployments/${deployId}`)
    
  } catch (e: any) {
    ElMessage.error(e.response?.data || '触发部署失败')
  } finally {
    triggeringDeploy.value = false
  }
}

const fetchDeployHistory = async (env: Environment) => {
  try {
    const res = await api.getDeployments(env.id)
    env.deployments = res.data
  } catch (e) {
    console.error('获取历史失败', e)
  }
}



onMounted(() => {
  fetchEnvironments()
  fetchAllServers()
})
</script>

<style scoped>
.env-config {
  padding: 8px;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}
h2 {
  margin: 0 0 8px 0;
  color: var(--text-primary);
  font-size: 28px;
  letter-spacing: -0.5px;
}
.subtitle {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.custom-collapse {
  border-top: none;
  border-bottom: none;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

:deep(.el-collapse-item) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

:deep(.el-collapse-item__header) {
  background-color: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border-color);
  padding: 0 20px;
  height: 60px;
  line-height: 60px;
}

:deep(.el-collapse-item__header.is-active) {
  border-bottom-color: var(--border-color);
}

:deep(.el-collapse-item__wrap) {
  background-color: transparent;
  border-bottom: none;
}

:deep(.el-collapse-item__content) {
  padding: 0;
  color: var(--text-secondary);
}

.env-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.env-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.mono-tag {
  font-family: var(--mono);
}

.deploy-type-tag {
  font-family: var(--mono);
  font-weight: 700;
  letter-spacing: 0.5px;
}

.env-content-wrapper {
  padding: 24px;
}

.label-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--text-primary);
}

.action-footer {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px dashed rgba(255, 255, 255, 0.1);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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

:deep(.code-textarea .el-textarea__inner) {
  background-color: #0d1117;
  color: #a6e22e;
  border: 1px solid var(--border-color);
  font-family: var(--mono);
  font-size: 13px;
  line-height: 1.6;
  padding: 12px;
  box-shadow: none;
  border-radius: 6px;
}
:deep(.code-textarea .el-textarea__inner:focus) {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.3);
}

:deep(.custom-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.6);
}
:deep(.custom-dialog .el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
}
:deep(.custom-form .el-form-item__label) {
  color: var(--text-secondary);
  padding-bottom: 4px;
}

.commit-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
}

.commit-item {
  background-color: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px;
}

.commit-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.commit-hash {
  font-family: var(--mono);
  color: var(--accent-blue);
  display: flex;
  align-items: center;
  gap: 4px;
}

.commit-author {
  display: flex;
  align-items: center;
  gap: 4px;
}

.commit-time {
  display: flex;
  align-items: center;
  gap: 4px;
}

.commit-msg {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.5;
  white-space: pre-wrap;
}
</style>
