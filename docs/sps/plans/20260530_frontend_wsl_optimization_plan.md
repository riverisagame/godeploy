# 解决 WSL 下启动极慢及界面功能缺失的修复计划

## 问题背景
目前 `demo.sh` 在 WSL Debian 中启动时存在两个致命问题：
1. **启动速度极慢**：在 WSL 中直接访问 `/mnt/d` (Windows NTFS) 进行 `go build` 和 `npm install` 时，由于跨 OS 文件系统 (9P 协议) 的 IO 瓶颈，导致启动需要数分钟甚至由于超时而失败。
2. **界面功能缺失（找不到历史记录、diff等）**：`demo.sh` 目前如果检测到 `web/dist` 存在，就会跳过前端构建。导致之前旧版本的前端产物一直被复用，新的 UI 组件（`DeployHistoryTable`, `DeployForm`）根本没有被编译和嵌入进后端二进制中。

## Proposed Changes

### [MODIFY] scripts/demo.sh
优化 `start_backend()` 函数，通过将缓存和构建目录转移到 WSL 原生的 `/tmp` 目录（ext4 文件系统），彻底绕过跨系统 IO 瓶颈，实现极速构建。
- **前端极速构建**：如果检测到是 WSL 环境，则自动将 `web` 目录拷贝到 `/tmp/godeployer_web_build`，在原生文件系统中执行 `npm install && npm run build`，完成后将产物拷贝回 `web/dist`。这能将前端构建速度提升十倍以上，并保证嵌入最新的 UI 代码。
- **后端极速构建**：如果检测到是 WSL 环境，注入 `GOMODCACHE="/tmp/gopath_demo/pkg/mod"` 和 `GOCACHE="/tmp/gopath_demo/cache"`，并将 `GOPROXY` 设为国内镜像，使 Go 依赖下载和编译都在原生文件系统中进行。
- **依赖追踪清理**：执行前通过判断 `web/dist/index.html` 的时间戳或者强制执行来保证编译产物是最新的。

## Verification Plan
1. 执行 `demo.sh start` 并观察在 WSL Debian 中是否能在 5-10 秒内启动。
2. 打开 `http://localhost:8080`，确认前端页面上展示了“历史记录”、“Diff”以及相关组件。
