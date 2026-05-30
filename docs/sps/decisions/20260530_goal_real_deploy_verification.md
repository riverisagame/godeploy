# 架构决策记录 (ADR) - 双模式 Diff 与真实发布集成测试设计

## 1. 业务目标
确认并确保上线前与上线后的双维度比对功能完全就绪，并且在 WSL Debian 的真实环境下通过端到端的部署，自测验证：
- 上线前预览（Preview Diff）文件列表展示逻辑：
  - 分支/Tag：全量文件清单。
  - Commit：增量变更文件清单。
- 上线前单文件 Diff：
  - 均支持 Live Diff (与线上对比) 与 Git Log Diff (本地变更)。
- 上线后历史 Diff：
  - Commit 任务：快照中完整保留 Live Diff 和 Git Log Diff。
  - Branch/Tag 任务：快照中仅保留 Git Log Diff。

## 2. 真实集成测试计划 (WSL Debian)
我们将编写一个专门的集成测试脚本 `scripts/test_goal_dualdiff.sh`（或 `test_goal_dualdiff.py`），在 WSL 内部运行，做如下覆盖：
1. **启动并初始化**：利用 `scripts/demo.sh restart` 保证后端干净且运行。
2. **预览接口测试**：
   - 用 `commit` (增量) 方式请求 `preview_diff` 接口，确认仅返回变更文件列表。
   - 用 `branch` (全量) 方式请求 `preview_diff` 接口，确认返回该分支下的全量文件列表。
   - 分别请求单文件 `diff_type=live` 和 `diff_type=git_log`，确认返回相应的 diff 差异文本。
3. **真实发布与快照归档测试**：
   - 触发一次 `target_type=commit` 真实部署，等待任务 success 后，调用 `tasks/:id/diff` 快照接口，比对快照中是否同时有 `diff` 和 `git_log_diff`。
   - 触发一次 `target_type=branch` 真实部署，等待任务 success 后，调用 `tasks/:id/diff` 快照接口，比对快照中是否只有 `git_log_diff` 且请求 `live` 时返回友好降级提示。
4. **前端编译与内联校验**：
   - 跑完 `npm run build` 和 `go build .`。

## 3. 安全与只读保障
测试是在 demo 环境自带的 Gitee Mock 镜像仓库上执行，只针对 `test-app` 等沙箱项目做临时 task 插入和测试，对真实系统 100% 只读且毫风毫雨无污染。
