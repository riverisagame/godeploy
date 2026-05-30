# Plan: Fix Branch Deployment Diff Freeze and Prevent Frontend Hangs

## Proposed Changes

### 1. Backend: Optimize Git Repo Lookup and Prevent Slow Walk Searches
- **文件**: [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- **修改点**:
  1. 在 `findGitRepo` 中，增加对 `commit` 参数的格式校验。如果 `commit` 长度不为 40 或者是分支名、空字符串，则直接返回空字符串，不进行任何 Walk。
  2. 在 `HandleGetTaskDiff` 获取 `gitRepoPath` 的地方，如果 `buildPath/.git` 不存在，优先尝试使用本地常驻 Bare 缓存目录 `getCacheDir(projectID)`。只有在 Bare 缓存目录也不存在时，才作为最后的降级手段走 `findGitRepo`。
- **文件**: [godeployer/engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go)
- **修改点**:
  1. 在 `cacheTaskDiff` 中获取 `gitRepoPath` 的地方，与上面做同样的优化：优先使用 `getCacheDir(projectID)`，避免降级到 `findGitRepo` 进行全局扫盘。

### 2. Backend: Support env_id and Automatic baseCommit Deduction in Preview Diff
- **文件**: [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- **修改点**:
  1. 在 `HandleGetProjectPreviewDiff` 中，读取 Query 中的 `env_id` 参数。
  2. 如果 `fromCommit` 为空，且 `env_id` 不为空，则自动查询该项目在此环境上最近一次成功部署的 `commit_id` 作为 `fromCommit` 的值，从而支持真正的 Live Diff 对比。

### 3. Backend: Limit Maximum File Tree Size to Prevent Browser DOM Hangs
- **文件**: [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- **修改点**:
  1. 在 `HandleGetProjectPreviewDiff` 中，如果 `targetType != "commit"`，获取的文件列表如果极多，限制返回的最大文件数（例如最多返回 2000 个文件），并在列表末尾追加一个虚拟的截断提示节点，防止前端 Element Plus Tree 渲染过载导致假死。

## Verification Plan

### Automated Tests
- 运行 `go test -v -race ./...` 确保所有后端单元测试 100% 通过且无 Data Race。

### Manual Verification
- 在 WSL 环境下，使用分支部署（如 `master`），然后点击“预览 Diff”，验证能瞬间打开并查看，无任何卡顿。
- 部署成功后，在部署历史列表中点击 Diff 按钮，验证其可以极速响应，文件列表和 Diff 页面完全流畅。
