# GoDeployer UI 简化和稳定性重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标:** 将 GoDeployer 前端从单体巨文件 + 数据流混乱状态，重构为组件化、状态清晰、部署流程可稳定运行的系统。

**策略:** 先修数据流（确保后端能正常跑通），再拆组件（不影响运行时），最后统一主题和测试。

**技术栈:** Vue 3 + Composition API + TypeScript, Element Plus, Axios, WebSocket

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `web/src/utils/api.ts` | **新建** — Axios 实例、全局拦截器（401/500）、统一错误提示 |
| `web/src/utils/deploy.ts` | **修改** — 增加 WebSocket 心跳、状态机 helper |
| `web/src/components/ProjectSidebar.vue` | **新建** — 左侧项目列表 + 环境选择 |
| `web/src/components/DeployForm.vue` | **新建** — 版本选择 + 备注 + 部署/预览按钮 |
| `web/src/components/DeployHistoryTable.vue` | **新建** — 历史表格 + 操作按钮 |
| `web/src/components/LogTerminal.vue` | **新建** — 日志弹窗 + WS 管理 |
| `web/src/components/DiffDialog.vue` | **新建** — Diff 对比弹窗 + 只读文件树 |
| `web/src/components/UserSettingsDialog.vue` | **新建** — 账号配置弹窗 |
| `web/src/views/Dashboard.vue` | **重构** — 精简为组合容器 |
| `web/src/views/UserManagement.vue` | **修改** — 暗色主题统一 |
| `web/src/main.ts` | **修改** — 注册 api 实例 |
| `web/src/router.ts` | **不改** |

---

### Task 1: 创建 Axios 实例 + 全局拦截器

**Files:**
- Create: `web/src/utils/api.ts`
- Test: `web/src/__tests__/api.test.ts`

- [ ] **Step 1: 写 api.ts**

```typescript
import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status
    const data = error.response?.data
    if (status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      localStorage.removeItem('role')
      router.push('/login')
      ElMessage.error('登录已过期，请重新登录')
    } else if (status === 403) {
      ElMessage.error(data?.error || '权限不足')
    } else if (status >= 500) {
      ElMessage.error(data?.error || '服务器内部错误，请稍后重试')
    }
    return Promise.reject(error)
  }
)

export default api
```

- [ ] **Step 2: 写 api.test.ts**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import api from '../utils/api'

describe('api interceptor', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('attaches token from localStorage', async () => {
    localStorage.setItem('token', 'test-token')
    // 使用一个不会实际发出的请求来验证拦截器
    const requestInterceptor = api.interceptors.request.handlers[0]
    const config = await requestInterceptor.fulfilled({ headers: {} } as any)
    expect(config.headers.Authorization).toBe('Bearer test-token')
  })

  it('does not attach token when not present', async () => {
    const requestInterceptor = api.interceptors.request.handlers[0]
    const config = await requestInterceptor.fulfilled({ headers: {} } as any)
    expect(config.headers.Authorization).toBeUndefined()
  })
})
```

- [ ] **Step 3: 跑测试确认通过**

Run: `cd web && npx vitest run src/__tests__/api.test.ts`
Expected: PASS

- [ ] **Step 4: 修改 main.ts 移除旧 axios 全局配置**

将 `main.ts` 中所有 `axios.defaults.headers.common['Authorization']` 相关代码去掉，由 api.ts 接管。

- [ ] **Step 5: Commit**

```bash
git add web/src/utils/api.ts web/src/__tests__/api.test.ts web/src/main.ts
git commit -m "feat: add axios instance with global interceptors"
```

---

### Task 2: 修复核心数据流 — 消除 Mock + 单一状态机

**Files:**
- Modify: `web/src/views/Dashboard.vue`（全部 script 部分）
- Modify: `web/src/utils/deploy.ts`

- [ ] **Step 1: deploy.ts 增加状态机类型和 helper**

在 `web/src/utils/deploy.ts` 末尾添加：

```typescript
export type DeployPhase = 'idle' | 'previewing' | 'confirming' | 'deploying' | 'done' | 'error'

