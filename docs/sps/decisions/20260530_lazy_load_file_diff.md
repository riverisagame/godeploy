# ADR: 引入单文件懒加载 Diff 机制与左右分栏布局

## 问题背景
在大项目（如 `crmeb-shop`）中，一次发布可能包含数十个文件，差异文本总体大小可能达到数兆甚至数十兆。一上来就从后端传输全量 Diff，并让前端一次性渲染，极易造成网络延迟、内存溢出和浏览器界面卡死。

## 提议方案：懒加载单文件 Diff 与左右分栏体验
为了提供媲美 IDE 和 GitHub 的高品质体验，我们将 Diff 预览对话框重构为**左右分栏布局**加**单文件懒加载**机制。

### 1. 左右分栏交互
- **左侧侧边栏 (宽 300px)**：展示变更文件列表表格，支持滚动和过滤。
- **右侧工作区 (自适应)**：
  - 默认状态：提示“请在左侧选择文件以查看代码差异”。
  - 选中文件状态：展示当前选中文件的按需渲染差异。
  - 加载中状态：右侧使用轻量级加载动画或骨架屏，提示“正在加载该文件的代码差异...”。

### 2. 前端请求逻辑变更
- **初始弹窗**：调用后端 `/preview_diff` 或 `/tasks/:id/diff` 接口时不传入 `file` 参数，后端只返回 `files` 变更文件列表，主 Diff 内容为空。渲染速度降为毫秒级。
- **点击文件**：当用户在左侧列表点击某个文件时：
  1. 设置 `selectedDiffFile`，并开启右侧 `loadingFileDiff` 加载状态。
  2. 向后端接口发送带有 `file` 参数的按需请求：`.../preview_diff?to=xxx&file=src/main.go`。
  3. 收到该特定文件的极小 Diff 响应后，将数据渲染到右侧，同时关闭加载状态。

### 3. 后端接口优化
- 接口 `/api/projects/:id/preview_diff` 和 `/api/tasks/:id/diff` 引入可选查询参数 `file`。
- 若 `file` 为空：
  - 跳过读取/计算全量大 Diff 文本，直接返回变更文件列表 `files`，`diff` 返回空字符串。
- 若 `file` 不为空：
  - 调用 Git 命令仅对比该特定文件路径：`git diff <from> <to> -- <file>`。
  - 直接返回当前单个文件的 diff 结果。

---

## 修改可能触达的现有功能
- 后端：`godeployer/api.go` 的 `HandleGetProjectPreviewDiff` 和 `HandleGetTaskDiff` 函数。
- 前端：`web/src/views/Dashboard.vue` 文件的 Diff 对话框 DOM 结构与点击处理方法。
