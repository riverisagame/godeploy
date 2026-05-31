# DDD 全量战术模式落地 — 设计规格

## 状态

已设计，待 Review 后进入实现计划阶段。

## 目标

将 GoDeploy 从"目录重命名式 DDD"改造为完整的战术 DDD 落地：充血模型、值对象、领域服务、依赖反转。全程纳米级 TDD，零功能破坏。

---

## 章节 1：Domain 层 — 值对象 + 实体方法

### 1.1 新增值对象

**新文件**: `godeployer/domain/value_objects.go`

```go
package domain

type DeployStatus string

const (
    StatusPending   DeployStatus = "pending"
    StatusDeploying DeployStatus = "deploying"
    StatusSuccess   DeployStatus = "success"
    StatusFailed    DeployStatus = "failed"
    StatusAborted   DeployStatus = "aborted"
)

func (s DeployStatus) Valid() bool {
    switch s {
    case StatusPending, StatusDeploying, StatusSuccess, StatusFailed, StatusAborted:
        return true
    }
    return false
}

func (s DeployStatus) IsTerminal() bool { return s == StatusSuccess || s == StatusFailed || s == StatusAborted }
func (s DeployStatus) IsRunnable() bool { return s == StatusPending }
```

### 1.2 实体行为注入：DeployTask

**修改文件**: `godeployer/domain/entity.go`

- L103: `Status string` → `Status DeployStatus`

**新增方法**（同文件末尾）：

```go
var ErrInvalidTransition = errors.New("deploy task: invalid status transition")

func (t *DeployTask) Start() error {
    if t.Status != StatusPending {
        return ErrInvalidTransition
    }
    t.Status = StatusDeploying
    return nil
}

func (t *DeployTask) Complete() error {
    if t.Status != StatusDeploying {
        return ErrInvalidTransition
    }
    t.Status = StatusSuccess
    return nil
}

func (t *DeployTask) Fail() error {
    if t.Status != StatusDeploying {
        return ErrInvalidTransition
    }
    t.Status = StatusFailed
    return nil
}

func (t *DeployTask) Abort() error {
    if t.Status != StatusPending && t.Status != StatusDeploying {
        return ErrInvalidTransition
    }
    t.Status = StatusAborted
    return nil
}

func (t *DeployTask) IsActive() bool {
    return t.Status == StatusPending || t.Status == StatusDeploying
}
```

### 1.3 实体行为注入：DeployJob

**修改文件**: `godeployer/domain/entity.go`

```go
func NewDeployJob(taskID int64, config *Config, logFilePath string) *DeployJob {
    ctx, cancel := context.WithCancel(context.Background())
    return &DeployJob{
        Ctx:         ctx,
        Cancel:      cancel,
        TaskID:      taskID,
        Config:      config,
        LogFilePath: logFilePath,
    }
}
```

### 1.4 补全 ProjectRepository

**修改文件**: `godeployer/domain/repository.go`

当前空接口，保留并补全为最小可用接口：

```go
type ProjectRepository interface {
    GetAllProjects(config *Config) []ProjectSummary
}

type ProjectSummary struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

### 改动半径

| 文件 | 操作 |
|------|------|
| `domain/value_objects.go` | 新建 |
| `domain/entity.go` | Status 字段类型 + DeployTask/DeployJob 方法 + ErrInvalidTransition |
| `domain/repository.go` | ProjectRepository 补全 |

---

## 章节 2：Domain 层 — 领域服务

### 2.1 NodeExecutor 接口（依赖反转）

**新文件**: `godeployer/domain/deployment_service.go`

```go
package domain

// NodeExecutor 定义部署节点操作接口。
// domain 层定义接口，infrastructure/ssh 层实现——依赖反转原则。
type NodeExecutor interface {
    Rsync(ctx context.Context, node ServerConfig, releaseName string) error
    SwitchSymlink(ctx context.Context, node ServerConfig, releaseName string) error
    RunCommand(ctx context.Context, node ServerConfig, cmd string) ([]byte, error)
}
```

### 2.2 DeploymentService

```go
type DeploymentService struct {
    taskRepo TaskRepository
}

func NewDeploymentService(taskRepo TaskRepository) *DeploymentService {
    return &DeploymentService{taskRepo: taskRepo}
}

// Execute 编排两阶段部署：并行 rsync → 统一切换软链。
// 纯业务规则，不包含 SSH/IO 细节。
func (s *DeploymentService) Execute(
    ctx context.Context,
    task *DeployTask,
    nodes []ServerConfig,
    executor NodeExecutor,
) error { /* 规则从 deploy_service.go RunDeploy 中迁移 */ }

