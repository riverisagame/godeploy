<template>
  <el-container class="layout-container">
    <el-aside width="240px" class="aside">
      <div class="logo">PDeploy</div>
      <el-menu :default-active="activeMenu" router background-color="transparent" text-color="#94A3B8" active-text-color="#F8FAFC">
        <el-menu-item index="/projects" class="cursor-pointer">
          <el-icon><Menu /></el-icon>
          <span>项目管理</span>
        </el-menu-item>
        <el-menu-item index="/servers" class="cursor-pointer" v-if="isAdmin()">
          <el-icon><Platform /></el-icon>
          <span>服务器管理</span>
        </el-menu-item>
        <el-menu-item index="/users" class="cursor-pointer" v-if="isAdmin()">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    
    <el-container class="main-wrapper">
      <el-header class="header">
        <div class="header-right">
          <el-dropdown>
            <span class="el-dropdown-link cursor-pointer">
              {{ username }}
              <el-tag v-if="userRole" size="small" :type="userRole === 'admin' ? 'danger' : 'info'" effect="dark" style="margin-left: 8px; margin-right: 4px;">{{ userRole === 'admin' ? '管理员' : '开发者' }}</el-tag>
              <el-icon><arrow-down /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleLogout">退出</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      
      <el-main class="main-content">
        <router-view></router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Menu, Platform, ArrowDown, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getUserInfo, isAdmin } from '../utils/auth'

const route = useRoute()
const router = useRouter()

const username = ref('User')
const userRole = ref('')

onMounted(() => {
  const user = getUserInfo()
  if (user) {
    username.value = user.username
    userRole.value = user.role
  }
})

const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/projects')) return '/projects'
  return path
})

const handleLogout = () => {
  localStorage.removeItem('token')
  ElMessage.success('已退出登录')
  router.push('/')
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  background-color: var(--bg-dark);
}
.aside {
  background-color: var(--bg-card);
  border-right: 1px solid var(--border-color);
  z-index: 10;
  display: flex;
  flex-direction: column;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  font-size: 24px;
  font-weight: 700;
  font-family: var(--mono);
  letter-spacing: 1px;
  border-bottom: 1px solid var(--border-color);
  background: linear-gradient(90deg, #3B82F6 0%, #60A5FA 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 0 15px rgba(59, 130, 246, 0.4);
}
.main-wrapper {
  background-color: var(--bg-dark);
}
.header {
  background-color: rgba(17, 24, 39, 0.6);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 24px;
  height: 60px;
}
.el-menu {
  background-color: transparent !important;
  padding-top: 12px;
}
.el-menu-item {
  margin: 4px 12px;
  border-radius: 8px;
  transition: all 0.2s ease;
  height: 48px;
  line-height: 48px;
}
.el-menu-item:hover {
  background-color: rgba(59, 130, 246, 0.1) !important;
  color: #F8FAFC !important;
}
.el-menu-item.is-active {
  background-color: rgba(59, 130, 246, 0.15) !important;
  color: var(--accent-blue) !important;
  position: relative;
}
.el-menu-item.is-active::before {
  content: '';
  position: absolute;
  left: -12px;
  top: 50%;
  transform: translateY(-50%);
  height: 24px;
  width: 4px;
  background-color: var(--accent-blue);
  border-radius: 0 4px 4px 0;
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.6);
}
.el-dropdown-link {
  display: flex;
  align-items: center;
  color: var(--text-primary);
  font-weight: 500;
  padding: 6px 12px;
  border-radius: 6px;
  transition: background-color 0.2s ease;
}
.el-dropdown-link:hover {
  background-color: rgba(255, 255, 255, 0.05);
}
.main-content {
  overflow-y: auto;
  overflow-x: hidden;
}
</style>
