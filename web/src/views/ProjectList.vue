<template>
  <div class="project-list">
    <div class="header-actions">
      <h2>项目管理</h2>
      <el-button type="primary" @click="dialogVisible = true">新建项目</el-button>
    </div>

    <el-table :data="projects" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="项目名称" width="180" />
      <el-table-column prop="repo_url" label="Git 仓库" />
      <el-table-column prop="keep_releases" label="保留版本数" width="120" />
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="viewEnvs(scope.row)">环境配置</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create Project Dialog -->
    <el-dialog title="新建项目" v-model="dialogVisible" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="项目名称">
          <el-input v-model="form.name" placeholder="例如: pdeploy-web"></el-input>
        </el-form-item>
        <el-form-item label="Git 仓库">
          <el-input v-model="form.repo_url" placeholder="例如: git@github.com:..."></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="createProject" :loading="creating">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const router = useRouter()

const projects = ref<any[]>([])
const dialogVisible = ref(false)
const creating = ref(false)
const form = ref({
  name: '',
  repo_url: ''
})

const fetchProjects = async () => {
  try {
    const res = await axios.get('/api/projects')
    projects.value = res.data
  } catch (e) {
    ElMessage.error('获取项目失败')
  }
}

const createProject = async () => {
  if (!form.value.name || !form.value.repo_url) {
    ElMessage.warning('请填写项目名称和Git仓库地址')
    return
  }
  
  creating.value = true
  try {
    const res = await axios.post('/api/projects', form.value)
    ElMessage.success('创建成功')
    projects.value.push(res.data)
    dialogVisible.value = false
    form.value.name = ''
    form.value.repo_url = ''
  } catch (e: any) {
    ElMessage.error(e.response?.data || '创建失败')
  } finally {
    creating.value = false
  }
}

const viewEnvs = (row: any) => {
  router.push(`/projects/${row.id}/environments`)
}

onMounted(() => {
  fetchProjects()
})
</script>

<style scoped>
.project-list {
  padding: 24px;
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

:deep(.el-table) {
  background-color: var(--bg-card);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

:deep(.el-table th.el-table__cell) {
  background-color: rgba(15, 23, 42, 0.6);
  color: var(--text-primary);
  font-weight: 600;
  border-bottom: 1px solid var(--border-color);
}

:deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-card);
}

:deep(.el-table--enable-row-hover .el-table__body tr:hover > td.el-table__cell) {
  background-color: rgba(59, 130, 246, 0.05);
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

:deep(.el-input__wrapper) {
  background-color: var(--bg-dark);
  box-shadow: 0 0 0 1px var(--border-color) inset;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--accent-blue) inset !important;
}
</style>
