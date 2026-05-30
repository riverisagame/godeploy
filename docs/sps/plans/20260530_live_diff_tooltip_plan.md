# Plan: 历史部署记录中 Live Diff 置灰按钮的交互提示优化

## Proposed Changes

### Frontend: Dashboard Page Tooltip Wrapping
- **文件**: [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- **修改点**:
  在 `web/src/views/Dashboard.vue` 的 `<el-radio-button value="live" ...>` 按钮内部，将按钮文本使用 `el-tooltip` 进行包裹。
  - **修改前**:
    ```vue
    <el-radio-button value="live" :disabled="!isPreDeploying && activeTask && activeTask.target_type !== 'commit'">与线上对比 (Live Diff)</el-radio-button>
    ```
  - **修改后**:
    ```vue
    <el-radio-button value="live" :disabled="!isPreDeploying && activeTask && activeTask.target_type !== 'commit'">
      <el-tooltip
        content="全量部署 (Branch/Tag) 历史仅归档本地变更 (Git Log Diff)，无 Live Diff 归档。仅 Commit 部署支持查看历史 Live Diff 快照。"
        placement="top"
        :disabled="isPreDeploying || !activeTask || activeTask.target_type === 'commit'"
      >
        <span>与线上对比 (Live Diff)</span>
      </el-tooltip>
    </el-radio-button>
    ```

---

## Verification Plan

### Automated Tests
- 运行前端单元测试 `npm run test` 确保无组件渲染与逻辑破坏。

### Manual Verification
- 启动项目并登录后台，进入“部署历史”页面。
- 点击一个 Branch 或 Tag 的部署历史，查看弹窗。
- 悬浮在被禁用的“与线上对比 (Live Diff)”按钮上，验证是否正确展示 Tooltip 提示。
