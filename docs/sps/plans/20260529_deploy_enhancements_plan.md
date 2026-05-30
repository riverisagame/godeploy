# UI-003 / ARCH-010: 动态过滤、双视角快照与部署备注高级特性纳米计划

## 1. 计划目标
- 为 `deploy_tasks` 数据表安全平滑地添加 `description` 和 `extra_exclude` 字段，绝对无损历史数据。
- 升级部署预览接口，提供本次上线的变更文件列表。
- 后端部署引擎支持解析 `extra_exclude` 并组装到 `rsync` 排除参数中。
- 升级对比快照，从 `.log` 升级为具有 `{"files": "...", "diff": "..."}` 结构的高内聚 `.json` 快照，平滑兼容历史快照。
- 前端支持变更文件可视化 `el-tree` 勾选过滤、填写备注，以及历史双视窗 Tab 切换。

---

## 2. 纳米级修改步骤

### [步骤一] 修改数据库层 `godeployer/db.go`
- 在 `InitDB` 中的 `RepairStalledTasks` 附近（或其后），加一段动态 schema 迁移代码：
  1. 通过 `PRAGMA table_info(deploy_tasks)` 查询现有字段。
  2. 若 `description` 不存在，执行 `ALTER TABLE deploy_tasks ADD COLUMN description TEXT DEFAULT ""`。
  3. 若 `extra_exclude` 不存在，执行 `ALTER TABLE deploy_tasks ADD COLUMN extra_exclude TEXT DEFAULT ""`。
- 在 `deploy_tasks` 的结构体中（若存在）追加这两个字段。

### [步骤二] 修改后端 `godeployer/api.go` 的结构体与初始化
- 修改 `Task` 实体结构体，新增 `Description` 和 `ExtraExclude` 字段（JSON 字段名匹配）。
- 修改 `HandleCreateTask`：
  1. 接收 POST 的 `description` 和 `extra_exclude`（字符串，多个用逗号隔开，或 json 数组字符串）。
  2. 在 `INSERT INTO deploy_tasks` 时增加这两个字段的值写入。
- 升级 `HandleGetProjectPreviewDiff`：
  1. 接收 `from` 和 `to` commit 参数。如果 `from` 为空，自动在 `deploy_tasks` 查找该项目该环境上一次成功的 `commit_id` 作为 `from`，若无则使用 `to^`（前一个提交）。
  2. 调用 `git diff --name-only <from> <to>` 命令，将返回的扁平相对路径按换行切分为 `files` 字符串数组，与 `diff` 文本一同返回。

### [步骤三] 修改后端引擎层文件同步参数拼接 `godeployer/engine.go` (或 `ssh.go`)
- 定位后端 rsync 推送的方法（可能位于 `RunDeploy` 或 rsync 相关逻辑中）。
- 从数据库读取当前 task 的 `extra_exclude` 字段。
- 将 `extra_exclude` 按照逗号或空格分割，生成 `--exclude='相对路径'` 命令行参数数组，追加到最终的 `rsync` 命令构建中，确保同步过程排除被过滤文件。

### [步骤四] 升级快照缓存逻辑为高内聚 JSON
- 修改 `HandleGetTaskDiff`：
  - 拼装文件路径升级为：`task_<id>_changes.json`。
  - 读取时，若为 JSON 文件，直接解析并返回 `{"files": "...", "diff": "..."}`。
  - 若读取到旧的 `_diff.log` 文本文件，降级解析为 `{"files": "历史快照，暂无列表", "diff": "旧文本内容"}`。
  - 写入时，运行 `git diff --name-status <prev> <current>` 作为 `files` 内容，与 `diff` 文本组装成 JSON 写入 `task_<id>_changes.json`（支持 2MB 大小限额截断与可用空间判断）。

### [步骤五] 修改前端 `web/src/views/Dashboard.vue` 的模板与脚本
- **触发上线 Dialog**：
  - 添加部署备注 `el-input` 输入框。
  - 添加 `el-collapse` 或展开区，显示“本次变更文件过滤勾选树”。
  - 当选中要发布的 Commit 后，触发 `fetchPreviewChanges` 接口。
  - 得到文件列表后，用一段建树算法在前端递归生成嵌套 JSON 树，绑定到带 Checkbox 的 `el-tree`。
  - 在提交部署时，统计未选中的叶子节点路径，作为 `extra_exclude` 传入 POST 参数。
- **历史记录 Table**：
  - 新增一列 `部署备注`。
- **对比 Dialog**：
  - 改用 `el-tabs` 结构，提供 `文件变更列表`（通过 `el-table` 或高亮展现）和 `代码差异对比` 双视窗。

---

## 3. 验证与 TDD 计划
- **RED 阶段**：
  - 物理创建测试文件 `godeployer/api_enhance_test.go`。
  - 编写对 `HandleGetProjectPreviewDiff` 返回 `files` 数组的断言。
  - 编写对创建任务入参 `description` 写入以及 `extra_exclude` 校验的断言。
  - 编写 rsync 排除参数包含 `extra_exclude` 的 mock 验证测试。
  - 执行 `go test`，在无业务代码时看到测试失败。
- **GREEN 阶段**：编写最小实现代码使测试全部通过。
- **手动验收**：在前端操作勾选、查看双 tab 页面。
- **Git 提交推送**：通过 Windows 系统将改动推送至 Github。
