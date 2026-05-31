# DDD 全量战术模式落地 — 纳米级 TDD 实现计划

**关联规格**: `docs/sps/specs/20260531-ddd-full-tactical-design.md`
**状态**: 计划锁定，待逐项核对

---

## Phase 0: 测试基线

### T0.1 建立测试基线
- **文件**: 全部
- **操作**: 运行 `go test -v -race ./godeployer/...`，确认全部通过
- **操作**: 运行 `cd web && npm run test`，确认全部通过
- **出口**: 基线通过，记录结果到 `docs/sps/logs/20260531_phase0_baseline.md`

---

## Phase 1: 值对象落地 (DeployStatus)

> 策略：先定义类型 + 常量 → 再逐个文件替换字符串 → 每文件立即测试

### T1.1 新建 `domain/value_objects.go`
- **新建文件**: `godeployer/domain/value_objects.go`
- **内容**: `DeployStatus` 类型 + 5 个常量 + `Valid()/IsTerminal()/IsRunnable()` 方法
- **验证**: `go build ./godeployer/domain/` 编译通过

### T1.2 修改 `domain/entity.go` — DeployTask.Status 字段类型
- **文件**: `godeployer/domain/entity.go`
- **L103**: `Status string` → `Status DeployStatus`
- **验证**: `go build ./godeployer/domain/` 编译通过（会因调用方类型不匹配而失败，预期行为）

### T1.3 修改 `domain/repository.go` — 接口签名适配
- **文件**: `godeployer/domain/repository.go`
- **L23**: `UpdateTaskStatus(id int, status string) error` → `status DeployStatus`
- **L25**: `UpdateTaskStatusBatch(ids []int, status string) error` → `status DeployStatus`
- **验证**: `go build ./godeployer/domain/` 编译通过

### T1.4 修改 `infrastructure/db/task_repository.go` — 实现层签名适配 + 查询适配
- **文件**: `godeployer/infrastructure/db/task_repository.go`
- **L51**: 函数签名 `status string` → `status domain.DeployStatus`
- **L57**: `[]string{"pending", "deploying"}` → `[]domain.DeployStatus{domain.StatusPending, domain.StatusDeploying}`
- **L63**: 函数签名 `status string` → `status domain.DeployStatus`
- **验证**: `go test -v -race ./godeployer/infrastructure/db/`

### T1.5 修改 `infrastructure/db/db.go` — 修复停滞任务中的状态常量
- **文件**: `godeployer/infrastructure/db/db.go`
- **L105** (repairStalledTasks): `"aborted"` → `domain.StatusAborted`
- **验证**: `go build ./godeployer/infrastructure/db/`

### T1.6 修改 `application/deploy_service.go` — UpdateTaskStatus 签名适配
- **文件**: `godeployer/application/deploy_service.go`
- **L561**: `func (e *DeployEngine) UpdateTaskStatus(taskID int64, status string)` → `status domain.DeployStatus`
- **所有调用处替换**（~17处）: `"pending"` → `domain.StatusPending`, `"deploying"` → `domain.StatusDeploying`, `"success"` → `domain.StatusSuccess`, `"failed"` → `domain.StatusFailed`
- **验证**: `go test -v -race ./godeployer/application/`

### T1.7 修改 `interfaces/api/api.go` — 状态常量适配
- **文件**: `godeployer/interfaces/api/api.go`
- 所有 SQL INSERT/UPDATE 中字符串: `"pending"` → `domain.StatusPending`, `"deploying"` → `domain.StatusDeploying`, `"success"` → `domain.StatusSuccess`, `"failed"` → `domain.StatusFailed`
- TaskRes 结构体中 Status 字段保持不变（JSON 序列化自动用 DeployStatus 的 string 值）
- SQL WHERE 比较: `"deploying"`, `"pending"` → 对应常量
- **验证**: `go test -v -race ./godeployer/interfaces/api/`

### T1.8 适配所有 `*_test.go` 文件中的状态字符串
- **涉及文件**（基于探索数据）:
  - `application/engine_test.go` — ~25处 INSERT/UPDATE SQL + expectedStatus 参数
  - `interfaces/api/api_test.go` — ~8处
  - `interfaces/api/api_branch_diff_test.go` — ~4处
  - `interfaces/api/api_enhance_test.go` — ~2处
  - `interfaces/api/api_sqlite_purego_test.go` — ~1处
  - `infrastructure/db/task_repository_test.go` — L22: `Status = "pending"` → `Status = domain.StatusPending`
- **操作**: 全局替换 `"pending"` → `domain.StatusPending` 等，每文件改完立即 `go test`
- **验证**: `go test -v -race ./godeployer/...` 全部通过

