# Milestone 1: 基础设施强化与性能对冲 (Nanoplan)

## 1. 目标与背景
当前 `SSHExecutor.RunCommand` 每执行一条指令（如 `setup`, `release`, `symlink` 等），都会发起一次完整的 TCP 握手、SSH 密钥验证并创建新的 `ssh.Client`。一次标准部署包含近 10 个远程命令，频繁的 SSH 握手极大拖慢了部署速度并增加了控制节点的端口消耗。

本计划将对其进行性能重构，并编写 Benchmark 压测。

## 2. 计划路径与原子步骤

### 步骤一：编写压测基准 (Red Phase)
- **文件**: `docs/sps/logs/benchmark_results.md` (记录基线)
- **文件**: `godeployer/ssh_test.go` [NEW]
- **变更**:
  1. 编写 `BenchmarkSSHExecutor_RunCommand` 压测函数。
  2. 运行压测并记录改造前（未复用连接）的基准耗时。

### 步骤二：SSHExecutor 连接池化改造 (Green Phase)
- **文件**: `godeployer/ssh.go`
- **变更 (10-20 行)**:
  1. 为 `SSHExecutor` 结构体新增 `client *ssh.Client` 字段和 `mu sync.Mutex` 读写锁。
  2. 增加内部方法 `getClient() (*ssh.Client, error)`，实现惰性初始化（Lazy Initialization）。如果 `client` 存在且未断开，则直接复用；否则建立新连接。
  3. 修改 `RunCommand` 逻辑，使用 `getClient()` 获取 `ssh.Client`，仅创建 `Session` 而不再反复 Dial TCP。
  4. 新增 `Close()` 方法用于部署结束后释放资源。

### 步骤三：生命周期接管 (Refactor Phase)
- **文件**: `godeployer/engine.go`
- **变更**:
  1. 在 `DeployEngine.RunDeploy` 的结束处理中（`defer` 处），若 `executor` 实现了 `io.Closer`，则显式调用 `Close()` 释放复用的 SSH 连接。

### 步骤四：二次压测与对比验收
- **文件**: `docs/sps/logs/benchmark_results.md`
- **变更**:
  1. 运行 `go test -bench` 并对冲性能。
  2. 记录连接复用后的 P99 延迟，确保响应速度显著提升，并将结果存档。

---
**[出口准则]**：用户 Review 后，启动第一步（基线测试）。
