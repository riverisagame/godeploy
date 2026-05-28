# 纳米级执行计划: 任务差异对比与高级提交过滤 UI 升级

## 阶段目标
响应用户的“Diff 视窗更大化”与“精准多维提交记录筛选”需求，重构前台 Diff 展示弹窗为全屏模式，并在前后端打通针对特定分支/Tag、特定文件、特定提交人的高级 `git log` 查询能力。

## 子任务拆解与原子动作

### 步骤 1: 后端 `GetCommits` API 参数扩充
- **文件**: `d:\claudeprj\deploy\godeployer\git.go`
- **动作**:
  - 修改 `GetCommits` 函数签名，增加 `ref string` 参数。
  - 在构建 `git log` 命令参数时，若 `ref != ""`，将 `ref` 追加到 `args` 中（如 `args = append(args, ref)`），并移除原本硬编码的 `--all`。
  - 若 `ref == ""`，则保留 `--all` 查询全库记录。
- **文件**: `d:\claudeprj\deploy\godeployer\api.go`
- **动作**:
  - 在 `HandleGetProjectCommits` 中，解析前端传入的查询参数 `ref := c.Query("ref")`。
  - 将 `ref` 传入 `GetCommits` 函数。

### 步骤 2: 前端 Diff UI 空间扩展
- **文件**: `d:\claudeprj\deploy\web\src\views\Dashboard.vue`
- **动作**:
  - 将 `diffVisible` 对应的 `<el-dialog>` 增加 `fullscreen` 属性（或改为 `width="100%"` 和 `top="0"`），取消 `width="90%"`。
  - 修改内置的 `<div class="diff-container">` 内联样式：从 `height: 70vh;` 改为 `height: calc(100vh - 120px);`，实现最大化可视区域。
  - 确认单选按钮（`side-by-side` 与 `line-by-line`）的位置，确保全屏下顶部栏依然美观。

### 步骤 3: 前端提交记录的高级过滤与交互增强
- **文件**: `d:\claudeprj\deploy\web\src\views\Dashboard.vue`
- **动作**:
  - 在 `commitFilters` 状态对象中增加 `ref: ''` 字段。
  - 在 `targetType === 'commit'` 的筛选面板区域中，增设一个“选择分支/Tag”下拉框，与关键字、提交人、文件路径并排显示，供用户将搜索范围收缩至特定分支。
  - 修改 `fetchCommits` 方法，发起请求时带上 `&ref=${commitFilters.ref}` 参数。

## 出口准则
- 后端能正确解析带有分支引用的查询，并返回正确的过滤结果。
- 前端 Diff 弹窗全屏覆盖，可视区域增大一倍。
- 生成的组件不破坏现有的权限控制与深色主题样式。
