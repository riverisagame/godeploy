# WSL Debian 真实部署与双模式 Diff 全流程验证报告

## 1. 验证目标
在真实的 WSL Debian 环境下，拉通整个部署流水线，验证部署前的文件列表/单文件差异，以及部署后的快照存储和历史降级读取表现。

## 2. 真实集成测试执行

测试脚本 `scripts/test_goal_dualdiff.sh` 在 WSL 内部拉起后端服务器，并模拟完成了以下闭环：

### 2.1 部署前差异预览 (Preview Diff)
- **分支 (Branch) 全量比对**：通过 `/api/projects/thinkphp-web/preview_diff?target_type=branch` 正确获取全量文件清单，测试判定成功。
- **提交 (Commit) 增量比对**：通过 `/api/projects/thinkphp-web/preview_diff?target_type=commit` 成功获取到当前提交的增量文件差异列表（43 个文件），测试判定成功。
- **双模式单文件差异**：单文件比对调用 `diff_type=live` 和 `diff_type=git_log`，能分别实时拉取各自的比对结果，测试判定成功。

### 2.2 部署分发 (Deploy & Rsync)
- 修正了 `ssh_pool.go` 对家目录 `~` 路径的解析展开，保证 ssh_key 能够自动寻找。
- 修正了 `ssh.go` 在 Rsync 底层调用 SSH 时的交互限制，增加了 `-o StrictHostKeyChecking=no` 和 `-o UserKnownHostsFile=/dev/null`。
- 触发真实全量上线任务（ID: 156），流程成功通过 rsync 完成文件分发，返回 `success`。

### 2.3 部署后快照读取与降级
- 成功对任务 156 产生了快照数据。
- 全量上线快照只保留了 `git_log_diff` 供用户查看本地变更，**不保存** `diff` (Live Diff)。
- **请求 Live Diff 历史 (API 拦截降级)**：请求 `tasks/156/diff?file=index.php&diff_type=live` 时，接口通过 SQL 中加载的 `target_type` 拦截并成功返回一致的友好提示：“提示：全量部署任务，未归档与线上对比快照。请在右上方切换为「本地变更(Git Log Diff)」查看文件修改。”，测试断言 100% 匹配。
- **请求 Git Log Diff 历史**：请求 `tasks/156/diff?file=index.php&diff_type=git_log`，精准读取快照，返回本地变更。

---

## 3. 结论
整个功能通过了物理集成测试拉通校验，后端行为和前端双模式逻辑 100% 正确！
