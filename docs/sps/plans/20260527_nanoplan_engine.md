# TASK-004: GoDeployer 核心部署引擎纳米计划

本计划旨在实现 GoDeployer 部署引擎的关键链路：Git 代码检出、构建执行、Rsync (带 `--link-dest` 硬链接增量) 传输、原子软链接切换（`ln` + `mv` 无缝发布）、前置/后置钩子执行以及回滚。

---

## 1. 拟修改/创建文件清单

- **[NEW]** `godeployer/engine.go` - 核心部署流水线引擎
- **[NEW]** `godeployer/ssh.go` - SSH 执行与 Rsync 传输客户端

---

## 2. 纳米级任务细分 (代码量限制在 10-20 行/步)

### 2.1 任务 1: SSH 客户端与命令执行封装
*文件路径*: `godeployer/ssh.go`

- **[x] [Sub-Task 1.1]** 编写 `SSHClient` 结构体，封装 `golang.org/x/crypto/ssh` 鉴权与会话管理。
- **[x] [Sub-Task 1.2]** 编写 `RunSSHCommand` 函数，支持通过私钥连接目标服务器并执行命令，返回 stdout/stderr。
- **[x] [Sub-Task 1.3]** 编写 `RunRsync` 函数，通过 os/exec 执行本地 `rsync` 进程，传入 `--link-dest` 指向上一次部署的发布包目录以开启硬链接同步。

### 2.2 任务 2: 本地 Git 检出与构建执行
*文件路径*: `godeployer/engine.go`

- **[x] [Sub-Task 2.1]** 编写 `checkoutCode` 函数，在本地工作区执行 `git clone`（或对已存在的分支执行 `git pull`），并切换到指定的 Branch/Tag/Commit。
- **[x] [Sub-Task 2.2]** 编写 `runLocalBuild` 函数，读取项目 `before_sync` 的配置，在代码检出目录中循环执行外部命令（如 `npm run build`）。

### 2.3 任务 3: 目标机发布包生成与 Rsync 同步
*文件路径*: `godeployer/engine.go`

- **[x] [Sub-Task 3.1]** 编写 `syncToServers` 函数，为当前环境下的所有目标服务器创建基于时间戳的新宿主机 release 目录。
- **[x] [Sub-Task 3.2]** 调用 Rsync 客户端，获取目标机 `releases/` 下最近一次成功发布的版本名（通过读取 SQLite 任务历史或目标机 releases 目录），将其传入 `RunRsync` 作为 `--link-dest` 的基准。

### 2.4 任务 4: 原子软链接切换与回滚
*文件路径*: `godeployer/engine.go`

- **[x] [Sub-Task 4.1]** 编写 `switchSymlink` 逻辑，在目标机异步/同步发送：
  1. `ln -sfn /path/releases/new_version /path/current_temp`
  2. `mv -Tf /path/current_temp /path/current`
  实现零停机原子软链接更新。
- **[x] [Sub-Task 4.2]** 编写 `RunRollback` 函数，将软链接 `current` 的指向改回上一个健康版本，并标记当前损坏版本为 `BAD_RELEASE`。

---

## 3. 验证与单元测试规划 (物理零污染)

- **物理零污染与隔离**：
  - 测试将采用本地 Mock 环回方式进行。由于真实的部署使用 SSH 客户端，测试将通过 Mock 接口 `RemoteExecutor` 模拟 SSH 和 Rsync 命令的执行状态与返回值，而不去发起真实连接。
  - 本地测试时，Git 操作将在 Go 提供的随机 `os.MkdirTemp` 临时仓库中进行，测试完成后立即销毁。
- **单元测试覆盖**：
  1. `TestEngine_GitCheckout`：测试在临时 Git 仓库中正确拉取指定分支或特定 Commit。
  2. `TestEngine_RsyncLinkDest`：验证 Rsync 拼装参数中是否正确包含了 `--link-dest`。
  3. `TestEngine_AtomicSymlink`：测试在 MockExecutor 中抓取到的 SSH 命令，是否正确满足先 `ln -sfn` 后 `mv -Tf` 的严苛序列。
  4. `TestEngine_Rollback`：模拟部署失败，验证是否能退回上一版本并更新 SQLite 中的任务状态。
