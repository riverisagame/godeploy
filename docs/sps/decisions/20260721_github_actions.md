# Github Actions CI/CD 设计方案与选型

## 1. 核心诉求
- 为项目增加 GitHub Actions 自动化工作流。
- 包含两部分：代码审查与测试 (CI)；发版与构建 (CD/Release)。
- 明确指定环境版本要求：Go `1.26.5`，并且使用最新稳定版工具链。
- 要求内置 `go test -race`（静态条件/竞态检测）以及 `golangci-lint`（代码风格和潜在隐患静态检查）。

## 2. 工具选型
| 功能 | 选用工具组件/Action | 版本策略 | 选择理由 |
| --- | --- | --- | --- |
| 基础环境 | `actions/checkout` | `v4` | 官方主推当前最稳定版 |
| Go环境配置 | `actions/setup-go` | `v5` | 原生内置了 GOCACHE 及 GOMODCACHE 支持加速 |
| Linter工具 | `golangci/golangci-lint-action` | `v6` | 官方原生镜像构建，包含并行加速，免手动下载，性能最优 |
| 跨平台编译打包 | 纯 Go 构建 (CGO_ENABLED=0) | N/A | 本项目使用 glebarez/sqlite (pure go)，完全脱离 CGO 依赖，交叉编译无痛。|
| 发版创建 | `softprops/action-gh-release` | `v2` | 轻量级、高可定制化且免依赖的 Release 发行工具 |

## 3. 对抗性边界防守
- **污染防范**：测试时，环境处于隔离沙盒，不涉及连接任何外部生产数据库。
- **并发性能**：通过配置 `matrix` (矩阵) 让全平台的编译(linux, windows, darwin)同时跑在多个runner，实现极致并行。
- **容错防线**：Lint 失败必须让 CI 爆红(RED)以防止劣质代码混入；当前项目的 `main` 分支有遗留编译问题，加入 CI 后第一次必然失败，这正是我们需要暴露的问题。
