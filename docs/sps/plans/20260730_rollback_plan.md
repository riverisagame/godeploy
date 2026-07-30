# 执行计划: 自动回滚与零污染部署 (2026-07-30)

## 目标
修复 `DeployEngine` 部署失败时环境污染的问题，实现出错自动回滚。

## 任务拆解 (纳米级)

### Task 1: 获取上一次成功的版本号
- **文件:** `internal/application/deploy_engine.go`
- **函数:** `runDeploySteps`
- **逻辑改动:** 在执行 `e.runner.Run` 前，查询 `e.deploySvc.GetLastSuccessfulRelease(env.ID)` (需要新增该方法或类似方法)，或者直接通过 SSH 在服务器上执行 `readlink current` 获取旧版本路径。由于 `DeployEngine` 不关心服务器细节，最好从 `e.deploySvc` 获取上一次成功的 `ReleaseName`。

### Task 2: 实现 DeployService 的 GetLastSuccessfulDeploy 方法
- **文件:** `internal/application/deploy_service.go`
- **函数:** 新增 `GetLastSuccessfulDeploy(envID uint) (*domain.Deployment, error)`
- **逻辑改动:** 
  1. 调用 repo 方法。

### Task 3: 实现 Repository 层的查询
- **文件:** `internal/infrastructure/persistence/deployment_repository.go`
- **函数:** 新增 `FindLastSuccessful(envID uint) (*domain.Deployment, error)`
- **逻辑改动:**
  1. SELECT * FROM deployments WHERE environment_id = ? AND status = 'success' ORDER BY id DESC LIMIT 1;
  *(限制在 10-20 行以内)*

### Task 4: 在失败时触发回滚
- **文件:** `internal/application/deploy_engine.go`
- **函数:** `runDeploySteps`
- **逻辑改动:** 
  1. 在获取到 `lastDeploy` 的情况下，若 `e.runner.Run(...)` 返回 error。
  2. 打印日志 ">>> 部署失败，触发自动回滚..."。
  3. 调用 `e.Rollback(deployment, env, lastDeploy.ReleaseName)` 进行异步回滚（因为该函数是异步的，或者将其改为同步/等待它完成以确保状态正确）。

### 预演与攻击
- 若是环境初次部署，没有上一次成功的版本怎么办？自动回滚应当跳过。
- Rollback 函数是异步执行的，会导致当前任务马上结束而回滚在后台，应改造成同步调用，或在当前方法里直接等待其完成。
