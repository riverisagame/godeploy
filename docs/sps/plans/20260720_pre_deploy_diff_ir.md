# 部署前 Diff 检查 (Pre-Deployment Diff)
@Date: 2026-07-20

## 1. 背景与目标
在用户确认部署环境前，系统应对比目标分支的最新代码 (`HEAD`) 与该环境上一次成功部署的代码 (`current_release` 的 commit_hash)，并列出这两者之间的所有 Git Commits，以便用户审查本次发布到底包含了哪些改动。如果该环境是首次部署，则回退展示最近的 10 条 commits。

## 2. 核心问题与解决方案 (Adversarial Protocol & 性能对冲)
- **浅克隆问题**: 目前的 `GitClient.CloneOrPull` 采用了 `--depth=1`，这会导致本地只存在单层提交，无法执行 `git log from..to`。
  - **解法**: 修改 `GitClient.CloneOrPull`，移除 `--depth=1`，进行全量克隆（考虑到一般项目的提交历史大小可控），并在每次 `fetch` 时拉取分支的完整历史。
- **并发与锁**: 如果多个请求同时要求获取某个项目的 Diff，可能会导致工作区状态异常。
  - **解法**: 在 `GitClient` 中引入针对 `projectName` 的细粒度锁 (如 `sync.Map` 管理的 `sync.Mutex`)，确保 `fetch` 和 `log` 操作不会与其他 `fetch` 或 `CloneOrPull` 冲突。

## 3. 具体执行计划 (原子化步骤)

### 3.1 基础设施层 (Infrastructure - GitClient)
- 在 `internal/infrastructure/git/client.go` 增加针对项目的细粒度并发锁 `projectLocks sync.Map`。
- 移除 `clone` 时的 `--depth=1` 参数。
- 增加 `CommitInfo` 结构体：包含 Hash, Message, Author, Date。
- 增加 `FetchAndGetCommits(repoURL, branch, projectName, fromCommit string) ([]CommitInfo, error)` 方法：
  - 对项目加锁。
  - 检查目录是否存在，不存在则 clone，存在则 fetch (安静模式 `git fetch -q origin branch`)。
  - 根据 `fromCommit` 构建 `git log` 命令：
    - 若 `fromCommit != ""`：执行 `git log fromCommit..origin/branch --pretty=format:"%H|%s|%an|%ad" --date=iso`
    - 若 `fromCommit == ""`：执行 `git log origin/branch -n 10 --pretty=format:"%H|%s|%an|%ad" --date=iso`
  - 解析输出并返回 `[]CommitInfo`。

### 3.2 领域与服务层 (Domain & Application)
- 在 `internal/domain/git.go` 中抽象出 `GitClient` 接口的变更，增加 `FetchAndGetCommits`。
- 在 `DeployService` 中新增 `GetEnvironmentDiff(projectID uint, envName string) ([]CommitInfo, error)`:
  - 查出环境 `Environment` 和项目 `Project`。
  - 查找该环境**最后一次成功**的 `Deployment`，提取 `CommitHash`。如果未找到，置为空。
  - 调用 `GitClient.FetchAndGetCommits` 返回结果。

### 3.3 接口层 (Interfaces - API)
- 在 `internal/interfaces/api/project_handler.go` 增加方法 `GetEnvironmentDiff`。
- 注册路由 `GET /api/projects/:id/environments/:env_name/diff`。

### 3.4 前端界面 (Web - Vue)
- 在 `EnvironmentConfig.vue` 增加弹窗 `<el-dialog title="部署确认 (Diff)">`。
- 将点击“立即部署”按钮的逻辑改为：
  - 弹出该 Dialog，并显示 Loading。
  - 调用 API 获取 Diff 列表。
  - 用 Table 展示。如果列表为空，则提示“没有新提交”。
  - 用户在 Dialog 中点击最终的“确认发布”后，再触发 `/api/deployments`。

## 4. 评审与出口 (User Review Required)
> [!IMPORTANT]
> 取消 `--depth=1` 将会使初次部署的拉取时间略微变长。这是由于必须要有本地历史记录才能完成本地的 log 对比。如果不取消，我们就只能依赖调用 Git 提供商（如 GitHub/GitLab）的 API 进行对比，这会引入外部依赖。当前的做法（取消浅克隆）是无外部依赖、最轻量且支持任意 Git 协议的解法。请确认此设计。

以上为 [SCAN] 与 [IR] 阶段产出，请 Review。若无误，请回复 '继续' 以启动 [COMPILING] (测试驱动) 阶段。
