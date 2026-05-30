# 架构决策记录 (ADR) - 历史部署记录中 Live Diff 置灰按钮的交互提示优化

## 1. 业务背景与痛点
用户反馈在“历史记录”列表中，某些部署任务的“与线上对比”按钮一直无法点击，处于置灰状态。
- **原因**：根据 [DUAL_DIFF_PERSISTENCE (ADR-018)](file:///d:/claudeprj/deploy/docs/sps/decisions/20260530_dual_diff_persistence_rules.md)，为防止磁盘爆满，全量部署（Branch / Tag）任务在上线后不会保留 Live Diff 快照，前端也随之置灰该按钮。
- **痛点**：缺乏明确的信息引导，用户无法理解为何该按钮被禁用，易误判为系统 Bug。

## 2. 决策方案 (方案 A)
- **选择**：维持现有的性能防护与磁盘安全设计，不强行生成大体积的全量 Live Diff 快照。
- **改进方式**：在前端 `Dashboard.vue` 的 `与线上对比 (Live Diff)` 按钮外包裹 `el-tooltip` 组件。
  - 当按钮被禁用（即 `!isPreDeploying && activeTask && activeTask.target_type !== 'commit'`）时，悬浮显示说明文字：“全量部署 (Branch/Tag) 历史仅归档本地变更 (Git Log Diff)，无 Live Diff 归档。仅 Commit 部署支持查看历史 Live Diff 快照。”
  - 当按钮未被禁用时，悬浮提示保持默认或不显示（通过 `el-tooltip` 的 `disabled` 属性动态控制）。

## 3. 并发与安全影响
- **零侵入性**：该改动纯属前端展示优化，不涉及后端逻辑或数据库的变更。
- **无性能负担**：组件层面使用 Element Plus 的原生 `el-tooltip`，无 CPU 与内存开销。