export function createDeployState() {
  return {
    phase: 'idle' as DeployPhase,
    error: '',
    taskId: null as number | null,
  }
}
```

测试: 添加单元测试验证状态转换。

- [ ] **Step 2: Dashboard.vue 消除所有 Mock 数据回退**

找到 `fetchHistory` 的 catch 块：
```
historyTasks.value = [
  { id: 3, release_name: '20260527101500', ... },
  { id: 2, ... },
  { id: 1, ... }
]
```

替换为：
```typescript
catch (error) {
  // 不返回 Mock 数据，只提示错误
  console.error('获取部署历史失败', error)
  historyTasks.value = []
}
```

- [ ] **Step 3: 替换多 flag 状态机**

找到:
```typescript
const isPreDeploying = ref(false)
const pendingDeployEnv = ref<Environment | null>(null)
```

替换为:
```typescript
const deployState = reactive(createDeployState())
```

将所有 `if (isPreDeploying.value)` 改为 `if (deployState.phase === 'confirming')`。
将所有 `isPreDeploying.value = true` 改为 `deployState.phase = 'confirming'`。

- [ ] **Step 4: 移除 watch 自动触发 preview_diff 的副作用**

删除整个 `watch([() => deployForm.branch, () => selectedProject.value?.id, () => activeEnvTab.value], ...)` 块。

将 `previewDeployDiff` 改造为手动触发：调用时才请求接口。

- [ ] **Step 5: 跑前端测试验证修改不影响已有测试**

Run: `cd web && npx vitest run`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/utils/deploy.ts web/src/views/Dashboard.vue
git commit -m "fix: eliminate mock data fallback and replace multi-flag with single state machine"
```

---

### Task 3: WebSocket 稳定性 — 心跳 + 重连

**Files:**
- Modify: `web/src/utils/deploy.ts`

- [ ] **Step 1: deploy.ts 增加 WebSocket 管理函数**

```typescript
export interface WSConfig {
  url: string
  token: string
  onMessage: (data: string) => void
  onStatusChange?: (connected: boolean) => void
}

export function createWSConnection(config: WSConfig): { close: () => void } {
  let ws: WebSocket | null = null
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let retryCount = 0
  const maxRetries = 3
  let closed = false

  function connect() {
    if (closed) return
    ws = new WebSocket(config.url)
    config.onStatusChange?.(true)

    ws.onopen = () => {
      config.onStatusChange?.(true)
      retryCount = 0
      ws?.send(JSON.stringify({ type: 'auth', token: config.token }))
      // 心跳
      pingTimer = setInterval(() => {
        ws?.send(JSON.stringify({ type: 'ping' }))
      }, 30000)
    }

    ws.onmessage = (event) => {
      config.onMessage(event.data)
    }

    ws.onclose = () => {
      config.onStatusChange?.(false)
      if (pingTimer) { clearInterval(pingTimer); pingTimer = null }
      if (!closed && retryCount < maxRetries) {
        retryCount++
        const delays = [1000, 3000, 5000]
        setTimeout(connect, delays[retryCount - 1])
      }
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  connect()

  return {
    close: () => {
      closed = true
      if (pingTimer) clearInterval(pingTimer)
      ws?.close()
      ws = null
    }
  }
}
```

- [ ] **Step 2: 写测试验证重连逻辑**

```typescript
import { describe, it, expect, vi } from 'vitest'
import { createWSConnection } from '../utils/deploy'

describe('createWSConnection', () => {
  it('attempts reconnection on close', async () => {
    const wsMock = { send: vi.fn(), close: vi.fn() }
    vi.spyOn(global, 'WebSocket').mockImplementation(() => wsMock as any)

    const onMessage = vi.fn()
    const conn = createWSConnection({
      url: 'ws://localhost:8080/api/ws/tasks/1/log',
      token: 'test',
      onMessage,
    })

    // Simulate close
    wsMock.onclose?.({} as any)
    await new Promise(r => setTimeout(r, 1500)) // wait for retry delay

    expect(global.WebSocket).toHaveBeenCalledTimes(2)
    conn.close()
  })
})
```

- [ ] **Step 3: Commit**

---

### Task 4: 提取 ProjectSidebar 组件

**Files:**
- Create: `web/src/components/ProjectSidebar.vue`
- Test: `web/src/__tests__/ProjectSidebar.spec.ts`

- [ ] **Step 1: 写 ProjectSidebar.vue**

Props: `projects: Project[]`, `selectedId: string`
Emits: `select-project(project)`