// ShouldRollback 判断部署失败后是否需要回滚。
func (s *DeploymentService) ShouldRollback(task *DeployTask) bool { ... }
```

### 2.3 与 Application 层协作

`DeploymentService` 定义"做什么"（两阶段提交、Fail-Fast、回滚决策），Application 的 `DeployEngine.RunDeploy` 负责"怎么做"（日志写入、事件推送、协程管理）。`DeployEngine` 注入 `*DeploymentService`。

### 改动半径

| 文件 | 操作 |
|------|------|
| `domain/deployment_service.go` | 新建 |

---

## 章节 3：Application 层 — 拆分 DeployEngine

从 `application/deploy_service.go`（700行）拆分为 5 个文件。

### 3.1 拆分方案

| 文件 | 行数 | 职责 |
|------|------|------|
| `deploy_service.go` | ~200 | DeployEngine 结构体、SubmitJob、StartDispatcher、Close、RunDeploy |
| `deploy_executor.go` | ~150 | runLocalBuild、rsyncToServer、atomicDeploy |
| `rollback_service.go` | ~100 | RunRollback、RunRollbackToTask |
| `diff_service.go` | ~100 | generateTaskDiff、cacheTaskDiff |
| `process_utils.go` | ~50 | runCmd 跨平台进程管理 |

### 3.2 DeployEngine 改造

```go
type DeployEngine struct {
    taskRepo   domain.TaskRepository
    deploySvc  *domain.DeploymentService    // 新增：注入领域服务
    executor   domain.NodeExecutor           // ssh.RemoteExecutor → domain.NodeExecutor
    pools      map[string]*ssh.SSHPool       // SSH 连接池仍直接依赖基础设施
    queue      chan *domain.DeployJob
    running    map[int64]*domain.DeployJob
    mu         sync.Mutex
    eventBus   *notifier.EventBus
    wg         sync.WaitGroup
    activeWorkers int32
    maxWorkers int32
}
```

### 3.3 依赖关系

```
DeployEngine
├── domain.DeploymentService (纯业务规则)
├── domain.NodeExecutor (SSH操作——依赖反转)
├── domain.TaskRepository (持久化)
├── ssh.SSHPool (连接管理——仍直接依赖)
└── notifier.EventBus (事件推送)
```

### 改动半径

| 文件 | 操作 |
|------|------|
| `application/deploy_service.go` | DeployEngine 字段 + 构造函数 + RunDeploy 调用 deploySvc |
| `application/deploy_executor.go` | 新建 |
| `application/rollback_service.go` | 新建 |
| `application/diff_service.go` | 新建 |
| `application/process_utils.go` | 新建 |

---

## 章节 4：Interfaces 层 — 拆分 api.go

从 `interfaces/api/api.go`（1514行）按 REST 资源拆分为 10 个文件。

### 4.1 拆分方案

| 文件 | 内容 |
|------|------|
| `routes.go` | APIHandler 结构体、SetupRoutesWithExecutor、路由表 |
| `handler_auth.go` | HandleLogin、LoginRequest |
| `handler_deploy.go` | HandleCreateTask、HandleGetTasks、HandleGetTaskDetail、HandleGetTaskLog、HandleWSLog、TaskRes、CreateTaskRequest |
| `handler_rollback.go` | HandleTriggerRollback |
| `handler_diff.go` | HandleGetTaskDiff、HandleGetProjectPreviewDiff、HandleGetProjectRefs、HandleGetProjectCommits、diffSemaphore |
| `handler_webhook.go` | HandleGithubWebhook、ComputeGithubSignature、UpdateGitBindingRequest |
| `handler_user.go` | HandleGetUsers、HandleCreateUser、HandleUpdateUser、HandleDeleteUser、UserResponse、CreateUserRequest、UpdateUserRequest |
| `handler_user_git.go` | HandleGetUserGitBinding、HandleUpdateUserGitBinding |
| `handler_project.go` | HandleGetProjects、HandleUpdateUserPermissions、checkProjectAccess、UpdatePermissionsRequest |
| `handler_prune.go` | HandleSystemPrune |

### 4.2 APIHandler

```go
type APIHandler struct {
    Config    *domain.Config
    DB        *sql.DB
    TaskRepo  domain.TaskRepository
    Executor  domain.NodeExecutor    // ssh.RemoteExecutor → domain.NodeExecutor
    Engine    *application.DeployEngine
    GitCache  *git.Cache
}
```

### 4.3 测试文件

现有 8 个 `*_test.go` 不拆分，已按功能分文件。只适配字符串常量。

### 改动半径

| 文件 | 操作 |
|------|------|
| `interfaces/api/routes.go` | 新建 |
| `interfaces/api/handler_*.go` (8个) | 新建 |
| `interfaces/api/api.go` | 最终删除 |
| 所有 `*_test.go` | 字符串常量适配 |

---

## 章节 5：Infrastructure 层归位 + 适配层

### 5.1 config.go 归位

| 操作 | 文件 |
|------|------|
| 新建 | `infrastructure/config/loader.go` — 包名 `config` |
| 修改 | `godeployer/main.go` — `LoadConfig` → `config.LoadConfig` + import |
| 移动 | `config_test.go` → `infrastructure/config/loader_test.go` |
| 删除 | `godeployer/config.go` |

`application/` 和 `interfaces/api/` 不感知此变更（只消费 `*domain.Config` 指针）。

### 5.2 SSH 适配层

**新文件**: `godeployer/infrastructure/ssh/adapter.go`

```go
package ssh

