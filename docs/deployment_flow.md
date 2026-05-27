# 部署流水线与“两阶段发布”机制

GoDeployer 的核心优势在于确保多节点集群的高可用与业务的无缝切换。为达到这一目标，我们引入了类似数据库中“两阶段提交” (Two-Phase Commit) 的设计概念。

## 流水线整体步骤

一次标准的部署请求将经历以下 5 个步骤：

1. **Step 1: Cloning Repository**
   - 调度系统在执行机（Runner）本地创建一个隔离的临时工作区目录。
   - 执行 `git clone` 将代码拉取到本地，确保与目标机器无关。

2. **Step 2: Checking out Target**
   - 根据用户指定的 `Commit ID` 或 `Branch/Tag`（如 `master`），执行 `git checkout`。
   - 锁定代码版本，确保此次部署的代码处于确定性状态。

3. **Step 3: Local Build Hooks**
   - 在执行机本地执行如 `npm run build`、`composer install` 等预编译/依赖安装动作。
   - **优势**: 显著降低目标生产服务器的 CPU/内存负担，避免因编译时内存耗尽导致生产宕机。

4. **Step 4: Phase 1 (Rsync 同步)**
   - 引擎开启高并发 SSHPool，将执行机上已经 Build 完毕的文件通过 `rsync -avz` 并发同步到所有目标服务器的**版本独立目录**（如 `/opt/app/releases/20260101999999`）。
   - **特点**: 此时业务代码尚未生效，即使该过程持续长达 10 分钟或中途网络中断，**绝对不会影响线上运行中的业务服务**。

5. **Step 5: Phase 2 (Symlink 软链接切换)**
   - 所有目标机器均同步成功后，引擎下发统一的秒级切换命令：
   - 将目标服务器的 `/opt/app/current` 软链接原子性地指向 `/opt/app/releases/20260101999999`。
   - **特点**: 该操作在毫秒级内完成，结合 PHP-FPM / Nginx 等机制，能够达成业务的无停机热更 (Zero-Downtime Deployment)。

## 脑裂保护与自动回滚 (Rollback)

在**Phase 2 软链接切换**过程中，如果某几台服务器的软链接切换成功，而其他几台因为磁盘只读等问题导致切换失败，系统就会陷入部分新版本、部分旧版本的**脑裂 (Brain-Split)** 危机。

GoDeployer 通过以下机制保护集群一致性：
1. **失败感知**: 调度系统捕获任何节点的 `exit status != 0`。
2. **集群级 Rollback**: 一旦触发失败，立刻暂停剩余节点的切换动作，并反向发起 `Rollback` 指令，强制将所有已切换成功的节点恢复为旧版本的路径（`previous_release`）。
3. **标记挂起**: 标记该 `Task` 为 Failed，输出详细错误栈并中止部署流程，确保用户人工介入前集群依旧对外提供稳定且一致的旧版服务。
