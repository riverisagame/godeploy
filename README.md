# GoDeploy (godeploy)

## 项目简介
GoDeploy 是一款轻量级、无侵入、并由 AI 全程参与设计的代码发布与自动化部署系统。项目采用 Go (1.25+) 开发后端 API，采用 Vue3/TypeScript 开发前端控制台。作为 PHP Deployer 的现代平替，它保留了基于 SSH 软链接原子发布的优雅逻辑，同时增加了基于 Web 的可视化多节点发布能力、完善的 RBAC 角色权限、发布防脑裂回滚支持、及实时 WebSocket 部署日志查看。

## 功能大全
- **🔐 多维度安全与 RBAC**: 详尽的 Admin、Maintainer、Viewer 多层级角色体系；强劲的 GitHub Webhooks 签名保护。
- **🚀 多节点并发发布**: 支持一次任务并发推送部署到集群中的多台机器。
- **🔄 原子化部署**: 采用 SSH 目录切换策略 (Current -> Releases) 与原子软连接 (`ln -sfn`) 确保平滑上线，故障无感知。
- **⏪ 防脑裂回滚机制 (Anti-Brain Split Rollback)**: 任何一台节点在切换软链接时失败，将自动下达全集群的回滚指令至安全点。
- **📡 实时 WebSocket 部署日志**: 前端接入 WebSocket，无缓冲实时查看每一步 shell 脚本执行细节与标准输出。
- **📦 内置调度缓冲池 (Job Queue)**: 避免大规模集中提交时耗尽资源，提供平滑排队部署。
- **⚙️ 强校验 CI/CD**: 严苛的 GitHub Action 持续集成集成：每次 Push 自动触发代码规范检查、单元测试执行与带有严密上下文生命周期的版本 Release 制品发布。

## 功能原理
1. **任务下发**: 用户/Webhook 通过 REST API 提交一个 Git Commit，该任务进入 `deploy_tasks` 数据库表。
2. **状态机与调度**: `DeployEngine` 监听 `pending` 任务，并放入具有并发数限制的协程池处理。
3. **环境隔离克隆**: 在本机为每个任务开辟独立的构建环境，从远端拉取 Git 源码，运行构建钩子 (Hook)。
4. **Rsync 分发 (Phase 1)**: 将构建产物通过 Rsync 增量分发到各目标节点的 `releases/<TIMESTAMP>` 目录中，此时并不影响线上业务服务。
5. **原子切换 (Phase 2)**: 当所有节点的 Phase 1 执行完成后，一次性并发执行 `ln -sfn` 将所有节点的 `current` 指向最新的 `releases/<TIMESTAMP>`。
6. **成功与通知**: 归档更新任务状态并触发对应平台（飞书/钉钉等）的 Webhook 消息流向用户。

## 如何安装

### 前提依赖
- [Go 1.25+](https://golang.org/) (由于核心依赖和 CI 约定，需要 Go 1.25 或以上)
- [Node.js 20+](https://nodejs.org/) (推荐，用于前端构建，版本>= 20)
- SQLite3 驱动内建（无额外依赖）
- 部署目标主机需开启 SSH 访问，并安装有 `rsync`

### 1. 下载源码与构建后端
```bash
git clone https://github.com/riverisagame/godeploy.git
cd godeploy
go mod download
go build -o godeployer_bin ./godeployer
```

### 2. 构建前端
```bash
cd web
npm install
npm run build
```
*(注：前端的构建产物最终放置在后端的静态代理服务目录下，跟随主服务运行)*

## 如何配置

服务启动前，您需要配置好 `config.yaml` 文件（可参考 `config.example.yaml`）。
以下是关键配置项说明：

```yaml
server:
  port: 8080                    # 后端 API 端口
  auth_secret: "your_jwt_secret" # 用于生成和签发 JWT Token 的密钥

database:
  driver: "sqlite"              # 当前版本固定为 sqlite
  dsn: "deploy.db"              # SQLite 物理文件路径

projects:                       # 配置你的目标项目
  - id: "my-app"
    name: "My App"
    repo_url: "git@github.com:user/my-app.git"
    environments:
      - id: "production"
        name: "Production"
        nodes:
          - host: "10.0.0.1"    # 目标服务器 1
            port: 22
            user: "deploy"
            deploy_path: "/var/www/my-app"
          - host: "10.0.0.2"    # 目标服务器 2
            port: 22
            user: "deploy"
            deploy_path: "/var/www/my-app"
```

编写好配置后，启动主服务：
```bash
./godeployer_bin --config=config.yaml
```

控制台此时将在 `http://localhost:8080/` 就绪。如为首个管理员账户，可在启动后通过工具生成初始账号密码及 Token。
