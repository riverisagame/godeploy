# TASK-018: Demo API-based Real Operation Simulation Plan (基于 API 真实模拟的 Demo 优化计划)

修改后端 `ssh.go` 引入本地免 SSH 旁路，并更新 `demo.sh` 脚本以通过多用户 API 调用真实触发所有 Demo 任务和日志生成。

## Proposed Changes

### Component: Backend Executor (godeployer/ssh.go)

#### [MODIFY] [ssh.go](file:///d:/claudeprj/deploy/godeployer/ssh.go)

1. 在 `RunCommand` 中：
   当 `s.Server.Host == "localhost" && s.Server.Port == 2222` 时，直接利用 `exec.Command` 在本地执行指令并返回结果，不进行 SSH 连接。
2. 在 `Rsync` 中：
   当 `s.Server.Host == "localhost" && s.Server.Port == 2222` 时，直接在本地执行 `rsync` 命令将文件从 `local` 同步到 `remote`，不通过 `-e ssh` 选项。

### Component: Demo Projects Configuration (demo_projects.d/*.yaml)

#### [MODIFY] [thinkphp.yaml](file:///d:/claudeprj/deploy/demo_projects.d/thinkphp.yaml)
#### [MODIFY] [webman.yaml](file:///d:/claudeprj/deploy/demo_projects.d/webman.yaml)
#### [MODIFY] [crmeb.yaml](file:///d:/claudeprj/deploy/demo_projects.d/crmeb.yaml)

1. 将 `repo` 属性指向本地的绝对路径（如 `$GITEE_WORKSPACE/think`），这样拉取过程是 100% 本地极速的。
2. 将 `servers` 中的 `host` 设为 `localhost`，`port` 设为 `2222`。
3. 模拟部分环境失败：
   - 将 `webman.yaml` 中的 `production` 环境目标路径改为 `/root/forbidden_deploy/webman`（无写入权限，导致 rsync 报权限错误失败）。
   - 将 `crmeb.yaml` 中的 `production` 环境的 `port` 改为 `2223`（在 ssh.go 里没有匹配的旁路，导致连接拒绝失败）。

### Component: Demo Bootstrap Script (scripts/demo.sh)

#### [MODIFY] [demo.sh](file:///d:/claudeprj/deploy/scripts/demo.sh)

1. `seed_db` 更改为 `seed_via_api`：
   - 只写入系统基础表（如清空原有 tasks 以便重新生成）。
   - 通过 `POST /api/users` 创建 `deployer` 和 `viewer`。
   - 分别获取 `admin`、`deployer`、`viewer` 的 Token。
   - 交错并循环使用这些 Token 发起 `POST /api/tasks` 部署任务（分别传入 Mock 仓库的 5 个真实 commit 哈希），实现多用户、多仓库的交错部署。
   - 在部署时通过轮询 `/api/tasks/:id` 接口直到状态更新为终止态（`success` 或 `failed`），再发起下一个，保证日志和顺序的整洁真实。

## Verification Plan

### Automated Tests
1. 运行 `bash scripts/demo.sh` 并观察部署引擎并发执行和产生的真实任务和日志。
2. 确认数据库中产生的 100% 任务数据与 Git mock 仓库的 commit-id、username 完美对应且无任何伪造字段。
3. 访问 [http://localhost:8080](http://localhost:8080) 查看历史日志。
