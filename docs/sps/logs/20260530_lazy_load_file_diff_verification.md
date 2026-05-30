# 验收验证报告: 单文件按需懒加载 Diff 与分栏重构

## 1. 验证目标
重构大项目（如 `crmeb-shop`）的代码 Diff 比对预览与审计历史比对界面，解决大 Diff 计算导致的后端 OOM/超时，以及前端渲染海量 DOM 导致一点击就卡死崩溃的性能红线问题。

## 2. 验证结果
- **单元测试框架**：Vitest
- **覆盖用例**：
  - `10. Diff 性能优化: 确保大文本被限制在 100KB 且不启用 lines 匹配防卡死` (验证通过)
  - `11. 单文件懒加载 Diff 机制: 点击文件时应异步拉取该文件的单独差异且初次加载时不获取大 diff` (验证通过)
- **测试通过率**：100% (共 32 个用例全部绿灯)

---

## 3. 重构功能变动

### 3.1 后端优化 ([godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go) & [godeployer/git.go](file:///d:/claudeprj/deploy/godeployer/git.go))
- 在 `git.go` 中编写并导出了 `GetDiffForFile` 函数，用于精确获取单文件的差异对比，并提供物理内存隔离截断保护。
- 在 `api.go` 的 `HandleGetProjectPreviewDiff` 与 `HandleGetTaskDiff` 中加入对 `file` 查询参数的处理：
  - 如果未指定 `file`：跳过读取/生成庞大的全量 diff 文本（直接将 `diff` 返回为空），仅通过 `git diff --name-only` 以毫秒级返回修改文件列表。
  - 如果指定了 `file`：调用按需 Git 逻辑，仅计算该单个文件的差异并轻量级返回。

### 3.2 前端优化 ([web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue))
- **分栏布局重构**：剥离原本的双 Tab 切换结构，设计为 IDE 级别的**左右分栏布局**（左侧 320px 宽度展示文件列表，右侧自适应区域渲染代码 Diff 或占位符提示）。
- **懒加载与缓存机制**：
  - 弹窗时先拉取文件清单，`diffText` 保持空。
  - 点击左侧文件行时，单独发起文件请求：`.../preview_diff?file=path` 或 `.../diff?file=path`。
  - 获取该单文件极小的 Diff（通常仅几 KB）后局部高亮渲染展现。
- **高亮联动高雅交互**：为选中的文件表格行配置了激活高亮样式 `row-active-selected`。
