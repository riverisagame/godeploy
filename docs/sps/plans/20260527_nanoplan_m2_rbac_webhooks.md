# Milestone 2: 核心业务接入层 (RBAC 与 Webhooks) Nanoplan

## 1. 目标
在绝对保证现有 SQLite 数据零污染、代码零回退的前提下，实现细粒度 RBAC 权限与基于防抖拦截的 Github Webhook 自动化部署。

## 2. 原子化执行链条 (10-20行约束)

### 步骤一：数据库平滑迁移 (RBAC Schema Migration)
- **文件**: `godeployer/db.go`, `godeployer/db_test.go`
- **动作**:
  1. **[RED]** 编写 `TestDB_Migration_Role`，断言旧版 `users` 表经过 `InitDB` 处理后，应包含 `role` 字段且 `admin` 用户具备 `admin` 权限。
  2. **[GREEN]** 在 `db.go` `InitDB` 中加入 DDL 补偿机制（使用 `ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'viewer'`）。捕捉 `duplicate column name` 错误以实现幂等。
  3. 执行无损数据修复：`UPDATE users SET role = 'admin' WHERE username = 'admin'`。

### 步骤二：JWT 令牌扩展与角色鉴权件 (Auth & Middleware)
- **文件**: `godeployer/auth.go`, `godeployer/api_test.go`
- **动作**:
  1. **[RED]** 扩展登录接口测试，验证 JWT 是否包含了 `role`，编写 `TestAuth_RoleMiddleware`。
  2. **[GREEN]** 将 `role string` 加入 `Claims`。
  3. 新增 `RoleMiddleware(allowedRoles ...string) gin.HandlerFunc`。利用 `c.Get("role")` 提取并判定。

### 步骤三：路由权限强绑定 (RBAC Routing)
- **文件**: `godeployer/api.go`
- **动作**:
  1. 挂载中间件：`GET /api/projects` 和 `GET /api/tasks` 开放给 `viewer`。
  2. `POST /api/tasks` 和 `POST /api/tasks/:id/rollback` 限制为 `admin` 或 `deployer`。
  
### 步骤四：Webhook 配置模型扩展
- **文件**: `godeployer/config.go`
- **动作**:
  1. **[GREEN]** 在 `ProjectConfig` 中增加 `WebhookSecret string` 和 `Branch string` 字段，解析 `config.yaml` 支持。

### 步骤五：Webhook 防抖控制器 (Thundering Herd Defense)
- **文件**: `godeployer/api.go`, `godeployer/api_test.go`
- **动作**:
  1. **[RED]** 编写 `TestAPI_GithubWebhook`，发送带签名校验的 Push Payload。
  2. **[GREEN]** 新增 `HandleGithubWebhook`。计算 `X-Hub-Signature-256` 并比对配置。
  3. 解析 `ref` 字段匹配分支。
  4. 复用已有的防抖查询，若存在 `deploying` 任务则强制返回 `409 Conflict`。
  5. 校验通过则隐式调用 `RunDeploy` 逻辑。

### 步骤六：全链路自动化验收
- **动作**: 运行 `go test -v ./...` 并存档日志。

---
**[出口准则]**：用户 Review 后，回复“继续”开始第一步：**数据库平滑迁移** 的 TDD 流程。
