# TEST_FAILED: SSH Local Bypass Initial Test (RED Phase)

验证在没有实现 `Local Bypass` 旁路逻辑时，`localhost:2222` 单元测试的执行表现。

## 测试环境
- WSL Debian
- Go 1.25.0
- `-race` 竞态检测启用

## 运行输出
```bash
=== RUN   TestSSHExecutor_LocalBypass
    ssh_local_bypass_test.go:33: RunCommand failed: SSH pool is not initialized
--- FAIL: TestSSHExecutor_LocalBypass (0.00s)
FAIL
FAIL	deploy/godeployer	0.035s
```

## 结论
符合预期。需要对 `godeployer/ssh.go` 进行最小化修改以实现对保留端口 `2222` 的本地处理逻辑。
