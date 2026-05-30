# 纳米级执行计划 - 实现双模式 Diff 与发布文件列表区分

## 1. 拟修改文件清单
### [MODIFY] [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- 增加 `isCommitHash` 自动推断辅助函数。
- 修改 `HandleGetProjectPreviewDiff` 接口，根据 `target_type` 返回变更文件列表（Commit）或全量文件列表（Branch/Tag），并接收 `diff_type` 参数，在 `diff_type == "git_log"` 时使用 `toCommit^` 到 `toCommit` 计算单文件差异。
- 修改 `HandleGetTaskDiff` 接口，同样接收 `diff_type` 和 `file` 参数，支持双模读取。

### [MODIFY] [godeployer/engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
- 修改 `cacheTaskDiff`，根据发布类型选择持久化 Git Log Diff 快照（Branch/Tag）还是 Live Diff 快照（Commit）。
- 并在协程内部加上 `recover()` 防护，防止因 DB 提前 Close 导致测试进程 Crash 退出。

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 在全屏 Diff 弹窗的右侧上方增加两个单选按钮或 Tab 选项：“本地变更 (Git Log Diff)”与“与线上对比 (Live Diff)”。
- 请求预览与懒加载接口时，携带 `target_type` 以及当前选中的 `diff_type`。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 后端 `api.go` 的双模获取与安全加固
- **目标路径**：`godeployer/api.go`
- **改动详情**：
  - 新增 `isCommitHash` 函数。
  - 在 `HandleGetProjectPreviewDiff` 中，根据 `isCommitHash(toCommit)` 的结果：
    - 如果是 commit：执行 `git diff --name-only fromCommit toCommit --`
    - 如果是 branch/tag：执行 `git ls-tree -r --name-only toCommit --`
  - 处理单文件 Diff 获取：
    ```go
    diffType := c.DefaultQuery("diff_type", "live")
    baseCommit := fromCommit
    if diffType == "git_log" {
        baseCommit = toCommit + "^"
    }
    ```
    传递给 `GetDiffForFile` 时，使用对应的 `baseCommit` 进行对比。

### 任务 2.2: 后端 `engine.go` 的快照过滤与协程安全
- **目标路径**：`godeployer/engine.go`
- **改动详情**：
  - 在 `cacheTaskDiff` 执行体最开始加上 `defer func() { recover() }()`。
  - 判断 `commitID` 是不是 commit hash：
    - 如果不是（即 branch/tag 全量发布），在生成缓存快照时，执行 `generateTaskDiff(commitID+"^", commitID, gitRepoPath)`。
    - 如果是 commit 部署，则执行 `generateTaskDiff(prevCommit, commitID, gitRepoPath)`。

### 任务 2.3: 前端 `Dashboard.vue` 的双按钮交互
- **目标路径**：`web/src/views/Dashboard.vue`
- **改动详情**：
  - 增加 `diffType` 响应式变量（默认为 `'live'`）。
  - 在右侧 Diff 展示区域的上方增加 Tab 切换。
  - 在 `loadSingleFileDiff` 中发送请求时，加上 `diff_type: diffType.value` 参数。

---

## 3. 验证方案
- 运行后端的 `go test ./...` 确保完全 PASS。
- 运行前端 Vitest 单元测试，确保无渲染卡死。
