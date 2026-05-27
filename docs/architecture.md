# GoDeployer 核心架构解析

GoDeployer 是一个极简、高性能的分布式部署系统，旨在替代传统的重度部署工具（如 Jenkins, Ansible, DeployerPHP）。它由 Go 语言从零编写，前端集成 Vue 3 进行单文件分发。本文档详细阐述其核心的协程池与任务调度架构。

## 1. 架构概览

GoDeployer 的核心可以划分为以下几个关键子系统：
- **Web API & Webhooks**: 提供 HTTP 接口，接收发布请求与 CI 触发。
- **调度引擎 (DeployEngine)**: 管理正在运行的部署任务（Task），处理多节点并发部署的生命周期。
- **状态机存储 (SQLite DB)**: 通过 Go 内存 SQLite (file::memory:?cache=shared) 实现零污染的任务状态流转。
- **SSH 协程池 (SSHPool)**: 高性能的 SSH 连接复用与管理机制。
- **实时通知总线 (Notifier)**: 通过 WebSocket 或 SSE 向前端推送部署实时日志与进度。

## 2. 并发调度机制

### 2.1 任务分发与隔离
每一个 Webhook 触发的部署请求都会产生一个独立的 `DeployTask`。
`DeployEngine` 通过一张 `running map[int64]*DeployTask` 来记录正在进行的任务，并为每个任务分配独立的全局 Context。如果用户取消任务，引擎将直接调用 `cancel()` 函数，终止该任务下所有的子协程和网络 I/O。

### 2.2 跨节点执行策略 (Multi-Node Concurrency)
为了保证多个服务器的部署一致性，GoDeployer 在进入 Phase 1（Rsync 文件同步）和 Phase 2（软链接切换）时，利用 `sync.WaitGroup` 针对集群中的所有节点派发 Goroutine 进行并发操作。

- **快速失败 (Fail-Fast)**: 只要有任何一个节点在关键生命周期（如 Rsync）失败，整体 Context 将被取消，防止集群陷入版本不一致的“脑裂”状态。
- **两阶段提交模型**: 详见 `deployment_flow.md`。

## 3. SSHPool 连接复用设计

由于大规模分布式系统部署频繁创建 SSH 连接会导致极其严重的性能损耗与句柄泄露，GoDeployer 实现了轻量级的 `SSHPool` 资源池：

```go
type SSHPool struct {
    clients chan *ssh.Client
    address string
    config  *ssh.ClientConfig
}
```

- **获取连接 (Acquire)**: 优先从 `clients` 通道中无阻塞获取空闲连接。如果没有空闲且未达最大并发上限，则动态创建新的 TCP/SSH 会话。
- **释放连接 (Release)**: 执行完毕后，将健康存活的 SSH Client 归还至通道，供后续的 `Rsync` 或 `Command` 复用。如果通道已满或连接已断开，则静默销毁。
- **节点隔离**: `DeployEngine` 维护了 `pools map[string]*SSHPool`（键名为 `Host:Port`），保证不同服务器实例互不干扰。

## 4. 零侵入内存数据库

为了满足**“物理零污染与 DDL 绝对禁绝”**的硬核设计规范，系统核心的审计日志与状态持久化默认使用 `go-sqlite3` 驱动的内存级 SQLite 实例。
- 连接字符串：`file::memory:?cache=shared`。
- 在服务启动时自动加载结构，并在关闭时随内存释放。
- 如需物理落盘记录，可通过 `config.yaml` 中配置 `data_dir` 映射独立文件。

## 5. WebUI 的静态嵌入

前端界面（Vue/React）在流水线中预先通过 Vite `npm run build` 打包。Go 语言使用 `//go:embed dist` 语法将打包后的静态资产直接烧录进二进制文件中。
这意味着在最终交付时，只需分发唯一的 `godeployer` 二进制文件，不再需要依赖 Nginx 或外部静态资源服务器。
