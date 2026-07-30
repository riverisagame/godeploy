# Bug修复计划 (2026-07-30)

## 目标
根据静态代码分析 (`golangci-lint`) 结果，修复现存的代码隐患，以提升代码的健壮性和生产可用性，不添加任何新功能。

## 任务拆解 (纳米级)

### Task 1: 忽略或记录不关注返回值的函数 (errcheck)
- **文件:** `internal/application/deploy_scheduler.go`
- **逻辑改动:** 将第33, 60, 73行的 `s.repo.Save(d)` 修改为 `_ = s.repo.Save(d)`，明确表示忽略错误，避免告警。或者捕获并记录日志。由于原先未处理且对上下文无致命影响，这里采取显式忽略。

### Task 2: 显式忽略 HTTP Body 关闭时的错误 (errcheck)
- **文件:** `internal/application/webhook_dispatcher.go`
- **逻辑改动:** 将第55行 `defer resp.Body.Close()` 替换为 `defer func() { _ = resp.Body.Close() }()`。
- **文件:** `internal/interfaces/api/webhook_handler.go`
- **逻辑改动:** 将第53行 `defer r.Body.Close()` 替换为 `defer func() { _ = r.Body.Close() }()`。

### Task 3: 补充测试断言 (ineffassign)
- **文件:** `internal/infrastructure/persistence/audit_repository_test.go`
- **逻辑改动:** 第60行 `logs` 被赋值后未被使用。在第62行下面补充对 `logs` 的断言：`assert.Len(t, logs, 1)`。

### Task 4: 代码规范优化 (staticcheck)
- **文件:** `internal/application/pipeline_runner.go`
- **逻辑改动:** 第55行 `if config.RunOn == "local" { ... } else if config.RunOn == "remote" { ... }` 优化为 `switch config.RunOn`，提升可读性。
