<template>
  <div class="deploy-detail">
    <div class="header-actions">
      <div>
        <h2>部署详情</h2>
        <p class="subtitle">实时查看部署日志及状态 (ID: {{ route.params.id }})</p>
      </div>
      <el-button @click="$router.back()" size="large">
        <el-icon class="el-icon--left"><Back /></el-icon> 返回
      </el-button>
    </div>

    <!-- Terminal Window -->
    <div class="terminal-window">
      <!-- Terminal Header -->
      <div class="terminal-header">
        <div class="mac-buttons">
          <div class="mac-btn close"></div>
          <div class="mac-btn minimize"></div>
          <div class="mac-btn maximize"></div>
        </div>
        <span class="terminal-title">Deployment Console</span>
      </div>
      
      <!-- Logs Area -->
      <div class="terminal-body" ref="logContainer">
        <div v-for="(log, idx) in logs" :key="idx" class="log-line">
          <span class="log-number">{{ idx + 1 }}</span>
          <span :class="getLogClass(log)">{{ log }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { Back } from '@element-plus/icons-vue'

const route = useRoute()
const logs = ref<string[]>([])
const logContainer = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

const getLogClass = (log: string) => {
  if (log.includes('ERROR') || log.includes('FAILED')) return 'log-error'
  if (log.includes('>>>')) return 'log-system'
  if (log.includes('SUCCESS') || log.includes('Done')) return 'log-success'
  return 'log-normal'
}

const connectSSE = () => {
  const deployID = route.params.id
  if (!deployID) return

  eventSource = new EventSource(`/api/deployments/${deployID}/logs`)
  
  eventSource.onmessage = (event) => {
    if (event.data === '[EOF]') {
      if (eventSource) eventSource.close()
      logs.value.push('>>> Connection closed. Deployment finished.')
      scrollToBottom()
      return
    }
    logs.value.push(event.data)
    scrollToBottom()
  }

  eventSource.onerror = (error) => {
    console.error('SSE Error:', error)
    logs.value.push('>>> Error connecting to log stream.')
    if (eventSource) eventSource.close()
    scrollToBottom()
  }
}

onMounted(() => {
  connectSSE()
})

onUnmounted(() => {
  if (eventSource) {
    eventSource.close()
  }
})
</script>

<style scoped>
.deploy-detail {
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

.terminal-window {
  flex: 1;
  background-color: #0d1117; /* GitHub Dark Dimmed Background */
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
  font-family: var(--mono);
}

.terminal-header {
  background-color: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid var(--border-color);
  padding: 12px 16px;
  display: flex;
  align-items: center;
}

.mac-buttons {
  display: flex;
  gap: 8px;
}

.mac-btn {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}
.mac-btn.close { background-color: #ff5f56; }
.mac-btn.minimize { background-color: #ffbd2e; }
.mac-btn.maximize { background-color: #27c93f; }

.terminal-title {
  margin-left: 16px;
  color: var(--text-secondary);
  font-size: 13px;
  font-family: var(--font-family); /* Use sans-serif for title */
}

.terminal-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background-color: #0d1117;
}

.log-line {
  display: flex;
  margin-bottom: 4px;
  font-size: 13px;
  line-height: 1.5;
}

.log-number {
  color: rgba(255, 255, 255, 0.2);
  margin-right: 16px;
  user-select: none;
  min-width: 24px;
  text-align: right;
}

.log-normal {
  color: #c9d1d9; /* GitHub Dark Text */
  word-break: break-all;
}

.log-error {
  color: #ff7b72; /* GitHub Dark Red */
}

.log-system {
  color: #79c0ff; /* GitHub Dark Blue */
  font-weight: 600;
}

.log-success {
  color: #3fb950; /* GitHub Dark Green */
}
</style>
