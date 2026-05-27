# ADR-002: 多项目多环境配置驱动及扩展特性设计 (v2)

## 1. 需求背景
用户需要支持“多项目加多环境”的部署能力，采用配置文件（YAML）的形式声明。此外，根据进一步对齐：
- **前端配置管理**：采用“配置即代码”理念，前端仅作“只读展示”，不允许在线修改。
- **敏感凭证与动态值**：支持在 YAML 中使用 `${ENV_VAR}` 占位符，由 Go 后端在启动加载时从环境变量中动态读取替换。
- **多用户审计**：引入用户登录机制，在部署记录中清晰区分不同用户提交的部署任务。
- **事件通知机制**：预留部署事件通知接口，以便未来扩展（如 Webhook、邮件等通知）。

## 2. 方案选型与架构定义

### 2.1 配置即代码与环境变量
- 采用 YAML 格式配置项目和环境。
- Go 后端解析 YAML 时，采用自定义过滤器或 `os.ExpandEnv` 进行环境变量的动态替换。例如：
  `jwt_secret: "${JWT_SECRET}"`
  `ssh_key_path: "${HOME}/.ssh/id_rsa"`

### 2.2 用户鉴权与部署审计
- **表结构设计**：
  - `users` 表：存储 `id`, `username`, `password_hash`（使用 bcrypt 算法加密）, `role` (如 admin, deployer), `created_at`。
  - `deploy_tasks` 表：存储 `id`, `project_id`, `env_id`, `commit_id`, `status`, `release_name`, `user_id` (创建者ID), `username` (创建者用户名), `config_snapshot` (配置快照), `created_at`。
- **验证方式**：基于 JWT 的 Stateless 认证，前端在 Request Header 中携带 `Authorization: Bearer <token>`。

### 2.3 通知系统设计 (事件驱动)
- 定义 `DeployEvent` 结构体，包含：
  - 任务 ID
  - 项目名称
  - 环境名称
  - 发起人 (User)
  - 触发动作 (部署/回滚)
  - 当前状态 (开始/成功/失败)
  - Commit 详情 (Hash, Author, Message)
- 预留通用通知接口 `notifier.Notifier`：
  ```go
  type EventType string
  const (
      EventDeployStart   EventType = "deploy:start"
      EventDeploySuccess EventType = "deploy:success"
      EventDeployFailed  EventType = "deploy:failed"
  )

  type DeployEvent struct {
      Type      EventType
      TaskId    int64
      Project   string
      Env       string
      Operator  string
      Commit    string
      Timestamp int64
  }

  type Notifier interface {
      Send(event *DeployEvent) error
  }
  ```
- 默认实现：`LogNotifier`（仅将通知事件输出到系统日志）和 `WebhookNotifier`（预留发送给外部 URL 的接口）。

---

## 3. 详细配置文件结构设计 (`config.yaml`)

```yaml
global:
  sqlite_path: "./data/godeployer.db"
  log_path: "./data/logs"
  workspace_path: "./data/workspace"
  ssh_key_path: "${HOME}/.ssh/id_rsa"
  server_port: 8080
  jwt_secret: "${JWT_SECRET}"

# 自动扫描项目配置的目录
project_config_dir: "./projects.d"

# 预留通知配置接口
notification:
  enabled: true
  providers:
    webhook:
      url: "${NOTIFY_WEBHOOK_URL}"
      enabled: false
```

---

## 4. 核心流程与三路思考

### 4.1 实现路径 (Implementation)
1. Go 启动时首先加载并替换环境变量，将系统核心配置载入内存。
2. 扫描 `projects.d` 目录下的所有 `.yaml`，完成解析并同样执行环境变量替换。
3. 数据库初始化，如果 `users` 表为空，默认创建一个 `admin` 用户（密码通过环境变量 `${ADMIN_PASSWORD}` 或默认值生成，并提示于终端）。
4. 部署时，Go 从 JWT Token 中提取当前发起部署的用户信息，写入 `deploy_tasks` 表中的 `user_id` 和 `username`。
5. 任务的每一步状态变更（开始、成功、失败），触发事件队列，调用所有已注册的 `Notifier` 执行通知。

### 4.2 自我攻击与安全对冲 (Adversarial & Security Audit)
- **非授权部署**：恶意用户绕过前端直接调用 `/api/deploy` 接口。
  - *对冲策略*：所有敏感 API 均需通过 JWT 中间件验证，且在执行部署前校验该用户角色是否有权限对该项目和环境执行写操作（可在配置中增加用户组权限控制）。
- **环境变量缺失**：如果配置文件中使用了 `${NOTIFY_WEBHOOK_URL}` 但未在宿主机上定义该环境变量，可能导致系统崩溃或静默失败。
  - *对冲策略*：当解析到未定义的环境变量时，记录 Warning 日志，如果是关键变量（如数据库路径、JWT 秘钥），启动时立即 Panic 并拒绝启动，确保系统不带着隐患运行。

### 4.3 性能对冲 (Performance & Concurrency Audit)
- **并发请求通知**：大量部署同时结束时，外部通知（如 Webhook）由于网络延迟，如果同步调用会严重阻塞 Go 的部署主流程协程。
  - *对冲策略*：通知发送必须**异步执行**。引入内部 Channel 缓冲队列，由独立的 Goroutine 消费队列并异步调用外部接口。限制并发发送数量并设置超时时间（如 5 秒），防止因第三方通知接口缓慢而挂起整个部署系统。

---

## 5. 待确认问题
无。已将用户对前端只读、环境变量、多用户登录与通知接口的需求全面对齐。