### Phase 1 出口
- `go test -v -race ./godeployer/...` 全部通过
- 不允许任何 `"pending"` / `"deploying"` / `"success"` / `"failed"` / `"aborted"` 字符串直接出现在非 domain 包的代码中

---

## Phase 2: 实体充血 (DeployTask/DeployJob 方法)

### T2.1 新增 `DeployTask` 状态机方法
- **文件**: `godeployer/domain/entity.go`
- **新增**（文件末尾）:
  - `var ErrInvalidTransition = errors.New("...")`
  - `func (t *DeployTask) Start() error`
  - `func (t *DeployTask) Complete() error`
  - `func (t *DeployTask) Fail() error`
  - `func (t *DeployTask) Abort() error`
  - `func (t *DeployTask) IsActive() bool`
- **新增 import**: `"errors"`
- **验证**: `go test -v -race ./godeployer/domain/`

### T2.2 新增 `NewDeployJob` 构造函数
- **文件**: `godeployer/domain/entity.go`
- **新增**: `func NewDeployJob(taskID int64, config *Config, logFilePath string) *DeployJob`
- **验证**: `go build ./godeployer/domain/`

### T2.3 补全 `ProjectRepository` 接口
- **文件**: `godeployer/domain/repository.go`
- **L14-15**: 替换为 `GetAllProjects(config *Config) []ProjectSummary` + `ProjectSummary` 结构体
- **验证**: `go build ./godeployer/domain/`

### T2.4 编写充血模型单元测试
- **新建文件**: `godeployer/domain/entity_test.go` (追加)
- **用例**: 
  - pending → Start() → deploying ✓
  - deploying → Complete() → success ✓
  - deploying → Fail() → failed ✓
  - pending → Abort() → aborted ✓
  - success → Start() → ErrInvalidTransition ✗
  - 所有边界条件
- **验证**: `go test -v -race ./godeployer/domain/`

### Phase 2 出口
- DeployTask 状态变更全部通过实体方法而非直接赋值
- DeployJob 构造通过 `NewDeployJob` 而非结构体字面量
- 单元测试覆盖所有状态转换

---

## Phase 3: 领域服务 + NodeExecutor 接口

### T3.1 新建 `domain/deployment_service.go`
- **新建文件**: `godeployer/domain/deployment_service.go`
- **内容**:
  - `NodeExecutor` 接口: `Rsync()`, `SwitchSymlink()`, `RunCommand()`
  - `DeploymentService` 结构体 + `NewDeploymentService(taskRepo)`
  - `Execute()` 方法 — 从 `application/deploy_service.go:RunDeploy` 中迁移两阶段提交逻辑
  - `ShouldRollback()` 方法
- **注意**: Phase 3 只定义接口和骨架，不迁移实现逻辑（避免破坏现有功能）。Execute 方法体在 Phase 5 填入
- **验证**: `go build ./godeployer/domain/`

### T3.2 编写 DeploymentService 纯逻辑测试
- **新建文件**: `godeployer/domain/deployment_service_test.go`
- **用例**: 
  - 阶段1全成功 → 阶段2执行 → 状态 success
  - 阶段1任一失败 → 不进入阶段2 → 回滚判断
  - Context 取消 → 终止执行
- **验证**: `go test -v -race ./godeployer/domain/`

### Phase 3 出口
- `domain.NodeExecutor` 接口定义在 domain 层
- `DeploymentService` 骨架完成，测试通过

---

## Phase 4: SSH 适配层

### T4.1 新建 `infrastructure/ssh/adapter.go`
- **新建文件**: `godeployer/infrastructure/ssh/adapter.go`
- **内容**:
  - `NodeAdapter` 结构体，包装 `*SSHPool`
  - `NewNodeAdapter(pool) *NodeAdapter`
  - `Rsync()` — 从 `DeployEngine.rsyncToServer` 逻辑迁移
  - `SwitchSymlink()` — 从 `DeployEngine.SwitchSymlink` 逻辑迁移
  - `RunCommand()` — 底层命令执行包装
- **验证**: `go build ./godeployer/infrastructure/ssh/`

### T4.2 编写 NodeAdapter 测试
- **新建文件**: `godeployer/infrastructure/ssh/adapter_test.go`
- **用例**: 验证 Rsync/SwitchSymlink/RunCommand 方法签名与 domain.NodeExecutor 一致
- **验证**: `go test -v -race ./godeployer/infrastructure/ssh/`

