# 纳米级执行计划 - 真实部署与双维度 Diff 自测流程

## 1. 拟修改与新增文件清单

### [NEW] [scripts/test_goal_dualdiff.sh](file:///d:/claudeprj/deploy/scripts/test_goal_dualdiff.sh)
- 编写端到端真实部署和预览测试脚本，包含：
  - 自动编译最新后端并重启 demo 服务。
  - 获取 JWT Token。
  - 测试上线前预览比对（Preview Diff）是否在 `branch` (全量) 下返回全量列表，在 `commit` (增量) 下返回变更列表。
  - 测试单文件 Diff 在 `diff_type=live` 和 `diff_type=git_log` 时的差异返回。
  - 触发真实上线（增量与全量），部署完成后，去检查 `task_diff` json 文件是否如期包含/精简快照，并校验 `/api/tasks/:id/diff` 的友好降级返回值。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 编写 `scripts/test_goal_dualdiff.sh`
- **逻辑内容**：
  1. 重启服务：
     `bash scripts/demo.sh stop && bash scripts/demo.sh start`
  2. 登录并提取 `admin` 的 Bearer Token。
  3. 执行比对与部署验证，检验增量（commit）和全量（branch）：
     - 测试一：预览 `/api/projects/thinkphp-web/preview_diff?to=master&target_type=branch`：断言返回全量文件（包含很多 php 文件），且不抛错。
     - 测试二：预览单文件 `preview_diff` 带有 `file=application/index/controller/Index.php&diff_type=live` 且带有 `diff_type=git_log`。
     - 测试三：以 `commit` 和 `branch` 两种方式各发起一次新部署任务。
     - 测试四：等任务状态为 `success` 时，从 `/api/tasks/:id/diff` 读取并做断言。全量任务对应的 live 模式必须返回降级提示。
  4. 验证结束，提示 Success。

---

## 3. 验证方案
- 在 WSL Debian 中运行：
  `wsl -d Debian -- bash scripts/test_goal_dualdiff.sh`
  必须看到所有测试断言全部通过（Exit Code 0）。
