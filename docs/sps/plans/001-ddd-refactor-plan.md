# 目标 (Goal)
将 `godeployer` 扁平包重构为标准 DDD 结构 (Clean Architecture)，确保业务逻辑“0侵入”及通过纳米级 TDD 保证原有功能 100% 完整。

## User Review Required
> [!IMPORTANT]
> 这是一个巨大的架构调整。为了满足“原子化执行”和“纳米级 TDD”，本计划分为 5 个 Phase。每个 Phase 的改动都会伴随 `go test` 的红绿验证。因为我们在调整包结构，会导致短暂的编译失败，我们需要您的同意以“新建 -> 验证 -> 切换 -> 清理”的策略执行。

## Proposed Changes

### Phase 1: 构建 Domain 层 (无外部依赖)
将散落在各个文件中的结构体抽离到领域层，剥离所有外部依赖（如 `*sql.DB`）。

#### [NEW] `godeployer/domain/entity.go`
- 移动 `api.go` 中的 `UserResponse`, `LoginRequest`。
- 移动 `engine.go` 中的 `DeployJob`。
- 移动 `config.go` 中的 `Config`, `ServerConfig` 等。

#### [NEW] `godeployer/domain/repository.go`
- 提取接口 `UserRepository` 和 `TaskRepository`，隔离直接的 SQL 查询。

---

### Phase 2: 构建 Infrastructure 层 (被驱动端)
将 SQLite、SSH、EventBus 移动到具体的基础设施包，实现领域层接口。

#### [NEW] `godeployer/infrastructure/sqlite/db.go`
- 将原 `db.go` 移动至此，将其改造为结构体方法以实现 `domain.UserRepository` 等接口。
- [NEW] 同步移动对应的 `db_test.go`。

#### [NEW] `godeployer/infrastructure/ssh/executor.go`
- 将原 `ssh.go` 和 `ssh_pool.go` 移动至此。

#### [NEW] `godeployer/infrastructure/notifier/eventbus.go`
- 将原 `notifier.go` 移动至此。

---

### Phase 3: 构建 Application 层 (用例/服务层)
剥离原先混杂在 API Handler 和 Engine 里的业务逻辑，建立 Application Service。

#### [NEW] `godeployer/application/deploy_service.go`
- 将 `engine.go` (`DeployEngine`) 迁移至此。修改其对 `*sql.DB` 的硬编码依赖，改为依赖注入 `domain.TaskRepository`。
- [NEW] 同步移动 `engine_test.go`。

#### [NEW] `godeployer/application/auth_service.go`
- 将 `auth.go` (JWT, Hash) 迁移至此。

---

### Phase 4: 构建 Interfaces 层 (驱动端)
将 Web 层完全隔离。

#### [NEW] `godeployer/interfaces/api/routes.go`
#### [NEW] `godeployer/interfaces/api/handlers.go`
- 移动 `api.go`，将 `APIHandler` 改为依赖 Application 层的 Service，不再直接包含 `*sql.DB`。
- [NEW] 同步移动所有的 `api_*_test.go`。

---

### Phase 5: 依赖注入与收尾
在启动入口组装所有的包，并进行全量清理。

#### [MODIFY] `godeployer/main.go`
- 重构 `BootstrapApp` 和 `StartServer`，完成依赖注入链：实例化 DB -> `sqlite.NewUserRepository()` -> `application.NewAuthService()` -> `api.NewAPIHandler()`。

#### [DELETE] 删除旧的扁平文件
- 删除 `godeployer/api.go`, `godeployer/db.go`, `godeployer/engine.go`, `godeployer/auth.go`, `godeployer/ssh.go`, `godeployer/ssh_pool.go`, `godeployer/config.go` 等根目录文件。

## Verification Plan

### Automated Tests
- 在每一个 Phase 执行完毕后，运行 `go test -v -race ./godeployer/新包名` 进行单元级别验证。
- 在 Phase 5 组装完毕后，在项目根目录执行完整测试：`go test -v -race ./...`，确保没有任何的 Data Race 和编译错误。
- (可选) 前端 e2e 测试验证：`cd web && npm run test:e2e`。
