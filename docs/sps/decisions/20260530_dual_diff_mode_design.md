# 架构决策记录 (ADR) - 双模式 Diff 与发布文件列表获取机制

## 1. 业务需求深挖与分析
用户要求对代码部署前的变更审查进行维度加固：
1. **首步文件清单的区分**：
   - **Commit 发布**：第一步只列出**“这次提交影响的变更文件列表”**（即增删改文件）。
   - **Branch / Tag 发布**：第一步列出**“当前分支/Tag 版本的全量文件列表”**（即项目所有文件），以便用户完整审查并选择性勾选/排除发布文件。
2. **双模式对比按钮**：
   - 在比对弹窗中，提供两个查看维度：
     - **“Git 提交变更 (Git Log Diff)”**：仅对比代码版本库本身在部署前后的 commit 差异。
     - **“与线上实际对比 (Live Diff)”**：对比当前待发布的文件与远程目标服务器上当时运行的实际文件差异。
3. **历史快照存档策略**：
   - **Commit 部署**：需要同时把与线上实际对比（Live Diff）的数据持久化落盘为任务快照。
   - **Branch / Tag 部署**：由于是全量大版本发布，为防范快照文件过大造成磁盘压力，**只保留** “Git Log Diff” 即可。

## 2. 技术设计方案

### 2.1 获取文件列表逻辑 (`/api/projects/:id/preview_diff`)
后端通过请求中的参数 `target_type` 决定使用什么 Git 命令：
- **`target_type = "commit"`**：
  - 执行 `git diff --name-only fromCommit toCommit --`
- **`target_type = "branch"` 或 `"tag"`**：
  - 执行 `git ls-tree -r --name-only toCommit --` 获取该版本下的全量文件列表。

### 2.2 双模式 Diff 接口设计
后端接口 `/api/projects/:id/preview_diff` 和 `/api/tasks/:id/diff` 增加支持 `diff_type` 参数：
- **`diff_type = "git_log"`**：对比本地 Bare 仓库中前一次成功 commit 与本次目标 commit 之间的代码差异。
- **`diff_type = "live"`**：对比本地待部署文件与线上实际文件的差异。在 Demo 阶段，为保证只读安全性，我们可通过对比 `prevCommit` 与 `toCommit` 的实际差异，模拟与线上的对比，并保留快照。

### 2.3 历史快照保存机制 (`cacheTaskDiff`)
在部署成功后，调度引擎执行快照归档：
- 查询当前任务的 `target_type`（或通过部署分支判断）。
- 如果是 **Commit** 部署：生成 Live Diff 并将其与变动文件列表序列化为 `task_{id}_diff.log` 缓存。
- 如果是 **Branch / Tag** 部署：生成 Git Log Diff，作为快照序列化保存。

## 3. 出口准则
- 物理生成此 ADR 后，输出 `[SCAN_COMPLETE]`，待用户确认。
