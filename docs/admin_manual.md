# GoDeployer 系统运维与管理手册

本手册针对系统管理员，涵盖 GoDeployer 的服务安装、用户权限控制机制（RBAC）以及持续集成流水线（Webhook）的对接指南。

## 1. 安装与服务管理

通过提供的一键安装脚本，您可以迅速在 Linux 系统（Ubuntu/Debian, CentOS/Rocky 等）上部署该系统。

### 安装命令
```bash
wget https://raw.githubusercontent.com/riverisagame/godeploy/master/scripts/install.sh
sudo bash install.sh
```

### 目录约定
- **二进制文件**: `/usr/local/bin/godeployer`
- **配置文件**: `/etc/godeployer/config.yaml`
- **应用数据目录**: `/var/lib/godeployer/`
- **日志排查**: `journalctl -fu godeployer.service`

## 2. 用户 RBAC 权限控制

系统通过 `db.go` 中内置的 SQLite 表 `users` 与 `user_roles` 进行极简的权限认证管理，主要将用户身份划分为三类：

1. **Guest (访客)**: 无任何权限，通常用于只读展示监控页面。
2. **Deployer (部署员)**: 可以浏览项目状态，拥有执行某指定项目环境部署（`HandleCreateTask`）的权利，但无权修改全局配置。
3. **Admin (管理员)**: 掌握系统的绝对控制权，可以配置 `Project` / `Server`，查阅所有的详细系统日志，添加或删除其他用户。

**修改配置**: 管理员可以在 WebUI 的 Dashboard 界面中修改用户权限，或者直接修改 `/etc/godeployer/config.yaml` 结合本地 SQLite 实现紧急账户提权。

## 3. Webhook 自动化流水线集成

为实现极致的 DevOps 自动化，GoDeployer 原生暴露了 RESTful Webhook 接口，可以直接对接 GitHub, GitLab, 或 Bitbucket 的 Push Events。

### 触发接口
`POST /api/webhook/deploy`

**Request Payload (JSON)**:
```json
{
  "token": "你的_Webhook_安全_Token",
  "project_id": "proj-multi",
  "env_id": "env-prod",
  "commit_id": "master"
}
```

### 最佳实践 (GitHub Actions 联动)

在你的业务代码仓库中（如 Vue/Laravel 项目），配置以下 `.github/workflows/deploy.yml` 即可实现 Push `master` 时自动触发发布：

```yaml
name: Trigger Deployment

on:
  push:
    branches:
      - master

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger GoDeployer Webhook
        run: |
          curl -X POST https://your-deployer-server.com/api/webhook/deploy \
            -H "Content-Type: application/json" \
            -d '{
              "token": "${{ secrets.DEPLOYER_WEBHOOK_TOKEN }}",
              "project_id": "my-backend",
              "env_id": "production",
              "commit_id": "${{ github.sha }}"
            }'
```

这样即达成了从“代码合并”到“生产服务器毫秒级切换”的全自动黑盒闭环。
