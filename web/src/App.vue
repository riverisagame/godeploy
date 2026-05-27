<template>
  <div class="app-container">
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => {
  // 全局注入 html 暗色类名，启用 Element Plus 的暗色系统
  document.documentElement.classList.add('dark')
})
</script>

<style>
/* 全局样式覆盖，保证暗色沉浸感 */
html, body, #app, .app-container {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  background-color: #121212;
  color: #e0e0e0;
  font-family: 'Inter', 'Roboto', 'Outfit', sans-serif;
  overflow: hidden;
}
</style>
