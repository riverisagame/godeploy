# 架构决策与需求对齐 (Milestone 2: RBAC & Webhooks)

## 1. 需求背景
随着系统的扩展，我们需要：
1. **细粒度权限控制 (RBAC)**：区分管理员、部署操作员、只读观察者，防止越权部署或误删数据。
2. **自动化流水线触发 (Webhooks)**：当代码仓库有 Push 操作时，自动调用系统接口发起部署。

## 2. 硬核设计与自我攻击 (Adversarial Audit)

### 2.1 RBAC 权限系统设计
- **方案**：在 `users` 表中新增 `role` 字段。定义三种角色：`admin`, `deployer`, `viewer`。
- **JWT 扩展**：将 `role` 写入 JWT Claim 中，并在 `auth.go` 实现 `RoleMiddleware(allowedRoles ...string)` 供各个路由挂载。
- **物理攻击对冲 (Schema Migration)**：不能删除现有的 `users` 表！必须编写无损的 DDL：`ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'viewer'`，并紧接着执行 `UPDATE users SET role = 'admin' WHERE username = 'admin'`。这样绝对保证历史数据零污染。

### 2.2 Webhooks 触发器设计
- **方案**：增加 `POST /api/webhooks/github` 接口。
- **安全拦截**：强制在请求头校验 `X-Hub-Signature-256`。密钥 (Webhook Secret) 配置在 `ProjectConfig` 结构体中。
- **性能对冲 (Thundering Herd Problem)**：如果短时间内爆发大量 Git Push 怎么办？
  - *自我攻击*：如果排队（Queue），会导致积压和陈旧版本的连续部署，浪费资源。
  - *决策*：秉承最简架构，复用已有的“部署锁”逻辑。如果当前项目有任务处于 `deploying`，Webhook 请求直接返回 `409 Conflict`。Github 会记录这次失败，但这是预期内的**去抖 (Debounce)** 行为，始终保持部署最新且唯一的版本。

### 2.3 影响范围评估
1. `db.go`: 需增加无损 `ALTER TABLE` 迁移逻辑。
2. `config.go`: 需在 `ProjectConfig` 增加 `WebhookSecret` 与 `Branch` 字段。
3. `api.go`: 新增 Webhook 处理控制器，修改现有路由加上 `RoleMiddleware` 拦截器。

---
**[出口准则]**：用户确认此架构决策后，输出 `[SCAN_COMPLETE]`，进入纳米级计划制定阶段。
