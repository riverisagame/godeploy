# GoDeploy - Agent 指令

## 项目概要

GoDeploy 是基于 Go + Vue3 的代码发布与自动化部署系统。后端用 Gin 框架，前端用 Vite + Element Plus + TypeScript，SQLite 存储，SSH + rsync 实现原子化部署。

## 常用命令

```bash
# 后端：编译与运行
go build -o godeployer_bin ./godeployer     # 纯后端编译（不含前端）
go build .                                    # 完整编译（需前端已构建）
go run main.go --config=config.yaml           # 开发运行

# 后端测试（强制 -race）
go test -v -race ./...

# 前端
cd web && npm install && npm run dev          # 开发模式（代理 /api 到 :8080）
cd web && npm run build                       # 生产构建（输出到 web/dist/）
cd web && npm run test                        # Vitest 单元测试（jsdom）
cd web && npm run test:e2e                    # Playwright E2E（自动启动后端+前端）

# Demo 环境
bash scripts/demo.sh                          # 完整初始化（首次）
bash scripts/demo.sh start                    # 仅启动后端
```

## 编译顺序（重要）

前端产物 `web/dist/` 通过 `//go:embed dist` 嵌入到后端二进制中（`godeployer/main.go:21`）。**完整编译前必须先 `cd web && npm run build`**，否则嵌入静态资源为空。

开发时跳过此步：`go build -o godeployer_bin ./godeployer` 编译纯后端，由 Vite dev server 单独提供前端。

## CI 流水线

GitHub Actions（`.github/workflows/ci.yml`）3 个 Job，PR 任一失败即阻断合并：

| Job | 命令 | 依赖 |
|-----|------|------|
| `backend-test` | `go test -v -race ./...` | — |
| `frontend-unit` | `cd web && npm ci && npm run test` | — |
| `e2e-test` | `cd web && npm run test:e2e` | 前两个通过后执行 |

CI 使用 `npm ci`（非 `npm install`），要求 `package-lock.json` 同步。

## 架构

```
main.go                          # 入口，调用 godeployer.StartServer()
godeployer/                      # 所有后端逻辑（单一 package，无子包）
  main.go          StartServer, BootstrapApp, SetupStaticEmbed
  api.go           路由注册 / SetupRoutesWithExecutor / APIHandler
  auth.go          JWT + bcrypt 认证
  db.go            SQLite 初始化 + 表结构（users, deploy_tasks, webhooks）
  engine.go        DeployEngine 调度器（协程池，并发限制 3）
  config.go        YAML 配置加载
  ssh.go           SSH 执行器 + rsync
  ssh_pool.go      SSH 连接池
  notifier.go      事件总线 + WebSocket 实时推送
  sys_windows.go / sys_unix.go / disk_*.go   平台抽象
web/                             # Vue 3 + Vite 前端
  src/             Vue 组件、路由、API 封装
  e2e/             Playwright E2E 测试
docs/                            # 配置、架构、部署流程文档
doc/                             # 开发文档（DEVELOPMENT.md 内容已过时，参考 README）
scripts/          demo/install/update 等 shell 脚本
docs/sps/         Superpowers SDD 工作流产物（plans/ decisions/ logs/）
```

## 关键约束

- **godeployer 是单一扁平 package**：所有后端逻辑在 `godeployer/` 目录下，无子包。import 路径为 `deploy/godeployer`，不可按子目录拆分引用。
- **SQLite 单连接模式**：`godeployer/db.go:23` — `db.SetMaxOpenConns(1)`，并发写被串行化。
- **Go 版本要求 1.25+**（`go.mod` 声明 `go 1.25.0`）。
- **CI 强制 `-race`**：`go test -v -race ./...`，任何 data race 都会导致 CI 失败。
- **前端 Vite dev 代理**：`web/vite.config.ts` 将 `/api` 转发到 `BACKEND_PORT`（默认 8080）。
- **Config 结构**：全局 YAML + `project_config_dir` 下 `*.yaml` 文件组成项目配置。
- **RBAC 角色**：Admin（全部权限）、Maintainer（部署权限）、Viewer（只读）。

## 代码约定

- 代码注释和文档使用简体中文。
- Superpowers SDD 引用格式：`// @Ref: docs/sps/plans/... | @Date: YYYY-MM-DD`（见 `godeployer/config.go:24`、`godeployer/api.go:37`）。
- 测试代码与被测文件同目录，命名 `*_test.go`。

## GitNexus

`.gitnexusignore` 已配置忽略 demo 相关目录（第三方仓库、生成数据），避免索引噪声。

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **godeploy** (1973 symbols, 3461 relationships, 23 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/godeploy/context` | Codebase overview, check index freshness |
| `gitnexus://repo/godeploy/clusters` | All functional areas |
| `gitnexus://repo/godeploy/processes` | All execution flows |
| `gitnexus://repo/godeploy/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |
| Work in the Godeployer area (133 symbols) | `.claude/skills/generated/godeployer/SKILL.md` |

<!-- gitnexus:end -->
