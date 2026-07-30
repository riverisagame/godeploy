# 选型决策：Docker, K8s 与 Helm 部署架构

## 需求分析与疑问
用户请求“生成docker k8s helm”配置文件。由于 `godeploy` 是一个持续部署工具，其内部实现严重依赖于系统级的二进制命令。

- **Git 依赖**: `internal/infrastructure/git/client.go` 显式调用了系统的 `git` 命令执行 clone、worktree 等操作。
- **SSH/Sync 依赖**: `internal/infrastructure/ssh/client.go` 显式调用了系统的 `rsync`（或 `scp`）来进行文件增量同步。
- **数据库依赖**: `internal/infrastructure/persistence/` 使用了 `github.com/glebarez/sqlite`，这是 CGO-Free 的纯 Go SQLite 驱动。

### 架构选型与对冲审计
1. **Dockerfile 选型**
   - **基础镜像**: 编译期使用 `golang:1.26-alpine`，运行期使用 `alpine:latest`。
   - **系统依赖**: 必须在运行期镜像中 `apk add --no-cache git rsync openssh-client`。
   - **编译参数**: `CGO_ENABLED=0 GOOS=linux`，因为 SQLite 驱动是纯 Go 的，关闭 CGO 提升可移植性且免受 libc 动态链接干扰。
   - **性能与安全对冲**: Alpine 镜像极小，关闭 CGO 确保安全无外部注入。非 root 用户运行？由于涉及 SSH 密钥和挂载，可以通过 PUID/PGID 控制，但默认先简化。
2. **K8s & PVC 规划**
   - 必须配置持久化存储 (PVC) 以防止 Pod 重启导致数据丢失。
   - 挂载点 1: 数据库文件路径 (由 `DB_PATH` 环境变量指定，默认为 `/app/data/pdeploy.db`)。
   - 挂载点 2: Git 仓库工作区 (由 `WORKSPACE_DIR` 指定，默认为 `/app/workspace`)。
   - 挂载点 3: (可选) SSH 密钥存储 (或者作为 Secret 挂载)，默认为 `~/.ssh`，这里我们在容器内统一定位到 `/app/data/.ssh`。
3. **Helm Chart 结构**
   - 经典的 `deployment.yaml`, `service.yaml`, `pvc.yaml`, `configmap.yaml`, `secret.yaml`。
   - 所有的配置如 `JWT_SECRET` 将暴露在 `values.yaml` 中，便于生产环境安全注入。

## 对现有系统的影响
- **零影响**: 仅添加部署配置文件 (`Dockerfile`, `k8s/`, `helm/`)，不修改任何一行 Go 业务代码，符合绝对只读和零污染准则。
- **并发与性能**: Helm Chart 中 Deployment 默认副本数必须为 1 (`replicas: 1`)，因为 SQLite 在容器网络存储上多读单写能力受限，且内部调度器通过单机内存锁控制并发。