### Phase 4 出口
- `ssh.NodeAdapter` 实现 `domain.NodeExecutor` 接口（编译期验证）
- 现有 `ssh.RemoteExecutor` 接口不受影响

---

## Phase 5: Application 层拆分 + DeploymentService 接入

### T5.1 新建 `application/process_utils.go`
- **新建文件**: `godeployer/application/process_utils.go`
- **移植**: 从 `deploy_service.go` L693-719 移动 `runCmd` 函数
- **删除**: `deploy_service.go` 中对应行
- **验证**: `go test -v -race ./godeployer/application/`

### T5.2 新建 `application/deploy_executor.go`
- **新建文件**: `godeployer/application/deploy_executor.go`
- **移植**:
  - `getPool()` (L55-68)
  - `RunLocalBuild()` (L70-91)
  - `SwitchSymlink()` (L94-112) — 保留但内部改为调用 `e.executor.SwitchSymlink()`（后续适配）
- **删除**: `deploy_service.go` 中对应行
- **验证**: `go test -v -race ./godeployer/application/`

### T5.3 新建 `application/rollback_service.go`
- **新建文件**: `godeployer/application/rollback_service.go`
- **移植**:
  - `RunRollbackToTask()` (L116-141)
  - `RunRollback()` (L145-171)
- **删除**: `deploy_service.go` 中对应行
- **验证**: `go test -v -race ./godeployer/application/`

### T5.4 新建 `application/diff_service.go`
- **新建文件**: `godeployer/application/diff_service.go`
- **移植**:
  - `cacheTaskDiff()` (L571-668)
  - `generateTaskDiff()` (L670-691)
- **删除**: `deploy_service.go` 中对应行
- **验证**: `go test -v -race ./godeployer/application/`

### T5.5 改造 `application/deploy_service.go` — 注入 DeploymentService
- **修改**: `DeployEngine` 结构体新增 `deploySvc *domain.DeploymentService` 字段
- **修改**: `NewDeployEngine` 构造函数签名新增参数
- **修改**: `executor ssh.RemoteExecutor` → `executor domain.NodeExecutor`
- **修改**: `RunDeploy` 方法中调用 `e.deploySvc.Execute()` 替代内联的两阶段提交逻辑
- **修改**: `main.go` 中 `NewDeployEngine` 调用处传入 `deploySvc` 和 `adapter`
- **验证**: `go test -v -race ./godeployer/...`

### Phase 5 出口
- `deploy_service.go` 缩减到 ~200 行
- DeployEngine 通过 `domain.DeploymentService` 编排部署，通过 `domain.NodeExecutor` 执行操作
- 全量测试通过

---

## Phase 6: Interfaces 层拆分

### T6.1 新建 `interfaces/api/routes.go`
- **新建文件**: `godeployer/interfaces/api/routes.go`
- **移植**:
  - `APIHandler` 结构体（L34-40）— Executor 字段类型改为 `domain.NodeExecutor`
  - `SetupRoutes()` (L42-44)
  - `SetupRoutesWithExecutor()` (L48-117)
- **删除**: `api.go` 中对应行
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.2 新建 `handler_auth.go`
- **新建文件**: `godeployer/interfaces/api/handler_auth.go`
- **移植**: `LoginRequest`(L119-122) + `HandleLogin`(L124-158)
- **删除**: `api.go` 中对应行
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.3 新建 `handler_deploy.go`
- **新建文件**: `godeployer/interfaces/api/handler_deploy.go`
- **移植**: `CreateTaskRequest`(L599-606) + `HandleCreateTask`(L608-750) + `HandleGetTasks`(L753-795) + `HandleGetTaskDetail`(L798-812) + `HandleGetTaskLog`(L816-873) + `HandleWSLog`(L883-965) + `TaskRes`
- **删除**: `api.go` 中对应行
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.4 新建 `handler_rollback.go`
- **新建文件**: `godeployer/interfaces/api/handler_rollback.go`
- **移植**: `HandleTriggerRollback`(L969-1022)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.5 新建 `handler_diff.go`
- **新建文件**: `godeployer/interfaces/api/handler_diff.go`
- **移植**: `diffSemaphore`(L32) + `HandleGetTaskDiff`(L1025-1205) + `HandleGetProjectPreviewDiff`(L332-441) + `HandleGetProjectRefs`(L242-296) + `HandleGetProjectCommits`(L298-330)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.6 新建 `handler_webhook.go`
- **新建文件**: `godeployer/interfaces/api/handler_webhook.go`
- **移植**: `UpdateGitBindingRequest`(L443-446) + `HandleGithubWebhook`(L1217-1333) + `ComputeGithubSignature`(L1209-1214)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.7 新建 `handler_user.go`
- **新建文件**: `godeployer/interfaces/api/handler_user.go`
- **移植**: `UserResponse`(L485-493) + `CreateUserRequest`(L513-520) + `UpdateUserRequest`(L549-555) + `HandleGetUsers`(L495-511) + `HandleCreateUser`(L522-547) + `HandleUpdateUser`(L557-583) + `HandleDeleteUser`(L585-597)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.8 新建 `handler_user_git.go`
- **新建文件**: `godeployer/interfaces/api/handler_user_git.go`
- **移植**: `HandleGetUserGitBinding`(L448-461) + `HandleUpdateUserGitBinding`(L463-483)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.9 新建 `handler_project.go`
- **新建文件**: `godeployer/interfaces/api/handler_project.go`
- **移植**: `UpdatePermissionsRequest`(L215-217) + `HandleGetProjects`(L178-213) + `HandleUpdateUserPermissions`(L219-240) + `checkProjectAccess`(L161-176)
- **验证**: `go build ./godeployer/interfaces/api/`

