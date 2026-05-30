# 纳米级执行计划 - 修正双模式 Diff 归档及前端比对按钮

## 1. 拟修改文件清单

### [MODIFY] [godeployer/engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
- 修改 `cacheTaskDiff` 函数，将全量上线 (branch/tag) 和增量上线 (commit) 的快照获取与持久化逻辑对齐最新决策。

### [MODIFY] [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- 优化 `HandleGetTaskDiff`：在获取历史全量部署任务的 live diff 时，若快照中无 live diff，且本地已被 prune 清理导致无法 git diff，则返回友好说明：“该任务为全量部署任务，根据策略未归档与线上对比快照，请切换为本地变更对比。”

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 升级部署前预览弹窗和历史部署 Diff 弹窗的布局。在单文件 Diff 展示区顶部增加 `diff_type` 单选按钮组（“本地变更” / “与线上对比”）。
- 对历史弹窗，若 `task.target_type != 'commit'`，则禁用（disabled）“与线上对比”单选按钮，防止无快照报错。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 修改 `godeployer/engine.go` 中的快照逻辑
- **修改位置**：`godeployer/engine.go` 中的 `cacheTaskDiff`。
- **改动详情**：
  ```go
  // @Ref: docs/sps/plans/20260530_dual_diff_persistence_plan.md | @Date: 2026-05-30
  var liveDiffStr, gitLogDiffStr string
  if targetType == "commit" {
      // 增量上线：两者都要保留
      liveDiffStr, filesStr = generateTaskDiff(prevCommit, commitID, gitRepoPath, 60*time.Second)
      gitLogDiffStr, _ = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
  } else {
      // 全量上线：仅保留 git log diff，不保留 live diff 差异
      gitLogDiffStr, filesStr = generateTaskDiff(commitID+"^", commitID, gitRepoPath, 60*time.Second)
      liveDiffStr = ""
  }
  ```
  在写入 `cacheMap` 时：
  ```go
  cacheMap := map[string]string{
      "files":        filesStr,
      "diff":         liveDiffStr, // 全量时为空
      "git_log_diff": gitLogDiffStr,
  }
  ```

### 任务 2.2: 修改 `godeployer/api.go` 中的降级展示逻辑
- **修改位置**：`godeployer/api.go` 中的 `HandleGetTaskDiff`。
- **改动详情**：
  在单文件 diff 获取降级分支中，若 `diffType == "live"` 且 `targetFullDiff` 为空时，若 `cacheObj.GitLogDiff != ""`，说明是全量快照：
  ```go
  if diffType == "live" && targetFullDiff == "" {
      diffText = "提示：全量部署任务，未归档与线上对比快照。请在上方切换为「本地变更(Git Log Diff)」查看文件修改。"
      err = nil
  }
  ```

### 任务 2.3: 升级前端 `Dashboard.vue` 比对逻辑
- **修改位置**：`web/src/views/Dashboard.vue` 的弹窗及单文件加载逻辑。
- **改动详情**：
  1. 新增状态变量 `currentDiffType = ref('live')`。
  2. 在加载单文件 preview diff 接口 `/api/projects/:id/preview_diff` 和 历史 diff 接口 `/api/tasks/:id/diff` 时，将参数 `diff_type` 设置为 `currentDiffType.value`。
  3. 历史任务比对弹窗初始化时，如果任务的 `target_type !== 'commit'`，自动将 `currentDiffType.value` 切换为 `'git_log'`。
  4. 渲染单选按钮组件：
     ```html
     <el-radio-group v-model="currentDiffType" size="small" @change="handleDiffTypeChange">
       <el-radio-button label="live">与线上对比 (Live Diff)</el-radio-button>
       <el-radio-button label="git_log" :disabled="isHistoryMode && activeTask.target_type !== 'commit'">本地变更 (Git Log Diff)</el-radio-button>
     </el-radio-group>
     ```

---

## 3. 验证方案
- 运行后端的 `go test ./...`。
- 构建前端并整体联调。
