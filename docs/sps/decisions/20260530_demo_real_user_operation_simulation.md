# ARCH-024: Real User Operation Simulation for Demo Data (通过真实接口模拟生成 Demo 数据)

为了遵循“数据必须由系统真实运行和操作产生，绝对不伪造任何数据”的强约束，重新设计 Demo 数据的注入和运行机制。

## 选型背景与痛点

原设计中，所有历史数据甚至时间线都是通过直接修改 SQL 字段或者直接插入伪造的。这违背了真实测试原则。

## 核心设计决策

### 1. 消除一切 SQL 直接注入与平移，坚持绝对真实 (No SQL Simulation or Translation)
- 绝对不在数据库中伪造数据，也不通过 SQL 去平移 `created_at` 或强行注入日志。
- 所有的 `deploy_tasks` 和 `deploy_logs` 100% 由真实运行的后端 API 写入。部署任务的时间戳就是当前真实运行的时间戳。

### 2. 多用户并发/交错调用 API (Multi-User API Call Simulation)
- 在脚本中依次用三个不同的角色登录获取 token：
  - 管理员账号 `admin` (`admin123`)
  - 部署员账号 `deployer` (`deploy123`)
  - 观察员账号 `viewer` (`view123`)
- 使用不同的 Token 向 `POST /api/tasks` 提交部署。
- 例如：
  - 用 `admin` 部署 `CRMEB`，结果：部署成功（或失败）。
  - 用 `deployer` 部署 `CRMEB`，结果：接口返回 `403 Forbidden` 并记录真实的拒绝历史。
  - 用 `deployer` 部署 `ThinkPHP`，结果：成功。
  - 这样，系统任务列表里显示的 username (如 `admin`, `deployer`) 100% 是通过 API 鉴权机制真实绑定并产生的，具有绝对的说服力。

### 3. 免 SSH 本地旁路机制 (2222 保留端口)
- 在后端的 `SSHExecutor` (位于 `godeployer/ssh.go`) 中加入本地旁路机制。当且仅当目标配置为 `host: "localhost"` 且端口为 `2222` 时，直接在本地以 `exec.Command` 执行 `sh -c` 以及本地的 `rsync` 拷贝，不走任何 SSH 加密通道。
- 将 `demo_projects.d/*.yaml` 中的环境服务器配置改为 `host: "localhost", port: 2222`。
- 这样，系统会真实地运行 `rsync` 复制本地 Mock 仓库代码、输出真实的部署日志和执行软链接切换。

## 现有功能影响评估 (Blast Radius)

- **影响范围**：修改 `godeployer/ssh.go` 中针对特定保留端口 `2222` 的判断。
- **安全性**：100% 安全，不会干扰普通项目的 SSH 连接。

## 决策人与状态 (Status)
- **Status**: `PROPOSED` (待用户 Review)
- **Date**: 2026-05-30
