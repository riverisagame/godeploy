# ADR: 前端部署历史与一键回滚操作闭环

## 背景与现状
当前部署引擎已经完成了回滚 API (`POST /api/deployments/{id}/rollback`) 的支持，但在前端缺乏部署记录的展示，也缺少执行回滚的按钮入口。
由于历史记录只记录了部署的 `ID`, `CommitHash`, `Status`，并未记录目标机的物理 `Release` 目录名（即时间戳），系统在回滚时无法通过 ID 精准定位应该指向的目标目录。

## 需求深挖与质疑 (Adversarial Questioning)
- **质疑1**：如果允许回滚到任何一次部署记录，会不会出现回滚到“失败版本”导致系统瘫痪的风险？
  - **对齐**：必须在查询 API 返回的历史列表中，**仅针对状态为 `success` 的版本提供“回滚”操作**。失败版本不可作为回滚目标。
- **质疑2**：怎么确定要回滚的那个版本目录（Release Name）？
  - **对齐**：必须在 `domain.Deployment` 中增加 `ReleaseName string` 字段，通过 `gorm.Model` 的 `CreatedAt time.Time` 暴露创建时间给前端。并在部署结束 (`CompleteDeploy`) 时将 `releaseTimestamp` 存入数据库中。回滚接口将根据 `ReleaseName` 来执行 `ln -sfn` 软链切换。
- **质疑3**：回滚这动作本身会不会产生新的部署记录？
  - **对齐**：会的，回滚动作会在 `DeployService.TriggerDeploy` 中生成一条 `commit_hash` 为 `ROLLBACK_TO_{ReleaseName}` 的特殊新记录，以便溯源。

## 修改影响面 (Impact Assessment)
1. **数据库层**：`DeploymentModel` 新增 `ReleaseName` 字段，需要 GORM 自动迁移。由于采用 Sqlite，GORM 的 `AutoMigrate` 会自动追加字段，**无数据破坏风险（零污染）**。
2. **领域模型层**：`domain.Deployment` 新增 `ReleaseName`，JSON 序列化需要增加 `CreatedAt time.Time`。
3. **接口层**：`DeployHandler.ListDeployments` 暴露给前端的内容增加时间和 ReleaseName。限制返回条数，按时间倒序。
4. **前端UI**：`EnvironmentConfig.vue` 将增加一个【部署历史】的表格展示区，并支持回滚按钮。

## 性能对冲与安全性审计
- 增加字段 `ReleaseName` 对并发操作无影响。
- 前端分页/条数限制（为保证响应速度150ms以内，后端 `FindByEnvID` 取最近 20 条即可）。
- 回滚动作具备防御性：必须用户手工确认，并通过弹窗阻断误点。

## 结论
采用在原表追加 `ReleaseName` 和向前端透传 `CreatedAt`，前端在 `EnvironmentConfig.vue` 组件的折叠面板内追加展示最新 20 条发布记录，并为 success 提供回滚。
