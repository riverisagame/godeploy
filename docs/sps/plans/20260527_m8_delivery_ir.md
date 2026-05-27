# 纳米级计划：自动化发布与核心文档体系 (M8)

## 目标
建立基于 GitHub Actions 的 CD 流水线，并提供一键式系统安装与更新脚本；沉淀核心机制原理文档。

## 原子化任务拆解

### 1. [LINKING] 创建 Release 流水线
- **文件**: `.github/workflows/release.yml`
- **动作**: 编写基于 `tags` 的触发条件，编译静态内嵌 Vue 前端资源（`go:embed`）的 Go 二进制文件（Linux/Windows），并生成 GitHub Release。

### 2. [LINKING] 创建运维管理脚本
- **文件**: `scripts/install.sh`
- **动作**: 创建配置目录 `/etc/godeployer` 与数据目录 `/var/lib/godeployer`，下载或复制最新二进制文件，并生成 `systemd` 服务。
- **文件**: `scripts/update.sh`
- **动作**: 下载最新版本替换现存的二进制文件，重启 systemd 服务。

### 3. [LINKING] 撰写核心架构文档
- **文件**: `docs/architecture.md`
- **动作**: 阐释引擎与协程池架构、并发任务队列设计、内置 SQLite (Memory) 与状态机流转。
- **文件**: `docs/deployment_flow.md`
- **动作**: 阐释两阶段发布模型、同步机制与防止脑裂回滚。
- **文件**: `docs/admin_manual.md`
- **动作**: RBAC 权限划分模型、部署流水线的 Webhook 调用规约及系统运维指南。
