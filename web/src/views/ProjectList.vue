<template>
  <div class="project-list">
    <div class="header-actions">
      <div>
        <h2>项目管理</h2>
        <p class="subtitle">管理所有需要发布的项目及环境配置</p>
      </div>
      <el-button v-if="admin" type="primary" @click="dialogVisible = true" size="large">新建项目</el-button>
    </div>

    <el-row :gutter="24" class="project-grid" v-if="projects.length > 0">
      <el-col :xs="24" :sm="12" :md="8" :lg="6" v-for="prj in projects" :key="prj.id" style="margin-bottom: 24px;">
        <el-card shadow="hover" class="project-card cursor-pointer is-hover-shadow" @click="viewEnvs(prj)">
          <div class="card-header">
            <h3>{{ prj.name }}</h3>
            <el-tag size="small" type="info" effect="dark" class="id-tag">ID: {{ prj.id }}</el-tag>
          </div>
          <div class="card-body">
            <div class="info-row">
              <el-icon><Link /></el-icon>
              <span class="truncate" :title="prj.repo_url">{{ prj.repo_url }}</span>
            </div>
            <div class="info-row">
              <el-icon><CopyDocument /></el-icon>
              <span>保留版本数: <strong>{{ prj.keep_releases }}</strong></span>
            </div>
          </div>
          <div class="card-footer" style="justify-content: space-between;">
            <div>
              <template v-if="admin">
                <el-button type="primary" link @click.stop="openEdit(prj)">编辑</el-button>
                <el-popconfirm title="确定删除此项目及其所有环境配置吗？" @confirm="handleDelete(prj.id)" placement="top">
                  <template #reference>
                    <el-button type="danger" link @click.stop>删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </div>
            <el-button type="primary" text bg size="small" @click.stop="viewEnvs(prj)">配置环境 <el-icon class="el-icon--right"><ArrowRight /></el-icon></el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-empty v-else description="暂无项目，请点击右上角新建" :image-size="200"></el-empty>

    <!-- Create / Edit Project Dialog -->
    <el-dialog :title="isEdit ? '编辑项目' : '新建项目'" v-model="dialogVisible" width="480px" destroy-on-close class="custom-dialog" @closed="resetForm">
      <el-form :model="form" label-position="top" class="custom-form">
        <el-form-item label="项目名称">
          <el-input v-model="form.name" placeholder="例如: pdeploy-web" size="large"></el-input>
        </el-form-item>
        <el-form-item label="Git 仓库">
          <el-input v-model="form.repo_url" placeholder="例如: git@github.com:..." size="large"></el-input>
        </el-form-item>
        <el-form-item label="Webhook 密钥 (可选)">
          <el-input v-model="form.webhook_secret" placeholder="用于验证 GitHub/GitLab 请求签名" size="large" show-password></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="saving">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '../api'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { Link, CopyDocument, ArrowRight } from '@element-plus/icons-vue'
import type { Project } from '../types'
import { isAdmin } from '../utils/auth'

const router = useRouter()
const admin = computed(() => isAdmin())

const projects = ref<Project[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const saving = ref(false)
const form = ref({
  name: '',
  repo_url: '',
  webhook_secret: ''
})

const fetchProjects = async () => {
  try {
    const res = await api.getProjects()
    projects.value = res.data
  } catch (e) {
    ElMessage.error('获取项目失败')
  }
}

const resetForm = () => {
  isEdit.value = false
  editId.value = null
  form.value = { name: '', repo_url: '', webhook_secret: '' }
}

const openEdit = (prj: Project) => {
  isEdit.value = true
  editId.value = prj.id
  form.value = { name: prj.name, repo_url: prj.repo_url, webhook_secret: prj.webhook_secret || '' }
  dialogVisible.value = true
}

const handleDelete = async (id: number) => {
  try {
    await api.deleteProject(id)
    ElMessage.success('删除成功')
    fetchProjects()
  } catch (e: any) {
    ElMessage.error(e.response?.data || '删除失败')
  }
}

const handleSubmit = async () => {
  if (!form.value.name || !form.value.repo_url) {
    ElMessage.warning('请填写项目名称和Git仓库地址')
    return
  }
  
  saving.value = true
  try {
    if (isEdit.value && editId.value) {
      await api.updateProject(editId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await api.createProject(form.value)
      ElMessage.success('创建成功')
    }
    fetchProjects()
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.response?.data || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    saving.value = false
  }
}

const viewEnvs = (row: Project) => {
  router.push(`/projects/${row.id}/environments`)
}

onMounted(() => {
  fetchProjects()
})
</script>

<style scoped>
.project-list {
  padding: 8px;
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
.project-grid {
  margin-top: 16px;
}
.project-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}
:deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0 !important;
}
.card-header {
  padding: 20px 20px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.card-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 70%;
}
.id-tag {
  background-color: rgba(148, 163, 184, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.2);
  color: var(--text-secondary);
  font-family: var(--mono);
}
.card-body {
  padding: 20px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
}
.info-row .el-icon {
  color: var(--accent-blue);
  font-size: 16px;
}
.truncate {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.card-footer {
  padding: 12px 20px;
  background-color: rgba(0, 0, 0, 0.2);
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid rgba(255, 255, 255, 0.03);
}
</style>
