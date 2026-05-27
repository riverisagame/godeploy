# 纳米级执行计划: M6 Frontend WebSocket & UI Overhaul

## 阶段目标
将部署控制台的前端短轮询机制重构为 WebSocket，彻底消除 HTTP 轮询的额外开销。同时基于 Vue 的原生 Transition API 为关键数据变化添加高端、平滑的动效。

## 子任务拆解与原子动作

### 步骤 1: 后端引入 WebSocket 依赖与鉴权结构
- **文件**: `go.mod` & `go.sum`
- **动作**: 执行 `go get github.com/gorilla/websocket`。
- **文件**: `api.go`
- **动作**:
  - 定义 `var upgrader = websocket.Upgrader{CheckOrigin: ...}`。
  - 在 `SetupRoutesWithExecutor` 中新增非鉴权组路由 `r.GET("/api/ws/tasks/:id/log", handler.HandleWSLog)`（因为鉴权将放在 WS 建立后的首个 Payload 中）。

### 步骤 2: 后端实现 WebSocket 日志流式推送
- **文件**: `api.go`
- **动作**:
  - 实现 `HandleWSLog(c *gin.Context)`。
  - **鉴权阶段**: 连接后，启动 3 秒超时等待。读取客户端第一条 TextMessage 格式为 `{"type": "auth", "token": "xxx"}`。手动调用 `ValidateToken(token, secret)` 进行鉴权。失败则立即断开 `conn.Close()`。
  - **推送阶段**: 鉴权通过后，进入无限循环。根据 TaskID 查出状态与日志路径。利用类似 `os.Stat` 检查文件大小变化，发生变化则读取增量或全量，通过 `conn.WriteMessage` 发送。若任务状态不再是 `pending` 或 `deploying`，发送最终日志并关闭连接。为了避免 CPU 空转，循环中休眠 1 秒（相当于把原前端的轮询移至后端的轻量级协程中，大幅减少网络头开销）。

### 步骤 3: 前端网络层改造 (WebSocket)
- **文件**: `web/src/views/Dashboard.vue`
- **动作**:
  - 新增组件级变量 `let ws: WebSocket | null = null`。
  - 重构 `showLog(task)`：取消 `setInterval`。构建 `new WebSocket(...)`。
  - 在 `ws.onopen` 中，发送包含 `localStorage.getItem('token')` 的鉴权 JSON。
  - 在 `ws.onmessage` 中，接收后端推送的文本并赋值给 `logText.value`，并调用 `scrollToBottom()`。
  - 在 `ws.onerror` 和 `ws.onclose` 中，触发降级（Fallback），降级逻辑：如果 WS 意外断开，且任务仍在部署中，回退到原有的 `setTimeout` 轮询 HTTP `fetchTaskLog`。
  - 在 `closeLog()` 中，如果 `ws` 不为空，调用 `ws.close()`。

### 步骤 4: 前端视觉动效注入 (Premium Animations)
- **文件**: `web/src/views/Dashboard.vue` 及配套 `style.css`
- **动作**:
  - 将 `<div v-for="proj in projects">` (项目列表) 替换或包裹在 `<TransitionGroup name="list" tag="div">` 中。
  - 将历史记录表格的行（如果有原生方法），或者在历史任务数据更新时，利用 CSS `.list-enter-active, .list-leave-active` 添加优雅的 Opacity 和 Transform Y 轴滑入效果。
  - 针对 `terminal-container`（控制台黑框）增加入场动效，从下方微微浮现（`transform: translateY(10px); opacity: 0;` 到 `1`）。

### 步骤 5: 验证与无损性对冲
- **验证手段**:
  - `TestEngine_MultiNodeDeploy` 不受影响（纯 API 扩展不触碰核心执行层）。
  - 执行 `npm run build` 测试前端编译情况。
  - 模拟断网，验证 WebSocket 降级到短轮询的机制是否能正确缝合。
