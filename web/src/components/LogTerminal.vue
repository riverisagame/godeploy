<template>
  <el-dialog v-model="localVisible" title="构建与同步日志" width="80%" @close="handleClose" :close-on-click-modal="false" destroy-on-close>
    <div class="terminal-container">
      <div class="terminal-header">
        <span class="dot red"></span><span class="dot yellow"></span><span class="dot green"></span>
        <span class="term-title">Deploy Terminal - Task #{{ task?.id }}</span>
      </div>
      <pre ref="termBody" class="terminal-body">{{ logText }}</pre>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onUnmounted } from 'vue'
import axios from 'axios'
import { createWSConnection, buildWSUrl } from '../utils/deploy'

const props = defineProps<{ visible: boolean; task: any }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'status-changed', status: string): void }>()

const localVisible = ref(props.visible)
watch(() => props.visible, (v) => { localVisible.value = v; if (v) startLog() })

const logText = ref('')
const termBody = ref<HTMLElement | null>(null)
let logTimer: ReturnType<typeof setInterval> | null = null
let wsConnection: { close: () => void } | null = null

function scrollToBottom() {
  nextTick(() => { if (termBody.value) termBody.value.scrollTop = termBody.value.scrollHeight })
}

function startLog() {
  if (!props.task) return
  logText.value = '正在连接部署服务，拉取日志...'
  const task = props.task

  if (task.status === 'pending' || task.status === 'deploying') {
    const token = localStorage.getItem('token') || ''
    const host = window.location.host
    const protocol = window.location.protocol
    wsConnection = createWSConnection({
      url: buildWSUrl(protocol, host, task.id),
      token,
      onMessage: (data) => { logText.value += data; scrollToBottom() },
    })
    logTimer = setInterval(() => {
      axios.get(`/api/tasks/${task.id}`).then(r => {
        emit('status-changed', r.data.status)
        if (r.data.status !== 'pending' && r.data.status !== 'deploying') {
          cleanup()
        }
      }).catch(() => {})
    }, 3000)
  } else {
    axios.get(`/api/tasks/${task.id}/log`).then(r => {
      logText.value = r.data.log || '暂无日志输出...'
      scrollToBottom()
    }).catch(() => { logText.value = '正在等待日志文件生成...' })
  }
}

function cleanup() {
  if (logTimer) { clearInterval(logTimer); logTimer = null }
  if (wsConnection) { wsConnection.close(); wsConnection = null }
}

function handleClose() {
  cleanup()
  emit('close')
}

onUnmounted(cleanup)
</script>

<style scoped>
.terminal-container {
  background-color: #0b0e14;
  border-radius: 8px;
  border: 1px solid #1a2230;
  overflow: hidden;
}
.terminal-header {
  display: flex; align-items: center; height: 36px;
  background-color: #141a24; padding: 0 12px; gap: 6px;
}
.dot { width: 12px; height: 12px; border-radius: 50%; }
.dot.red { background-color: #ff5f56; }
.dot.yellow { background-color: #ffbd2e; }
.dot.green { background-color: #27c93f; }
.term-title { margin-left: 10px; font-size: 12px; color: #8a99ad; font-family: monospace; }
.terminal-body {
  padding: 16px; margin: 0; height: 380px; overflow-y: auto;
  color: #39ff14; background-color: #0b0e14;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px; line-height: 1.6;
}
</style>
