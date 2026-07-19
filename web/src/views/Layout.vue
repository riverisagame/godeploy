<template>
  <el-container class="layout-container">
    <el-aside width="200px" class="aside">
      <div class="logo">PDeploy</div>
      <el-menu default-active="/projects" router background-color="transparent" text-color="#94A3B8" active-text-color="#F1F5F9">
        <el-menu-item index="/projects">
          <el-icon><Menu /></el-icon>
          <span>项目管理</span>
        </el-menu-item>
        <el-menu-item index="/servers">
          <el-icon><Platform /></el-icon>
          <span>服务器管理</span>
        </el-menu-item>
        <el-menu-item index="/deployments">
          <el-icon><Promotion /></el-icon>
          <span>上线发布</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    
    <el-container>
      <el-header class="header">
        <div class="header-right">
          <el-dropdown>
            <span class="el-dropdown-link">
              Admin <el-icon><arrow-down /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>退出</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      
      <el-main>
        <router-view></router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Menu, Promotion, Platform, ArrowDown } from '@element-plus/icons-vue'
</script>

<style scoped>
.layout-container {
  height: 100vh;
  background-color: var(--bg-dark);
}
.aside {
  background-color: #0B1121;
  border-right: 1px solid var(--border-color);
  box-shadow: 2px 0 10px rgba(0,0,0,0.5);
  z-index: 10;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: var(--text-primary);
  font-size: 24px;
  font-weight: 700;
  font-family: var(--mono);
  letter-spacing: 1px;
  border-bottom: 1px solid var(--border-color);
  background: linear-gradient(90deg, #3B82F6 0%, #60A5FA 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 0 10px rgba(59, 130, 246, 0.3);
}
.header {
  background-color: rgba(15, 23, 42, 0.8);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 20px;
}
.el-menu {
  background-color: transparent !important;
}
.el-menu-item {
  margin: 8px 10px;
  border-radius: 6px;
  transition: all 0.2s ease;
}
.el-menu-item:hover {
  background-color: rgba(59, 130, 246, 0.1) !important;
}
.el-menu-item.is-active {
  background-color: var(--accent-blue) !important;
  color: #fff !important;
  box-shadow: 0 0 12px rgba(59, 130, 246, 0.4);
}
.el-dropdown-link {
  cursor: pointer;
  display: flex;
  align-items: center;
  color: var(--text-primary);
  font-weight: 500;
}
</style>
