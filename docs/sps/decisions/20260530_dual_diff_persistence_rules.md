# 架构决策记录 (ADR) - 全量与增量上线的双模式 Diff 归档策略

## 1. 业务背景
用户对代码上线前的 Diff 比对及上线后的快照归档提出了清晰的需求：
- **Tag / Branch 上线（全量上线）**：
  - **上线前**：文件列表展示该版本下的所有全量文件。支持单文件与线上进行对比（Live Diff），也支持本地变更对比（Git Log Diff）。
  - **上线后**：为了节约存储空间并避免无意义的对比，只保留 Git Log Diff 快照即可（即 `commitID^` 到 `commitID` 的 `git diff` 变化）。
- **Commit 上线（增量上线）**：
  - **上线前**：文件列表仅展示该次提交所影响的变更文件。支持 Live Diff 和 Git Log Diff。
  - **上线后**：因为是精确的增量发布，需要完整保留 Live Diff 和 Git Log Diff 两个维度的快照。

## 2. 方案设计

### 2.1 后端快照归档逻辑修正 (`engine.go` -> `cacheTaskDiff`)
在部署成功后触发的异步 `cacheTaskDiff` 中：
- 读取该 Task 的 `target_type`（`commit`, `branch`, `tag`）。
- **如果是增量上线 (`target_type == "commit"`)**：
  1. 生成 `liveDiffStr`（`prevCommit` 到 `commitID` 的 `git diff`）。
  2. 生成 `gitLogDiffStr`（`commitID^` 到 `commitID` 的 `git diff`）。
  3. 将两者和变更文件列表打包存入 `task_{id}_diff.log` 缓存。
- **如果是全量上线 (`target_type == "branch"` 或 `"tag"`)**：
  1. **不生成** `liveDiffStr`。
  2. 生成 `gitLogDiffStr`（`commitID^` 到 `commitID` 的 `git diff`）。
  3. 将 `gitLogDiffStr` 写入快照的 `git_log_diff` 字段，同时也将 `diff` 字段留空或复用该值，防止老版本前端报错。
  4. 将两者和全量文件列表打包存入 `task_{id}_diff.log` 缓存。

### 2.2 后端 API 接口适配 (`api.go` -> `HandleGetTaskDiff`)
- 接口 `/api/tasks/:id/diff` 接收 `diff_type`（`live` 或 `git_log`）和 `file`（获取单文件差异）。
- 当读取 JSON 快照时：
  - 若 `diff_type == "git_log"`，优先返回快照中的 `git_log_diff`；若不存在，降级返回 `diff`。
  - 若 `diff_type == "live"`：
    - 若快照中没有 `live`（即全量上线任务），且仓库在本地已被 prune 清理，则返回友好提示：“全量上线任务，未归档与线上对比快照，请查看本地变更差异。”

### 2.3 前端比对弹窗升级 (`Dashboard.vue`)
- 在预览上线弹窗（预览阶段）和任务详情弹窗（历史阶段）的文件比对区域上方，新增单选按钮组：
  - `本地变更 (Git Log Diff)`
  - `与线上对比 (Live Diff)`
- **状态控制**：
  - 预览阶段：两按钮均可用。
  - 历史阶段：
    - 若任务类型为 `commit`（增量）：两按钮均可用。
    - 若任务类型为 `branch` 或 `tag`（全量）：`与线上对比 (Live Diff)` 按钮禁用（Disabled），并在悬浮时提示“全量上线任务不保留线上对比快照”。
- **按需加载 (Lazy Load)**：
  - 切换按钮时，重新调用加载单文件 diff 接口，带上最新的 `diff_type`。

## 3. 影响评估与并发/安全对冲
- **磁盘占用**：全量上线不生成 Live Diff，可彻底避免大项目全量上线快照文件达数十兆导致服务器磁盘占满的问题。
- **并发安全**：单文件 diff 接口和快照保存逻辑均为只读或独立文件写入，不存在并发写冲突。