// NodeAdapter 包装 *SSHPool，将底层 SSH 连接池能力适配为 domain.NodeExecutor。
// 每个方法内部执行 Acquire → 执行命令 → Release。
type NodeAdapter struct {
    pool *SSHPool
}

func NewNodeAdapter(pool *SSHPool) *NodeAdapter { ... }

func (a *NodeAdapter) Rsync(ctx context.Context, node domain.ServerConfig, releaseName string) error { ... }
func (a *NodeAdapter) SwitchSymlink(ctx context.Context, node domain.ServerConfig, releaseName string) error { ... }
func (a *NodeAdapter) RunCommand(ctx context.Context, node domain.ServerConfig, cmd string) ([]byte, error) { ... }
```

现有 `ssh.RemoteExecutor` 接口保留不变——两个接口是不同抽象层级，不互相替代。

### 5.3 依赖注入链

`main.go` 组合根最终形态：

```
config.LoadConfig(path)
  → db.InitGORM(...)
    → db.NewTaskRepository(gormDB)
    → db.NewUserRepository(gormDB)
  → ssh.NewSSHPool(...)
  → ssh.NewNodeAdapter(pool)
  → domain.NewDeploymentService(taskRepo)
  → notifier.NewEventBus()
  → application.NewDeployEngine(taskRepo, deploySvc, adapter, eventBus)
  → api.SetupRoutesWithExecutor(config, sqlDB, taskRepo, adapter, engine)
```

### 改动半径

| 文件 | 操作 |
|------|------|
| `infrastructure/config/loader.go` | 新建 |
| `infrastructure/config/loader_test.go` | 从根移入 |
| `infrastructure/ssh/adapter.go` | 新建 |
| `godeployer/main.go` | import + 依赖注入链改造 |
| `godeployer/config.go` | 删除 |
| `godeployer/config_test.go` | 删除 |

---

## 执行策略：纳米级 TDD

### Phase 执行顺序

| Phase | 名称 | 说明 |
|-------|------|------|
| 0 | **测试基线** | `go test -v -race ./...` 全部通过 |
| 1 | **值对象落地** | `value_objects.go` + `DeployStatus` 类型 + 全量适配 |
| 2 | **实体充血** | DeployTask/DeployJob 方法注入 |
| 3 | **领域服务** | `DeploymentService` + `NodeExecutor` 接口 |
| 4 | **SSH 适配层** | `NodeAdapter` 实现 |
| 5 | **Application 拆分** | 按文件切分 deploy_service，注入 deploySvc |
| 6 | **Interfaces 拆分** | api.go 按资源拆分 |
| 7 | **config.go 归位** | 移到 infrastructure/config |
| 8 | **全量回归** | `go test -v -race ./...` + `cd web && npm run test:e2e` |

### TDD 规则

每个 Phase 内：
1. 先写测试适配，不写业务代码
2. 单文件粒度提交，每改完一个文件立即 `go test` 验证
3. 任一 Phase 失败则停下分析，不继续后续 Phase

### 迭代策略

按层迭代（domain → infrastructure → application → interfaces），利用各层间单向依赖确保每次改动范围可控。

---

## 验收标准

1. `go test -v -race ./...` 全部通过，零 data race
2. `cd web && npm run test` 全部通过
3. `cd web && npm run test:e2e` 全部通过
4. `go build .` 编译通过
5. `godeployer/` 根目录仅保留 `main.go` 和 `main_test.go`
6. 状态字符串字面量仅出现在 `domain/value_objects.go` 常量定义中
7. `DeployTask.Status` 不再被外部直接写入，状态变更通过实体方法或 repository 完成
8. `domain.NodeExecutor` 接口定义在 domain 层，实现在 infrastructure 层
