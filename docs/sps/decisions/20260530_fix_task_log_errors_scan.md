# 架构决策记录 - 修复任务部署日志 Git 树读取失败与缓存不一致问题

## 1. 背景与问题剖析

在 GoDeployer 演示或测试环境中，部署任务有时会以以下错误失败：
```
Error: git checkout failed: exit status 128 (output: fatal: unable to read tree (cfa3adb16633755e94c72e74c65987d8a3a04589))
```

经过深入调试与分析，发现根本原因为**Git 本地缓存库（Bare Repo Cache）的 Remote URL 不一致与清理机制漏洞**：

1. **缓存 URL 变更未同步**：
   - 之前测试中，`demo_projects.d/*.yaml` 中的 `repo` 可能指向了远程的 Gitee/Github 地址。
   - 本地裸缓存库（如 `demo_workspace/.cache/thinkphp-web.git`）一经创建，其 `origin` 便永久指向了远程地址。
   - 当 `scripts/demo.sh` 被修改为使用本地 Mock Git 仓库后，后端配置文件的 `repo` 已更改为本地 Mock 路径。
   - 然而，后端在部署时由于缓存目录已存在，只执行了 `git fetch origin`，即依然往远程 Gitee/Github 同步。
   - 此时，通过 API 触发部署的 commit 哈希是本地 Mock 生成的哈希，但在远程缓存中根本不存在该哈希，从而导致 `git checkout` 报 `unable to read tree`。

2. **缓存清理机制漏洞**：
   - 在 `scripts/demo.sh` 中，清理缓存的逻辑被包裹在 `if [ -f "$DB_PATH" ]` 中。如果数据库文件在运行前被删除了，清理逻辑便会被跳过，从而残留旧的缓存，触发上述问题。

3. **前端 Dashboard 缺失组件引入**：
   - 前端组件 `Dashboard.vue` 模板使用了 `DeployForm` 和 `DeployHistoryTable`，但 `<script setup>` 里缺失导入，导致前端无法定义并隐藏了整个操作区。

---

## 2. 解决方案设计与性能对冲

### 2.1 后端 Git 缓存健壮性优化
在 `godeployer/git.go` 的 `EnsureRepoCache` 函数中，增加 remote URL 校验：
- 在 fetch 之前，读取当前裸缓存的 `origin` URL（通过 `git remote get-url origin`）。
- 校验获取到的 URL 是否与配置中的 `repoURL` 严格一致。
- 如果不一致，说明项目仓库地址发生了更改，直接清空该项目的裸缓存目录，并重新进行 `git clone --bare`。
- **性能对冲**：校验 remote URL 的操作仅需执行一次 `git config` 或 `git remote` 查询，开销在 10ms 以内，不会影响系统的高并发和响应时间（依然控制在 150ms 以内）。

### 2.2 演示脚本清理优化
在 `scripts/demo.sh` 中：
- 将清除旧日志、Git 缓存的逻辑移出 `if [ -f "$DB_PATH" ]` 条件，确保每次调用 `seed` 或 `all` 都会彻底清理旧缓存，杜绝历史状态污染。
- 确保所有的 Mock commits 均在 `master` 主分支进行线性提交，再从中分出 `develop` 等分支，确保克隆出来的 refs 不会缺失。

### 2.3 前端组件导入修复
- 检查 `web/src/views/Dashboard.vue` 的组件导入，确保 `DeployForm` 和 `DeployHistoryTable` 导入正确。

---

## 3. 出口准则
- 物理写入此 ADR，并更新 `MASTER_LOG.md`。
- 输出 `[SCAN_COMPLETE]`，等待用户确认后进入 `[IR]` 纳米级计划阶段。
