# 纳米级计划 (IR): 极致测试覆盖率与详尽配置文档化

## Phase 1: 文档解耦与精细化 (Documentation)
### Step 1.1: 创建小白友好的详细配置文档
- **文件路径**: `docs/CONFIGURATION.md`
- **改动逻辑**:
  - 创建 Markdown 文件，从“基础认知（什么是部署器配置）”开始写。
  - **结构体映射**: 将 `config.go` 中的 `GlobalConfig`, `ProjectConfig`, `BuildConfig`, `EnvironmentConfig`, `ServerConfig` 全字段逐一解释。
  - **白话说明**: 对 `BeforeSync`（代码同步前在本地或远端拉取时的钩子）与 `BeforeSymlink`（远端新目录准备好后，软链切换前的钩子）做对比解释。对 `SSHKeyPath` 给出具体示例路径（如 `/home/user/.ssh/id_rsa`）。
  - **行数预估**: ~150 行 (Markdown)。

### Step 1.2: 更新主 README.md 引导
- **文件路径**: `README.md`
- **改动逻辑**:
  - 在配置说明板块（如果已有）或安装说明后，清空长篇大论的配置示例。
  - 替换为：`> 📚 **关于配置的终极指南**：请阅读 [CONFIGURATION.md](docs/CONFIGURATION.md)，里面包含了从入门到精通的所有配置项、钩子函数说明以及实战范例。`
  - **行数预估**: ~10 行。

## Phase 2: 后端测试漏洞填补 (Go Backend Coverage)
### Step 2.1: API 接口异常边界覆盖
- **文件路径**: `godeployer/api_test.go` (新建或追加)
- **改动逻辑**:
  - 增加测试 `TestHandleTasks_InvalidProject`，传入不存在的项目 ID，断言返回 HTTP 400/404。
  - 增加测试 `TestHandleDeploy_InvalidEnv`，传入配置中没有的环境 ID，断言拦截。
  - **行数预估**: ~30 行。

### Step 2.2: SSH 并发与超时防御测试
- **文件路径**: `godeployer/ssh_test.go`
- **改动逻辑**:
  - 增加测试 `TestSSHExecutor_Timeout`，通过 `context.WithTimeout(ctx, 1*time.Millisecond)` 创建上下文。
  - 执行 `executor.ExecuteCommand("sleep 5")`，断言返回 `context deadline exceeded`，确保不造成协程泄漏。
  - **行数预估**: ~20 行。

### Step 2.3: 数据库级联更新与锁竞争测试
- **文件路径**: `godeployer/db_test.go`
- **改动逻辑**:
  - 增加 `TestDB_ConcurrentTaskUpdates`，使用 `sync.WaitGroup` 启动 10 个 goroutine，同时调用 `UpdateTaskStatus` 对同一个 TaskID 修改状态。
  - 最终查询数据库确保没有 `database is locked`（由于设置了 SQLite Busy Timeout）。
  - **行数预估**: ~25 行。

## Phase 3: 前端组件交互渲染覆盖 (Vue Frontend Coverage)
### Step 3.1: Dashboard 核心组件挂载测试
- **文件路径**: `web/src/__tests__/Dashboard.spec.ts` (新建)
- **改动逻辑**:
  - 使用 `@vue/test-utils` 的 `mount`。
  - 引入 `vi.mock("axios")`，模拟 `/api/projects` 返回假数据。
  - 断言 DOM 中成功渲染出了假数据的 Project Name。
  - 触发项目切换的 Tab Click 行为，断言 active_tab 被更新。
  - **行数预估**: ~40 行。

### Step 3.2: 部署表单与确认弹窗阻断测试
- **文件路径**: `web/src/__tests__/DeployForm.spec.ts` 或对应组件
- **改动逻辑**:
  - 挂载包含“发版按钮”的组件，在未选择分支的情况下点击“Deploy”。
  - 断言触发了 `ElMessage.warning`（或拦截器返回 false），确保前端第一道防线生效。
  - **行数预估**: ~30 行。
