# 纳米级执行计划: 单文件懒加载 Diff 与左右分栏重构

## 1. 拟修改文件清单

### [MODIFY] [godeployer/api.go](file:///d:/claudeprj/deploy/godeployer/api.go)
- 优化 `HandleGetProjectPreviewDiff` 函数，增加可选 `file` 参数，控制仅对比特定文件或只获取文件列表。
- 优化 `HandleGetTaskDiff` 函数，增加可选 `file` 参数并进行类似的按需读取。

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 重构 Diff 对话框内部布局为左右分栏。
- 实现单文件点击时的懒加载逻辑与加载占位状态。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 重写后端 `HandleGetProjectPreviewDiff`
- **目标路径**：`godeployer/api.go` 第 320-394 行。
- **改动详情**：
  - 从 query 中获取 `file` 参数。
  - 若 `file != ""`：
    - 调用修改后的 `GetDiff`（或直接使用 `git diff` 指向特定文件路径，限制输出）获取单文件 diff 文本。
    - 直接向前端返回包含 `"diff": diffText` 且 `"files": []` 的响应并直接 return。
  - 若 `file == ""`：
    - 不调用 `GetDiff` 获取大差异。直接 `diffText = ""`。
    - 仅保留原本的获取文件列表逻辑，通过 `git diff --name-status`（或 `git diff --name-only`）获得列表，最后返回给前端。
- **预估代码改动量**：约 25 行。

### 任务 2.2: 重写后端 `HandleGetTaskDiff`
- **目标路径**：`godeployer/api.go` 第 969-1120 行。
- **改动详情**：
  - 从 query 中获取 `file` 参数。
  - 如果 `file != ""`，绕过缓存逻辑（或者在缓存基础上，直接在 git 中读取该单文件的实时 diff）：
    - `cmd := exec.CommandContext(ctx, "git", "diff", prevCommit, currentCommit, "--", file)`
    - 返回给前端包含该特定文件 diff 的 JSON 响应。
  - 如果 `file == ""`，跳过大 Diff 计算。直接读取缓存中被修改的文件列表（`files`）或者通过 `git diff --name-status` 快速计算出文件列表并返回，设置 `diff` 为空。
- **预估代码改动量**：约 25 行。

### 任务 2.3: 重构前端 Diff 对话框布局与加载状态
- **目标路径**：`web/src/views/Dashboard.vue`。
- **改动详情**：
  - 在 `<el-dialog v-model="diffVisible"` 下，移除 `<el-tabs v-else ...>` 的双 Tab 切换结构。
  - 采用 Flex 左右分栏布局结构：
    ```html
    <div class="diff-layout-split">
      <div class="diff-left-file-list">
         <!-- 变更文件列表表格 -->
      </div>
      <div class="diff-right-content-area" v-loading="loadingFileDiff">
         <!-- 占位符提示，或被选文件的高亮渲染 Diff -->
      </div>
    </div>
    ```
  - 新增状态变量：
    - `selectedDiffFile = ref('')`：保存当前点击的文件路径。
    - `loadingFileDiff = ref(false)`：单文件 diff 加载状态。
  - 新增方法 `loadSingleFileDiff(path: string, envId?: string)`：
    - 异步调用对应的预览/历史 diff 接口，并传递 `file=path` 参数。
    - 接收到返回值后写入 `diffText`（因为只包含单个文件的 diff，大小一般在几 KB 左右，即使是最大也是极小值）。
  - 重写 `handleDiffRowClick(row)`：
    - 触发 `selectedDiffFile.value = row.path`。
    - 调用 `loadSingleFileDiff(row.path)` 载入并渲染。
- **预估代码改动量**：约 60 行。

---

## 3. 验证方案

### 自动化测试
- 确保运行 `cd web && npm run test`。
- 我们将更新已有的单元测试用例，增加对懒加载单文件机制的行为验证。

### 手动验证
- 启动项目，点击对比或预览 Diff。
- 打开 Diff 弹框，确认：
  - 弹框瞬间出现（无需任何接口等待，只拉取了列表）。
  - 右侧默认提示“请从左侧选择要查看差异的文件”。
  - 点击左侧文件，右侧展示轻量级 loading 后迅速渲染出当前单个文件的差异，没有发生卡顿。
