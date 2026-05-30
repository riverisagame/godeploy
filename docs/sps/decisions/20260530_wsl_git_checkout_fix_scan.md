# WSL Debian Demo 无法发布问题排查与修复决策

## 1. 背景与现象
用户反馈在执行 `wsl debian demo` 测试时遇到发布失败。经过日志分析（`task_224.log`），具体的错误为：
`Error: git checkout failed: exit status 128 (output: fatal: unable to read tree (625c5287...))`

## 2. 根因分析
1. **WSL 与 Windows 跨文件系统的 Git 硬链接缺陷**：
   在 `godeployer/engine.go` 和 `godeployer/git.go` 中，为了极速部署，系统通过执行 `git clone`（或 `git clone --bare`）将演示库从本地源进行克隆。
   Git 在克隆本地目录时默认使用硬链接（hardlinks）优化对象复制。然而，在 WSL 访问 Windows 文件挂载（如 `/mnt/d/` 的 9P/DrvFS 协议）时，硬链接可能失败或权限异常，导致目标仓库虽然生成，但内部 Git 对象树实际上损坏。此时尝试 `git checkout <commit_id>` 就会抛出 `unable to read tree`。

2. **配置文件中硬编码了绝对路径（产生平台割裂）**：
   在 `scripts/demo.sh` 脚本的仓库关联环节中，系统使用 `pwd` 直接拼接了本地绝对路径注入到 `config.yaml`。当环境为 WSL 时，注入的路径是 `/mnt/d/...`；如果在 Windows 下运行，注入的则是 `D:\...`。这会导致环境切换时跨平台路径识别错乱。

## 3. 解决方案设计
1. **引擎防脆化 (Engine Resilience)**：
   - 修改 `godeployer/engine.go` 与 `godeployer/git.go`，在所有本地 `git clone` 命令中，强制追加 `--no-hardlinks` 参数，强制完整拷贝 Git 对象，彻底免疫跨文件系统挂载导致的索引损坏问题。
2. **演示脚本去绝对路径化 (Script Portability)**：
   - 调整 `scripts/demo.sh`，在替换 yaml 配置文件中的 `repo` 地址时，使用工作区相对路径 `demo_workspace/gitee_demo/$name` 而非 `pwd` 返回的绝对路径，确保生成的配置文件在所有系统中保持一致。

## 4. 影响范围（Blast Radius）
- `godeployer/engine.go` 的 `prepareWorkspace` 克隆逻辑（仅改变参数，不改变业务流）。
- `godeployer/git.go` 的 `EnsureRepoCache` 缓存逻辑。
- `scripts/demo.sh` 的 `repo` 路径替换。
修改非常局限，并且从物理上消除了跨平台兼容性的地雷，满足“纳米线测试驱动开发”中零污染、高对冲的硬核审计原则。
