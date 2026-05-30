# 纳米级执行计划 - 修复 Diff 弹窗文件树渲染

## 1. 拟修改文件清单
### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 修改 `previewDeployDiff` 方法，使其在获取到预览文件列表后同时更新 `fileTreeData`、`rawFilesList` 和 `defaultCheckedKeys`。
- 修改 `showDiff` 方法，使其在解析出变更文件列表后，提取出文件路径列表，并更新 `fileTreeData`、`rawFilesList` 和 `defaultCheckedKeys`。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 修复 `previewDeployDiff` 方法
- **目标路径**：`web/src/views/Dashboard.vue` 第 562-586 行。
- **改动详情**：
  - 原逻辑：
    ```typescript
    const files = res.data.files || []
    parsedDiffFiles.value = files.map((f: string) => ({ status: 'M', statusText: '变更', path: f }))
    ```
  - 新逻辑：
    ```typescript
    const files = res.data.files || []
    rawFilesList.value = files
    parsedDiffFiles.value = files.map((f: string) => ({ status: 'M', statusText: '变更', path: f }))
    fileTreeData.value = buildTree(files)
    defaultCheckedKeys.value = [...files]
    ```
- **代码变动量**：新增 3 行。

### 任务 2.2: 修复 `showDiff` 方法
- **目标路径**：`web/src/views/Dashboard.vue` 第 992-1039 行。
- **改动详情**：
  - 原逻辑：
    ```typescript
    // 解析变更文件列表
    if (rawFiles) {
      const lines = rawFiles.split('\n')
      const files: any[] = []
      lines.forEach(line => {
        // ... 解析逻辑
        files.push({ status, statusText, path })
      })
      parsedDiffFiles.value = files
    } else {
      parsedDiffFiles.value = [{ status: '?', statusText: '无数据', path: '暂无变更文件解析数据' }]
    }
    ```
  - 新逻辑：
    ```typescript
    // 解析变更文件列表
    if (rawFiles) {
      const lines = rawFiles.split('\n')
      const files: any[] = []
      const filePaths: string[] = []
      lines.forEach(line => {
        // ... 解析逻辑
        files.push({ status, statusText, path })
        filePaths.push(path)
      })
      parsedDiffFiles.value = files
      rawFilesList.value = filePaths
      fileTreeData.value = buildTree(filePaths)
      defaultCheckedKeys.value = [...filePaths]
    } else {
      parsedDiffFiles.value = [{ status: '?', statusText: '无数据', path: '暂无变更文件解析数据' }]
      rawFilesList.value = []
      fileTreeData.value = [{ label: '暂无变更文件解析数据', path: '暂无变更文件解析数据' }]
      defaultCheckedKeys.value = []
    }
    ```
- **代码变动量**：约 10 行。

---

## 3. 验证方案

### 自动化单元测试
- 在 `web` 目录下执行单元测试：`npm run test`。
- 新增单元测试断言，确保在调用 `previewDeployDiff` 和 `showDiff` 后，`fileTreeData` 正确包含了预期的节点。

### 手动验证
1. 打开浏览器控制台。
2. 在主页选择项目和分支，点击对应环境的“预览 Diff”，检查弹窗左侧是否显示了正确的文件树，点击叶子节点，右侧是否异步渲染 Diff。
3. 点击历史部署任务行的“查看 Diff”，检查弹窗左侧是否显示了正确的文件树，且不可勾选复选框（因为是非部署状态）。
