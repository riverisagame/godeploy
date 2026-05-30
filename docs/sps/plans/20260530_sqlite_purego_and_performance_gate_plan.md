# Plan: SQLite Pure Go Migration and System-Wide Performance Gate

## Proposed Changes

### 1. Backend: SQLite 100% Pure Go Driver Migration
- **文件**: [go.mod](file:///d:/claudeprj/deploy/go.mod)
- **修改点**:
  1. 移除 `github.com/mattn/go-sqlite3`。
  2. 引入 `modernc.org/sqlite`。
- **文件**: [godeployer/db.go](file:///d:/claudeprj/deploy/godeployer/db.go)
- **修改点**:
  1. 替换包级导入为 `_ "modernc.org/sqlite"`。
  2. 在 `InitDB` 函数中，将 `sql.Open("sqlite3", dsn)` 修改为 `sql.Open("sqlite", dsn)`。

### 2. Backend: System-Wide Process Throttling (Semaphore)
- **文件**: [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- **修改点**:
  1. 声明包级全局变量 `diffSemaphore = make(chan struct{}, 5)` 用以限制并发 Git Diff 命令。
  2. 在 `HandleGetProjectPreviewDiff` 和 `HandleGetTaskDiff` 的入口，使用 `select` 尝试获取该信号量。若在 3 秒内无法获取，则快速返回 `http.StatusTooManyRequests`，保护系统不被大流量卡死。
- **文件**: [godeployer/engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
- **修改点**:
  1. 声明包级全局变量 `deploySemaphore = make(chan struct{}, 3)` 用以限制后台并发部署的任务数。
  2. 在 `SubmitJob` 或后台并发执行部署的入口，若 `deploySemaphore` 已满则将任务排队或拒绝。

## Verification Plan

### Automated Tests
- 运行 `CGO_ENABLED=0 go test -v ./godeployer` 确保在**关闭 CGO** 的状态下，所有 30+ 后端测试依然完美通过。

### Manual Verification
- 验证生产构建在 `CGO_ENABLED=0` 下成功。
- 在浏览器中模拟多次快速点击部署和获取 diff 按钮，验证界面无假死且排队/限流提示生效。
