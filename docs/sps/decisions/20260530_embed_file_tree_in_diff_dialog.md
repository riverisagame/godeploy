# ADR: 将“上线文件勾选过滤树”融入“Diff 比对弹窗”的左侧栏

## 问题背景
用户反馈在界面上“仍然没有看到哪里能可选文件”。
这是因为：
1. 之前的上线文件可选列表放在“触发部署”主页面的表单内部，当用户点击“触发上线”或“预览 Diff”弹出比对窗口后，大弹窗遮挡了主页面，导致用户在预览 Diff 时无法看到或操作可选文件列表。
2. 之前的 Diff 弹窗的左侧只是一个纯文本的只读变更列表，不支持任何勾选排除操作。

## 重构方案：弹窗左侧嵌入可选文件树
为了实现“边看 Diff 边挑选要上线的文件”，我们将原先在主页面的“文件选择过滤树”彻底迁移到 **“Diff 比对弹窗”的左侧栏**。

### 1. 弹窗左侧树形改造
- 移除左侧原本的只读 `el-table`。
- 嵌入带勾选功能的 `el-tree`，绑定 `ref="fileTreeRef"`：
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
- **智能勾选框**：在“部署前预览确认”（`isPreDeploying === true`）时显示勾选框，允许用户取消勾选以排除同步；在“历史记录对比”时自动隐藏勾选框，仅提供节点点击查看 Diff 差异。

### 2. 节点点击懒加载
- 点击树中的文件节点（无子节点）时，触发 `selectedDiffFile` 更新，并按需发起单文件差异请求展示在右侧。

---

## 修改可能触达的现有功能
- 前端：`web/src/views/Dashboard.vue`。
  - 移除主页面部署表单中的 `file-filter-wrapper`。
  - 重构 Diff 弹框左侧为支持带 checkbox 的 `el-tree`。
  - 替换事件处理方法 `handleDiffRowClick` 为 `handleFileTreeNodeClick`。
