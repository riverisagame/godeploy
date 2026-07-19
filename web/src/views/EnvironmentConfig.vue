<template>
  <div class="env-config">
    <div class="header-actions">
      <h2>环境配置 (项目 ID: {{ projectId }})</h2>
      <el-button type="primary" @click="dialogVisible = true">新建环境</el-button>
    </div>

    <!-- Environments List -->
    <el-collapse style="margin-top: 20px;" class="collapse-container">
      <el-collapse-item v-for="env in environments" :key="env.name" :title="env.name + ' (' + env.branch + ')'">
        <el-form label-position="top" class="hook-form">
          <el-form-item label="构建类型" class="full-row">
            <el-tag>{{ env.deploy_type }}</el-tag>
          </el-form-item>
          <el-form-item label="前置脚本 (Pre-Deploy)">
            <el-input 
              v-model="env.pre_deploy" 
              type="textarea" 
              :rows="5"
              placeholder="部署前执行的命令，如 npm run build">
            </el-input>
          </el-form-item>
          <el-form-item label="后置脚本 (Post-Deploy)">
            <el-input 
              v-model="env.post_deploy" 
              type="textarea" 
              :rows="5"
              placeholder="部署后执行的命令，如 systemctl restart xxx">
            </el-input>
          </el-form-item>
          <el-form-item class="full-row" style="display: flex; gap: 16px;">
            <el-button type="primary" size="large" @click="saveHooks(env)" style="width: 200px;">保存 Hook 脚本</el-button>
            <el-button type="success" size="large" @click="startDeploy(env)" style="width: 200px;">
              <el-icon><VideoPlay /></el-icon> 立即部署
            </el-button>
          </el-form-item>
        </el-form>
      </el-collapse-item>
    </el-collapse>

    <!-- Create Environment Dialog -->
    <el-dialog title="新建环境" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="环境名称">
          <el-input v-model="form.name" placeholder="例如: test / prod"></el-input>
        </el-form-item>
        <el-form-item label="分支名称">
          <el-input v-model="form.branch" placeholder="例如: main"></el-input>
        </el-form-item>
        <el-form-item label="部署类型">
          <el-select v-model="form.deploy_type" placeholder="请选择">
            <el-option label="Symlink (软链接)" value="symlink"></el-option>
            <el-option label="Docker" value="docker"></el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="addEnvironment" :loading="creating">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { VideoPlay } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id

const environments = ref<any[]>([])
const dialogVisible = ref(false)
const creating = ref(false)

const form = ref({
  name: '',
  branch: 'main',
  deploy_type: 'symlink'
})

const fetchEnvironments = async () => {
  try {
    const res = await axios.get('/api/projects')
    // Find the current project
    const currentProject = res.data.find((p: any) => p.id == projectId)
    if (currentProject && currentProject.environments) {
      environments.value = currentProject.environments
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
    await ElMessageBox.confirm(`确定要部署环境 [${env.name}] 吗？`, '部署确认', {
      confirmButtonText: '确定部署',
      cancelButtonText: '取消',
      type: 'warning',
    })
    
    // Simulate getting a commit hash for deploy (could add an input in real app)
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
  padding: 24px;
  height: calc(100vh - 60px); /* 减去顶部 header 高度 */
  overflow-y: auto;
  box-sizing: border-box;
}
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
h2 {
  margin: 0;
  color: var(--text-primary);
  font-family: var(--mono);
  font-size: 24px;
}

/* 充分利用空间 */
.collapse-container {
  width: 100%;
}
.hook-form {
  width: 100%; /* 充分利用空间 */
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}
.full-row {
  grid-column: 1 / -1;
}

:deep(.el-collapse) {
  border-top: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
}
:deep(.el-collapse-item__header) {
  background-color: var(--bg-card);
  color: var(--text-primary);
  font-family: var(--mono);
  font-size: 16px;
  border-bottom: 1px solid var(--border-color);
}
:deep(.el-collapse-item__wrap) {
  background-color: var(--bg-dark);
  border-bottom: 1px solid var(--border-color);
}
:deep(.el-collapse-item__content) {
  padding: 20px;
  color: var(--text-secondary);
}
:deep(.el-form-item__label) {
  color: var(--text-primary);
  font-family: var(--mono);
}
:deep(.el-textarea__inner) {
  background-color: #000;
  color: #a6e22e; /* 类似代码高亮的亮绿色 */
  border: 1px solid var(--border-color);
  font-family: var(--mono);
  box-shadow: none;
}
:deep(.el-textarea__inner:focus) {
  border-color: var(--accent-blue);
}

:deep(.el-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
}
:deep(.el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
}
:deep(.el-input__wrapper), :deep(.el-select__wrapper) {
  background-color: var(--bg-dark);
  box-shadow: 0 0 0 1px var(--border-color) inset;
}
:deep(.el-input__wrapper.is-focus), :deep(.el-select__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--accent-blue) inset !important;
}
</style>