### T6.10 新建 `handler_prune.go`
- **新建文件**: `godeployer/interfaces/api/handler_prune.go`
- **移植**: `HandleSystemPrune`(L1337-1480) + `extractFileDiffFromLog`(L1485-1510)
- **删除**: `api.go` 中剩余所有行
- **验证**: `go test -v -race ./godeployer/interfaces/api/`

### T6.11 删除旧 `api.go`
- **删除文件**: `godeployer/interfaces/api/api.go`（所有内容已迁移）
- **验证**: `go test -v -race ./godeployer/interfaces/api/`

### Phase 6 出口
- `api.go` 不存在，被 10 个文件替代
- 所有 API 测试通过
- 路由表在 `routes.go` 中集中可读

---

## Phase 7: config.go 归位

### T7.1 新建 `infrastructure/config/loader.go`
- **新建文件**: `godeployer/infrastructure/config/loader.go`
- **包名**: `config`
- **移植**: 从 `godeployer/config.go` 移动 `LoadConfig` 函数
- **验证**: `go build ./godeployer/infrastructure/config/`

### T7.2 移动 `config_test.go`
- **操作**: 移动 `godeployer/config_test.go` → `godeployer/infrastructure/config/loader_test.go`
- **修改**: import `"deploy/godeployer"` → `"deploy/godeployer/infrastructure/config"`，调用 `godeployer.LoadConfig()` → `config.LoadConfig()`
- **验证**: `go test -v -race ./godeployer/infrastructure/config/`

### T7.3 修改 `main.go` 引用
- **文件**: `godeployer/main.go`
- **新增 import**: `"deploy/godeployer/infrastructure/config"`
- **L37**: `LoadConfig(configPath)` → `config.LoadConfig(configPath)`
- **验证**: `go test -v -race ./godeployer/main_test.go`（BootstrapApp 仍通过同包调用，内部已适配）

### T7.4 删除根目录旧文件
- **删除**: `godeployer/config.go`
- **验证**: `go test -v -race ./godeployer/...` 全部通过

### Phase 7 出口
- `godeployer/` 根目录仅保留 `main.go` 和 `main_test.go`

---

## Phase 8: 全量回归

### T8.1 全量后端测试
- **命令**: `go test -v -race ./...`
- **要求**: 全部通过，零 data race

### T8.2 编译验证
- **命令**: `go build .`
- **要求**: 编译通过，无链接错误

### T8.3 前端单元测试
- **命令**: `cd web && npm run test`
- **要求**: 全部通过

### T8.4 前端 E2E 测试
- **命令**: `cd web && npm run test:e2e`
- **要求**: 全部通过

### T8.5 更新 MASTER_LOG
- **文件**: `docs/sps/MASTER_LOG.md`
- **新增行**: `| 2026-05-31 | DDD-001 | DDD_TACTICAL | docs/sps/plans/20260531-ddd-full-tactical-plan.md | BUILD_SUCCESS |`

---

## 回滚策略

每个 Phase 内部通过 git commit 原子化保存。若任一 Phase 无法通过测试：
1. `git reset --hard HEAD~1` 回退到上一 Phase
2. 分析根因，重新规划
3. 从失败点继续

## 预计改动统计

| 层 | 新建 | 修改 | 删除 |
|----|------|------|------|
| domain | 3 | 2 | 0 |
| application | 4 | 1 | 0 |
| infrastructure | 3 | 2 | 0 |
| interfaces | 10 | 0 | 1 |
| 根目录 | 0 | 1 | 2 |
| **合计** | **20** | **6** | **3** |
