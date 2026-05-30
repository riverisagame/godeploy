# 纳米级执行计划: 解决 Diff 界面点击卡死问题

## 1. 拟修改文件清单

### [MODIFY] [Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 优化计算属性 `highlightedDiff` 中的安全防御截断限制和配置。
- 优化 `handleDiffRowClick` 中的定位逻辑，避免同步重排与卡死。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 优化 `highlightedDiff` 计算属性
- **目标路径**：`web/src/views/Dashboard.vue` 第 617-645 行。
- **改动详情**：
  - 将 `LIMIT` 变量从 `300 * 1024` 修改为 `100 * 1024` (100KB)。
  - 在 `html(text, { ... })` 传参中，移除 `matching: 'lines'` 或将其修改为 `matching: 'none'`，避免高复杂度的 Levenshtein 行匹配逻辑。
- **预估代码改动量**：约 5 行。

### 任务 2.2: 优化 `handleDiffRowClick` 函数
- **目标路径**：`web/src/views/Dashboard.vue` 第 582-600 行。
- **改动详情**：
  - 将 `nextTick` 回调更改为配合一个微小的 `setTimeout` 或 `requestAnimationFrame`（双重保险），让 Vue 在切换 Tab 挂载 DOM 以及浏览器完成渲染之后，再执行 `.d2h-file-header` 的查询定位。
  - 在 `el.innerText.includes(row.path)` 的匹配中做一层简单的非空防护，确保绝对安全。
- **预估代码改动量**：约 10 行。

---

## 3. 验证方案

### 自动化测试
- 前端测试：运行已有的 Vitest 单元测试确保不破坏核心渲染契约。
  - 命令：`cd web && npm run test`

### 手动验证
- 选择大项目进行发布预览 Diff，点击文件相对路径，确认：
  - 文件精准定位和滚动依然正常生效。
  - 界面完全流畅，没有发生任何浏览器挂起或一点击就卡死的现象。