```vue
<template>
  <aside class="sidebar">
    <div class="sidebar-title">部署项目</div>
    <el-scrollbar>
      <div
        v-for="proj in projects"
        :key="proj.id"
        class="project-item"
        :class="{ active: proj.id === selectedId }"
        @click="$emit('select-project', proj)"
      >
        <div class="proj-name">{{ proj.name }}</div>
        <div class="proj-id">{{ proj.id }}</div>
      </div>
      <el-empty v-if="projects.length === 0" description="未加载到项目配置" :image-size="60" />
    </el-scrollbar>
  </aside>
</template>
```

- [ ] **Step 2: 写测试验证选中高亮**

- [ ] **Step 3: 嵌入 Dashboard.vue 替代原侧栏**

- [ ] **Step 4: Commit**

---

### Task 5: 提取 DeployForm 组件

**Files:**
- Create: `web/src/components/DeployForm.vue`
- Test: `web/src/__tests__/DeployForm.spec.ts`

- [ ] **Step 1: 写 DeployForm.vue**

Props: `projectId: string`, `env: Environment`, `refsList`, `deployPhase: DeployPhase`
Emits: `deploy(description)`, `preview-diff`

核心逻辑: 版本选择（分支/Tag/Commit）+ 备注 + 两个按钮。

- [ ] **Step 2: 测试表单校验**

- [ ] **Step 3: 嵌入 Dashboard**

- [ ] **Step 4: Commit**

---

### Task 6: 提取 DeployHistoryTable 组件

**Files:**
- Create: `web/src/components/DeployHistoryTable.vue`
- Test: `web/src/__tests__/DeployHistoryTable.spec.ts`

Props: `tasks: Task[]`
Emits: `rollback(task)`, `show-diff(task)`, `show-log(task)`

纯展示组件，不做任何 API 调用。

- [ ] **Step 1: 写组件**
- [ ] **Step 2: 测试**
- [ ] **Step 3: 嵌入**
- [ ] **Step 4: Commit**

---

### Task 7: 提取 LogTerminal 组件

**Files:**
- Create: `web/src/components/LogTerminal.vue`
- Test: `web/src/__tests__/LogTerminal.spec.ts`

Props: `visible: boolean`, `task: Task | null`
Emits: `close`

内部管理 WebSocket 连接生命周期（使用 Task 3 的 `createWSConnection`）。

- [ ] **Step 1: 写组件**
- [ ] **Step 2: 测试 WebSocket 事件**
- [ ] **Step 3: 嵌入**
- [ ] **Step 4: Commit**

---

### Task 8: 提取 DiffDialog 组件

**Files:**
- Create: `web/src/components/DiffDialog.vue`
- Test: `web/src/__tests__/DiffDialog.spec.ts`

Props: `visible: boolean`, `task: Task | null`, `projectId: string`
Emits: `close`

简化模式：不处理部署确认，只读文件树，单一对比模式。

- [ ] **Step 1: 写组件**
- [ ] **Step 2: 测试**
- [ ] **Step 3: 嵌入**
- [ ] **Step 4: Commit**

---

### Task 9: 提取 UserSettingsDialog + 暗色主题统一

**Files:**
- Create: `web/src/components/UserSettingsDialog.vue`
- Modify: `web/src/views/UserManagement.vue`

- [ ] **Step 1: 提取弹窗组件**

将 Dashboard.vue 中的账号设置部分提取为独立组件，Props/Emits 同其他组件模式。

- [ ] **Step 2: UserManagement.vue 暗色主题化**

修改所有 style scoped，使其背景色/边框色/字体色与 Dashboard 的暗色主题一致（#0d1117 / #161b22 / #30363d 色系）。
去掉白色背景、去掉白底 header。

- [ ] **Step 3: Commit**

---

### Task 10: 集成测试 + E2E 补全

**Files:**
- Modify: `web/src/__tests__/Dashboard.spec.ts`

- [ ] **Step 1: 集成测试 — 子组件通信**

模拟选中项目 → 验证 ProjectSidebar emit → 验证 DeployForm 重新渲染 → 验证历史表格刷新。

- [ ] **Step 2: E2E 核心流程补全**

补充 `web/e2e/` 下核心部署流程用例：
1. 登录 → 跳转 Dashboard
2. 选中项目 → 选中环境 → 切换分支 → 点击预览 Diff
3. 触发部署 → 查看日志

- [ ] **Step 3: 跑完整测试套件**

Run: `cd web && npm run test && npm run test:e2e`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: add integration and e2e tests for core deploy flow"
```
