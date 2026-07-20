# P0 & P1 漏洞修复与系统重构纳米级计划 (IR阶段)

**日期**: 2026-07-20

根据您的决策（修复范围 P0+P1，完整 JWT 用户认证，命令注入白名单模板），以下是严格按照 10-20 行代码原子化改动原则制定的执行计划。

## 🔴 Phase 1: P0 致命缺陷修复 (Security & Concurrency)

### 任务 1.1: 修复部署记录 EnvID 传参错误 (P0-1)
- **目标文件**: `internal/interfaces/api/deploy_handler.go`
- **修改逻辑**:
  - `StartDeploy` 方法 L68，将 `h.svc.TriggerDeploy(req.ProjectID, 1, req.CommitHash)` 修改为 `h.svc.TriggerDeploy(env.ID, 1, req.CommitHash)`。

### 任务 1.2: 修复部署日志并发写入 Data Race (P0-2)
- **目标文件**: `internal/application/deploy_engine.go`
- **修改逻辑**:
  - 提取一个内部结构 `type deployLogger struct { mu sync.Mutex; lines []string; engine *DeployEngine; depID uint }`。
  - 实现 `appendLog(msg string)`，在内部加锁保护 `lines = append(lines, msg)`，并调用 `Publish`。
  - 修改 `StartDeploy` 和 `Rollback`，使用 `deployLogger` 替代闭包和裸 slice。

### 任务 1.3: 修复 SSE 早期日志丢失 (P0-5)
- **目标文件**: `internal/application/deploy_engine.go`
- **修改逻辑**:
  - `DeployEngine` 增加字段 `logHistory map[uint][]string` 及其互斥锁 `historyMu sync.RWMutex`。
  - 在 `appendLog` 时将日志写入 `logHistory`。
  - 在 `Subscribe(deploymentID)` 方法中，先获取锁，将该 deploymentID 对应的所有历史日志从 `logHistory` 拷贝出来，直接写入新创建的 `chan string` 中，然后再返回 channel。
  - `CloseSubscribers` 时从 `logHistory` 删除对应记录。

### 任务 1.4: 完整用户认证体系 - 领域与数据层 (P0-3)
- **目标文件 1**: `internal/domain/user.go` (新建)
  - 定义 `User` struct (ID, Username, PasswordHash)。
  - 定义 `UserRepository` interface。
- **目标文件 2**: `internal/infrastructure/persistence/user_repository.go` (新建)
  - 实现基于 GORM 的 `UserModel` 和 `SqliteUserRepository`。
- **目标文件 3**: `internal/application/auth_service.go` (新建)
  - 实现 `Login(username, password)` 返回 JWT token。
  - 引入 `golang-jwt/jwt/v5` 库和 `bcrypt`。

### 任务 1.5: 完整用户认证体系 - 接口层中间件 (P0-3)
- **目标文件**: `internal/interfaces/api/router.go` 和 `middleware.go` (新建)
  - 在 `middleware.go` 实现 `AuthMiddleware(next http.Handler)`，校验 Authorization Bearer JWT。
  - 在 `router.go` 注册 `POST /api/login` (免认证)。
  - 用 `AuthMiddleware` 包装所有 `/api/projects`, `/api/servers`, `/api/deployments` 路由。
  - 修改 `TriggerDeploy` 将 UserID 从 1 修改为从 `r.Context()` 中获取的实际 UserID。

### 任务 1.6: 消除命令注入 - 白名单与模板化命令 (P0-4)
- **目标文件**: `internal/domain/environment.go` & `deploy_engine.go`
  - 将 `EnvironmentModel` 的 `PreDeploy` / `PostDeploy` 重新定义，不再允许输入任意 bash 命令。
  - 在应用层定义 `CommandTemplates` 映射，如 `npm_build` -> `npm install && npm run build`。
  - 前端修改，将原先的自由文本输入框改为下拉选择组件（"无", "Node.js Build", "Go Build", "清理缓存"）。
  - 后端在 `StartDeploy` 执行 `PreDeploy` 时，根据传入的模板 ID 在安全沙箱或严格参数化的上下文中拼接。

---

## 🟠 Phase 2: P1 架构规范与数据修复 (Architecture & Data)

### 任务 2.1: 解除 Handler 对 Repo 的直接依赖 (P1-1)
- **目标文件**: `internal/application/server_service.go` (新建)
  - 封装 `GetServers` 和 `CreateServer`。
- **目标文件**: `internal/interfaces/api/server_handler.go`
  - 依赖替换为 `ServerService`，移除 `domain.ServerRepository`。

### 任务 2.2: 消除 DeployEngine 依赖倒置违规 (P1-2)
- **目标文件**: `internal/application/deploy_engine.go`
  - 定义接口 `type DeployStatusUpdater interface { CompleteDeploy(...) }`。
  - `DeployEngine` 依赖该接口，解除对 `*DeployService` 的具体结构体依赖。

### 任务 2.3: 为 Rollback 增加部署锁 (P1-3)
- **目标文件**: `internal/application/deploy_engine.go`
  - 在 `Rollback` 顶部复用 `e.getDeployLock(envKey)` 逻辑，防止正在部署时被回滚或同时触发两次回滚。

### 任务 2.4: 数据库索引与级联补全 (P1-4, P1-5, P1-6)
- **目标文件**: `internal/infrastructure/persistence/sqlite.go` & `deployment_repository.go`
  - `EnvironmentModel` 的 gorm 标签补充 `gorm:"uniqueIndex:idx_project_env"`，实现 `(ProjectID, Name)` 唯一索引。
  - `DeploymentModel` 增加 `gorm:"index"` 到 `EnvID`。
  - 修复 `TriggerDeploy` 中 `deployment` 创建时 `ProjectID` 赋值遗漏的问题（目前为 0）。

### 任务 2.5: 移除配置硬编码 (P1-7)
- **目标文件**: `cmd/pdeploy/main.go` & `internal/config/config.go` (新建)
  - 实现从环境变量读取 `PORT`, `DB_PATH`, `WORKSPACE_DIR`, `JWT_SECRET`，移除所有的硬编码魔法字符串。

### 任务 2.6: 处理被忽略的 Errors (P1-13)
- **目标文件**: `sqlite.go` L52, L59 等
  - `json.Unmarshal` 的错误检查不应被 `_` 覆盖，需要记录日志或降级处理。

---

## 🟡 Phase 3: P1 前端深度治理 (Frontend Refactoring)

### 任务 3.1: 拆分 God Component (P1-8)
- **目标文件**: `web/src/views/EnvironmentConfig.vue`
  - 将 Server 穿梭框抽取为 `ServerSelector.vue`。
  - 将环境变量列表抽取为 `EnvVarEditor.vue`。

### 任务 3.2: 引入 TypeScript Type 声明 (P1-9)
- **目标文件**: `web/src/types/index.ts` (新建)
  - 建立 `Project`, `Environment`, `Server`, `Deployment` 的严格 TS 接口，替换所有 `any`。

### 任务 3.3: 封装 Axios 与 API 层 (P1-10)
- **目标文件**: `web/src/api/request.ts` (新建)
  - 创建 axios 实例，全局拦截器处理 401 自动跳转 login，处理 500 统一抛出 Toast。

### 任务 3.4: 修复基础交互 BUG (P1-11, P1-12)
- **目标文件**: `web/src/views/Deployments.vue` (新建), `web/src/views/ServerList.vue`
  - 实现侧边栏的 `/deployments` 全局部署记录查看页。
  - 绑定 `ServerList.vue` 的删除按钮事件调用 API。
