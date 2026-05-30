# ARCH-023: Demo Bootstrap Script Optimization (一键启动 Demo 脚本优化)

优化 `scripts/demo.sh`，以实现 WSL Debian 架构下的一键部署、秒级启动、无需拉取、稳定无错、快速预览。

## 选型背景与痛点 (Context & Pain Points)

当前 `scripts/demo.sh` 在执行一键完整初始化 (`all`) 时，有以下痛点：
1. **网络拉取阻塞与高错误率**：强行拉取 top-think/think、walkor/webman 以及 ZhongBangKeJi/CRMEB (约 187MB) 三个 Gitee 仓库，在国内或海外网络环境下极其容易发生超时、DNS 解析失败或半途断链。
2. **启动复杂度高**：后端运行在 8080，前端需要用户手动到 `web` 目录下执行 `npm install && npm run dev`。未实现真正的一键预览。
3. **依赖重**：在仅需要预览系统或者进行前后端连调时，不需要真正克隆上百兆的 PHP 项目历史。

## 设计方案决策 (Design Decisions)

### 1. 默认使用本地 Mock Git 仓库 (Local Mock Git Repositories by Default)
- **实现机制**：当启动 demo 时，默认不再克隆远程 Gitee。而是直接在 `demo_workspace/gitee_demo/{think,webman,CRMEB}` 下通过 `git init` 本地创建微型仓库，并写入 dummy 文件。
- **Commit ID 映射**：本地 Mock 仓库中的 commit 历史需包含并匹配 `scripts/demo.sh` 中 `seed_db` 插入的真实 commit ID（例如 `43983ad1d0b...` 等），使得系统的 Diff、发布任务和状态展示能 100% 正常工作。
- **保留真实克隆作为可选项**：通过 `bash scripts/demo.sh --real` 或参数指定，允许用户克隆真实 Gitee 仓库。
- **真实构建与部署约束**：Mock 仓库中必须包含模拟的构建与代码文件。每次触发部署或验证时，部署引擎能够真实地拉取本地 Mock 仓库中的代码、执行编译逻辑（如模拟 composer/npm 命令产出）、并通过 SSH/rsync 真正传输到目标路径，实现完整的、无损的端到端真实部署链条验证，拒绝只在数据库里伪造数据。

### 2. 自动前端打包与内嵌 (Auto Frontend Build & Embed Integration)
- **体验提升**：检查本地 `web/dist` 是否有资源，若没有，在有 `node`/`npm` 依赖的情况下，提供自动/快捷的 `npm run build` 打包。然后利用 Go 的 `go:embed` 直接编译进后端二进制。
- **单端口服务**：用户启动后，仅需访问 `http://localhost:8080` 即可预览完整的前后端应用，不需要单独开启 5173 端口。

### 3. 增强进程与端口管理 (Robust Process & Port Lifecycle Management)
- **端口清理**：在 `start` 阶段，若 8080 端口已被占用，提示用户或自动杀掉旧进程。
- **状态感知**：精准通过 PID 文件和端口侦听来监控启动过程，避免 `go build` 和 `nohup` 异步引发的未就绪报错。

## 现有功能影响评估 (Blast Radius)

- **影响范围**：仅限于 `scripts/demo.sh`，对后端核心业务逻辑（`godeployer` 目录下的 go 源码）以及前端源码无侵入性。
- **安全性**：100% 只读或只在 `demo_workspace`、`demo_deployer.db` 以及临时目录中进行操作，绝对无损于用户的工作区或真实开发数据。

## 决策人与状态 (Status)
- **Status**: `PROPOSED` (待用户 Review)
- **Date**: 2026-05-30
