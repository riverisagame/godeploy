# 纳米级执行计划: 将文件过滤树嵌入 Diff 弹窗左侧

## 1. 拟修改文件清单

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 移除主页面发布卡片中的 `file-filter-wrapper` 面板。
- 将 `el-tree` 移植到 Diff 对话框左侧栏，控制其是否显示 Checkbox 勾选框。
- 绑定 `@node-click="handleFileTreeNodeClick"` 并编写事件处理器。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 移除主页面部署表单中的 `file-filter-wrapper`
- **目标路径**：`web/src/views/Dashboard.vue` 第 188-204 行。
- **改动详情**：
  - 将此部分的整个 HTML 树元素结构彻底删除，使主界面保持极度整洁。
- **预估代码改动量**：约 15 行。

### 任务 2.2: 移植 `el-tree` 到弹窗左侧
- **目标路径**：`web/src/views/Dashboard.vue` 第 360-382 行附近（即上次添加的 `.diff-left-sidebar` 区域）。
- **改动详情**：
  - 移除原先的只读 `el-table` 表格。
  - 填入具有可选功能的 `el-tree`，配置参数：
    ```html
    <el-tree
      ref="fileTreeRef"
      :data="fileTreeData"
      :show-checkbox="isPreDeploying"
      node-key="path"
      default-expand-all
      :default-checked-keys="defaultCheckedKeys"
      :props="{ label: 'label', children: 'children' }"
      @node-click="handleFileTreeNodeClick"
      style="background: transparent; color: #e0e0e0;"
    />
    ```
- **预估代码改动量**：约 15 行。

### 任务 2.3: 编写点击节点处理器 `handleFileTreeNodeClick`
- **目标路径**：`web/src/views/Dashboard.vue` 第 610 行附近（即原本的 `handleDiffRowClick` 位置）。
- **改动详情**：
  - 彻底删除已经不再使用的 `handleDiffRowClick` 函数。
  - 新增 `handleFileTreeNodeClick` 函数：
    ```typescript
    const handleFileTreeNodeClick = async (nodeData: any) => {
      if (!nodeData.children) {
        selectedDiffFile.value = nodeData.path
        await loadSingleFileDiff(nodeData.path)
      }
    }
    ```
- **预估代码改动量**：约 10 行。

---

## 3. 验证方案

### 自动化测试
- 运行 `cd web && npm run test`。
- 修改已有的单元测试中对 `.diff-file-table` / `handleDiffRowClick` 的模拟，更新为对 `handleFileTreeNodeClick` 的调用断言，保证测试通过。

### 手动验证
- 启动项目，选择要发布的版本，确认主页面不再有多余的文件树，卡片非常整洁。
- 点击“预览 Diff”或“触发上线”，确认：
  - 弹出的比对窗口左侧完整展示出带勾选框的文件目录树。
  - 勾选和取消勾选状态能够正确被记录，点击“确认并部署”时依然能根据勾选情况生成过滤规则。
  - 点击任何文件节点（无子树分支），右侧能立刻渲染加载出对应的 Diff，极为流畅。
