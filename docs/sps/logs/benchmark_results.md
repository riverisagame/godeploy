# SSHExecutor 性能基准测试与调优记录 (Milestone 1)

## 1. 压测环境
- CPU: Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz
- OS: Windows (amd64)
- Network: 127.0.0.1 (Local loopback mock SSH server)

## 2. 压测结果对比

### 2.1 [RED 阶段] 优化前：单次握手不复用
```text
BenchmarkSSHExecutor_RunCommand-12           146   7436965 ns/op      139879 B/op       963 allocs/op
```
**分析**: 在极低延迟的本地环回网络下，每次指令都需要进行 TCP 握手和 SSH 密钥协商，耗时约 **7.4 ms/op**，内存分配极高 (139KB/op)。在真实的广域网（延迟 50ms-100ms）环境下，单次连接开销会被放大至 300ms+，一次包含 10 步指令的部署会额外增加 3 秒以上的纯网络开销。

### 2.2 [GREEN 阶段] 优化后：惰性连接池化复用
```text
BenchmarkSSHExecutor_RunCommand-12          3632      333245 ns/op        7671 B/op       138 allocs/op
```
**分析**: 
在 `SSHExecutor` 引入内置 `sync.Mutex` 和连接持留复用后，单次远程执行的开销骤降至 **0.33 ms/op**，性能提升约 **22 倍**。内存分配大幅下降至 **7.6KB/op** (下降约 18 倍)。
在真实广域网环境中，由于避免了繁重的 TCP 三次握手和 SSH 密钥交换算法，整体部署速度预计将提升 1-3 秒，极大降低了对服务端的握手风暴风险。
