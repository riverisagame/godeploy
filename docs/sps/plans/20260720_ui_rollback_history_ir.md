# 纳米级执行计划 (IR): 前端部署历史与一键回滚操作闭环

## 阶段 1: 领域模型与数据库层改造 (Backend Data Layer)

### 子任务 1.1: 丰富 `domain.Deployment` 实体
- **目标文件**: `internal/domain/deployment.go`
- **动作**:
  - 在 `Deployment` 结构体中追加字段 `CreatedAt time.Time `json:"created_at"``。
  - 在 `Deployment` 结构体中追加字段 `ReleaseName string `json:"release_name"``。
  - 修改 `MarkSuccess` 方法签名为 `MarkSuccess(log string, releaseName string)`，内部执行 `d.ReleaseName = releaseName`。

### 子任务 1.2: 更新 `DeploymentModel` 与 Repository 映射
- **目标文件**: `internal/infrastructure/persistence/deployment_repository.go`
- **动作**:
  - `DeploymentModel` 结构体中追加 `ReleaseName string`。
  - `Save` 方法中双向映射 `ReleaseName`，即 `model.ReleaseName = d.ReleaseName`。
  - `FindByID` 方法中映射 `ReleaseName: model.ReleaseName` 以及 `CreatedAt: model.CreatedAt`。
  - `FindByEnvID` 方法中增加限制 `.Order("id desc").Limit(20)`，并映射 `ReleaseName` 与 `CreatedAt` 到 `domain.Deployment`。

---

## 阶段 2: 应用服务层与核心引擎串联 (Backend Engine Layer)

### 子任务 2.1: 升级 `DeployService.CompleteDeploy` 签名
- **目标文件**: `internal/application/deploy_service.go`
- **动作**:
  - 修改方法签名 `CompleteDeploy(id uint, success bool, log string, releaseName string) error`。
  - 调整成功时的调用：`d.MarkSuccess(log, releaseName)`。

### 子任务 2.2: 引擎流转中传递 ReleaseName
- **目标文件**: `internal/application/deploy_engine.go`
- **动作**:
  - 在 `StartDeploy` 方法的成功逻辑处，将原先的 `e.deploySvc.CompleteDeploy(deployment.ID, true, allLogs)` 替换为 `e.deploySvc.CompleteDeploy(deployment.ID, true, allLogs, releaseTimestamp)`。
  - 失败处替换为传空字符串 `""`。
  - 在 `Rollback` 方法中，将最后成功回写替换为 `e.deploySvc.CompleteDeploy(deployment.ID, true, allLogs, targetRelease)`，把目标 release 回填，保证依然可追溯。

### 子任务 2.3: `DeployHandler.Rollback` 适配 (无改动/预留)
- 经审查 `internal/interfaces/api/deploy_handler.go`，由于我们在引擎中直接传递了 `TargetRelease`，API层代码目前已接收了 `target_release` 字段，无需修改，天然兼容。

---

## 阶段 3: 前端 UI 操作闭环 (Frontend UI Layer)

### 子任务 3.1: 在 `EnvironmentConfig.vue` 增加部署历史表格
- **目标文件**: `web/src/views/EnvironmentConfig.vue`
- **动作**:
  - 在 `<template>` 中，在环境 `env-content-wrapper` 底部追加 `<h3>部署历史</h3>` 和 `<el-table>` 组件。
  - 表格列：ID、部署时间(`created_at`)、Commit/类型、Release目录名、状态(`status`)、操作列。
  - `created_at` 需要前端格式化为可读字符串 (如 `YYYY-MM-DD HH:mm:ss`)。

### 子任务 3.2: 接入回滚逻辑与日志查看跳转
- **目标文件**: `web/src/views/EnvironmentConfig.vue`
- **动作**:
  - `<script setup>` 中新增 API 请求方法 `fetchDeployments(envID: number)`。
  - 点击“环境折叠面板展开”或页面挂载时调用此方法，将历史列表绑定到对应的环境对象中 `env.deployments = ...`。
  - 操作列增加“查看日志”按钮，使用 `router.push('/deployments/' + row.id)`。
  - 操作列增加“回滚此版本”按钮（仅在 `status === 'success'` 时可见），点击触发带有 `ElMessageBox.confirm` 的提示弹窗，确认后调用 `POST /api/deployments/{row.id}/rollback`，参数带有 `target_release: row.release_name`。完成后刷新表格并跳转控制台。
