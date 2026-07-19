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
            <div class="action-footer">
              <el-button type="primary" @click="saveHooks(env)">保存脚本</el-button>
              <el-button type="success" @click="startDeploy(env)">
                <el-icon class="el-icon--left"><Promotion /></el-icon> 立即部署
              </el-button>
            </div>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { VideoPlay, VideoPause, Promotion, CopyDocument } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id

const environments = ref<any[]>([])
const dialogVisible = ref(false)
const creating = ref(false)
const activeNames = ref<string[]>([])

const form = ref({
  name: '',
  branch: 'main',
  deploy_type: 'symlink'
})

const fetchEnvironments = async () => {
  try {
    const res = await axios.get('/api/projects')
    const currentProject = res.data.find((p: any) => p.id == projectId)
    if (currentProject && currentProject.environments) {
      environments.value = currentProject.environments
      if (environments.value.length > 0) {
        activeNames.value = [environments.value[0].name]
      }
    }
  } catch (e) {
    ElMessage.error('获取环境列表失败')
  }
}

const addEnvironment = async () => {
  if (!form.value.name || !form.value.branch) {
    ElMessage.warning('请填写环境和分支名称')
    return
  }
  
  creating.value = true
  try {
    const res = await axios.post(`/api/projects/${projectId}/environments`, form.value)
    ElMessage.success('创建环境成功')
    environments.value = res.data.environments
    dialogVisible.value = false
    activeNames.value.push(form.value.name)
    form.value.name = ''
  } catch (e: any) {
    ElMessage.error(e.response?.data || '创建失败')
  } finally {
    creating.value = false
  }
}

const saveHooks = async (env: any) => {
  try {
    await axios.put(`/api/projects/${projectId}/environments/${env.name}`, {
      pre_deploy: env.pre_deploy,
      post_deploy: env.post_deploy
    })
    ElMessage.success(`${env.name} 环境 Hook 脚本保存成功`)
  } catch (e: any) {
    ElMessage.error(e.response?.data || '保存失败')
  }
}

const startDeploy = async (env: any) => {
  try {
    await ElMessageBox.confirm(`确定要部署环境 [${env.name}] 吗？\n部署分支: ${env.branch}`, '部署确认', {
      confirmButtonText: '确定部署',
      cancelButtonText: '取消',
      type: 'warning',
    })
    
    const commitHash = "latest" 
    
    const res = await axios.post('/api/deployments', {
      project_id: parseInt(projectId as string),
      env_name: env.name,
      commit_hash: commitHash
    })
    
    const deployId = res.data.ID
    ElMessage.success('已触发部署，正在前往控制台...')
    router.push(`/deployments/${deployId}`)
    
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data || '触发部署失败')
    }
  }
}

onMounted(() => {
  fetchEnvironments()
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
</style>
