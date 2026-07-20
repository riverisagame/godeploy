# 执行计划：环境变量与机密管理 (Env Vars)

## 1. [RED] Domain层与持久化修改 (Domain & Repo)

**目标**：扩展 `Environment` 实体，在 SQLite 中存储 JSON 格式的变量。
**改动文件**：
- `internal/domain/project.go`
  - 增加 `EnvVar` 结构体 (Key, Value, IsSecret)。
  - 在 `Environment` 中增加 `EnvVars []EnvVar`。
- `internal/infrastructure/persistence/project_repository.go`
  - `EnvironmentModel` 新增 `EnvVars string`。
  - `GetProjectWithEnvironments`：解析 `m.EnvVars` -> JSON Unmarshal。
  - `SaveEnvironment`：将 `d.EnvVars` JSON Marshal 存入 `model.EnvVars`。
- `internal/domain/project_test.go` (新增或更新)
  - 编写对 `Environment.EnvVars` 的序列化/读取测试。

## 2. [GREEN] API 层更新

**目标**：允许前端在保存配置时，提交环境变量数组。
**改动文件**：
- `internal/interfaces/api/project_handler.go`
  - 修改 `UpdateHooks` 请求体（或重命名为 `UpdateEnvConfig`）：接收 `env_vars`。
  - 解析到 `domain.Environment`，调用 repo 保存。

## 3. [GREEN] DeployEngine 注入环境文件 (Engine)

**目标**：在部署核心逻辑中将变量安全地写入到目标机器的 `.env` 文件。
**改动文件**：
- `internal/application/deploy_engine.go`
  - 在生成发布目录后，拼接环境变量内容。
  - 将内容通过 `base64` 编码避免转义问题，然后执行类似 `echo $B64 | base64 -d > .env` 的命令。
  - 若 `pre_deploy` 或 `post_deploy` 有内容，在其执行前，添加 `. .env && ` 使上下文生效（或 `export $(cat .env | xargs)`）。

## 4. [GREEN] UI 前端重构 (Frontend)

**目标**：在 `EnvironmentConfig.vue` 界面增加 Key-Value 表格。
**改动文件**：
- `web/src/views/EnvironmentConfig.vue`
  - 在 Hooks 编辑区旁边/下方新增一个 "环境变量" 配置区。
  - 表格：`Key` (Input), `Value` (Input, 若 `is_secret` 为 true 则 type='password'), `Is Secret` (Switch), `操作` (删除)。
  - 提供 `+ 添加变量` 按钮。
  - `saveHooks` (改名为 `saveConfig`) 发送时附加 `env.env_vars`。
