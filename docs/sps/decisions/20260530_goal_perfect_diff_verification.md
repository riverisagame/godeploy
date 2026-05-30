# 架构决策记录 (ADR) - 保证 Diff 过滤及局部比对功能 100% 可行可用

## 1. 深度分析与审计 (Adversarial Audit)
为了达成 100% 的高可用性与上线安全保障，我们从“实现”、“自我攻击”和“性能对冲”三个维度做深挖分析：

### 1.1 实现层面 (Implementation)
- **获取文件列表**：目前 `HandleGetProjectPreviewDiff` 通过 `git diff --name-only fromCommit toCommit` 获取。
- **获取单文件 Diff**：`HandleGetProjectPreviewDiff` 接收 `file=path` 参数，调用 `GetDiffForFile` 获取指定文件的 diff 内容。
- **问题兜底风险**：如果 `fromCommit`（即上次部署成功的 commit）因为仓库初次拉取、或者是极端的 shallow clone 等原因，导致在本地 Bare 库里找不到该 commit 对象（或者 `toCommit^` 不存在），则 `git diff` 命令会执行失败（如 `fatal: bad object`），导致返回空文件列表。
- **解决手段**：如果 `git diff --name-only fromCommit toCommit` 报错，系统应降级为通过 `git log -n 1 --name-only --format= toCommit` 提取单次 Commit 的变动文件，确保不出现白屏，并返回详细的系统警告提示用户。

### 1.2 自我攻击层面 (Self-Attack & Security)
- **Shell 注入风险**：在 `GetDiffForFile` 和 `HandleGetProjectPreviewDiff` 里，`file` 参数是作为 `exec.CommandContext` 的参数直接传递给 `git` 执行。由于 Go 的 `exec.Command` 底层使用的是系统级系统调用（如 `execve`），而不是通过 `sh -c` 执行，所以不存在参数被 Shell 拼接注入（如 `; rm -rf /`）的传统注入漏洞。但是如果路径参数中包含 `--` 或以 `-` 开头可能会被 Git 解释为命令行选项。
- **缓解防护**：我们应该在命令中加入双减号 `--` 来将选项与路径参数隔离：
  `git diff --name-only fromCommit toCommit` 改成 `git diff --name-only fromCommit toCommit --`
  目前我们在 `GetDiffForFile` 中已经使用了 `args = []string{"diff", fromCommit, toCommit, "--", file}` 进行隔离，这是安全的；但 `api.go` 的 `git diff --name-only` 命令需要添加 `--` 避免潜在参数溢出风险。

### 1.3 性能对冲与高并发 (Performance & Concurrency)
- **单通道 SQLite 限制**：部署平台具有单链接 SQLite 写入机制，对元数据操作是安全的。
- **大文件截断**：前端已集成限制 100KB 的安全截断。
- **后端内存保护**：当接口读取单个文件 Diff 时，后端在 `GetDiffForFile` 同样应用了 `limitBytes` 对读入的文本大小进行提前拦截截断，避免大文本在 Go 后端内存中引发 OOM。

## 2. 最终加固实施计划
1. **[加固 1] 后端 git 路径安全隔离**：在 `api.go` 第 383 行执行 `git diff --name-only` 时在尾部添加 `--` 作为防御屏障。
2. **[加固 2] 首次部署/空数据降级保护**：当 `fromCommit` 不存在或 Git 返回错误时，提供备用的降级查询，并在前端给出明确的兜底渲染。
3. **[加固 3] 后端单元测试加固**：物理新增对 `HandleGetProjectPreviewDiff` 的边界输入单元测试。
