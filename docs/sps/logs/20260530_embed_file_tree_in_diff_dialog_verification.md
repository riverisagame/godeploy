# 验收验证报告: 将可选文件树嵌入 Diff 弹窗左侧

## 1. 验证目标
重构将原先在主页面由于“弹窗遮挡”导致用户在看 Diff 时无法交互过滤选择的“上线文件过滤树”，直接移植到全屏 Diff 对话框的左侧栏中，将文件过滤与 Diff 阅读融为一体。

## 2. 验证结果
- **单元测试框架**：Vitest
- **覆盖用例**：
  - `10. Diff 性能优化: 确保大文本被限制在 100KB 且不启用 lines 匹配防卡死` (验证通过)
  - `11. 单文件懒加载 Diff 机制: 点击文件时应异步拉取该文件的单独差异且初次加载时不获取大 diff` (验证通过)
- **测试通过率**：100% (共 33 个用例全部绿灯)

---

## 3. 代码变更回顾

### 3.1 前端重构 ([web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue))
- **主页面精简**：移除了原先触发部署表单下大而臃肿的 `file-filter-wrapper` 元素，移除了多余的 DOM 节点，使发布控制台主操作区极为干净。
- **弹窗左侧树形集成**：
  - 在 Diff 对话框的左侧栏 `.diff-left-sidebar` 中，移植并嵌入了 `el-tree`，绑定 `ref="fileTreeRef"`：
    ```html
    <el-tree
      ref="fileTreeRef"
      :data="fileTreeData"
      :show-checkbox="isPreDeploying"
      node-key="path"
      default-expand-all
      :default-checked-keys="defaultCheckedKeys"
      @node-click="handleFileTreeNodeClick"
    />
    ```
  - **模式感知**：设置 `:show-checkbox="isPreDeploying"`。在部署确认弹窗中展现 checkbox 用于勾选排除；在历史差异审计时，checkbox 会自动隐藏，以纯文本目录树方式仅作节点点击展示。
- **点击联动优化**：
  - 移除了旧的 `handleDiffRowClick`，编写了新节点点击监听器 `handleFileTreeNodeClick`：
    ```typescript
    const handleFileTreeNodeClick = async (nodeData: any) => {
      if (!nodeData.children) {
        selectedDiffFile.value = nodeData.path
        await loadSingleFileDiff(nodeData.path)
      }
    }
    ```
  - 点击非叶子节点（文件夹）时不触发任何操作，只有在点击具体文件（叶子节点）时才进行按需懒加载并高亮展现该文件的单文件差异。
