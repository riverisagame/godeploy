# 纳米级计划：SSH连接池重构 (M1-Infra)

## 目标
解决单条 SSH 隧道复用带来的并发性能瓶颈与连接阻塞隐患，实现可限制最大并发数的 SSH 连接池。

## 原子化任务拆解

### 1. [RED] 编写测试与 Benchmark (TDD 驱动)
- **文件**: `godeployer/ssh_test.go`
- **动作**: 
  - 新增 `TestSSHPool_AcquireRelease` 测试：验证池大小限制和上下文超时获取机制。
  - 新增 `BenchmarkSSHPool_Concurrent` 压测：起 100 个 goroutine 并发执行 `RunCommand`。
- **行数**: ~60 行

### 2. [GREEN] 实现 SSHPool 数据结构与管理
- **文件**: `godeployer/ssh_pool.go` [NEW]
- **动作**:
  - 定义 `SSHPool` 结构体：包含 `ServerConfig`、`chan *ssh.Client`（空闲池）、`active` 计数、`maxConns`。
  - 实现 `NewSSHPool(server ServerConfig, maxConns int)`。
  - 实现 `Get(ctx) (*ssh.Client, error)`：若空闲池有则取；若 `active < maxConns` 则新建；否则阻塞等待直到 ctx 超时。
  - 实现 `Put(client, err)`：若无错放入空闲池，若出错则 Close 并减小 active 计数。
  - 实现 `Close()` 销毁所有连接。
- **行数**: ~80 行

### 3. [LINKING] 改造 SSHExecutor 对接连接池
- **文件**: `godeployer/ssh.go`
- **动作**:
  - `SSHExecutor` 结构体内部改为持有 `*SSHPool`，而非单一 `*ssh.Client`。
  - `NewSSHExecutor` 函数参数增加 `pool *SSHPool`。
  - `RunCommand` 函数内部改为：`client, err := pool.Get(s.Ctx)`，在 `defer pool.Put(client, err)` 中释放。
- **行数**: ~20 行变更

### 4. [LINKING] 全局引擎适配
- **文件**: `godeployer/engine.go`
- **动作**:
  - `DeployEngine` 增加字段 `sshPools map[string]*SSHPool` (按 Server IP/Host 区分)。
  - `RunDeploy` 与测试模块中动态实例化 `SSHExecutor` 的地方，改为从 engine 的池集合中提取 pool，或在初始化配置时统一步骤注册 pool。
- **行数**: ~30 行变更

## 物理无损与零污染策略
1. 测试全部基于 mock/内建 local test sshd 或使用既有的 `test-server` 镜像进行。
2. 连接池管理对执行命令的 payload 透明，不影响现有的任务编排逻辑。
