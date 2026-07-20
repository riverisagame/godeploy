# 架构决策记录 (ADR)：环境变量与机密管理 (Environment Variables & Secrets)

## 1. 背景与需求
当前系统在自动化部署中，能够运行前置和后置脚本，但无法为不同环境（如生产、测试）独立注入环境变量和机密信息（如数据库密码、API Key）。导致这些配置只能硬编码或者手动上传，不符合“一次构建，到处部署”的工程实践。

## 2. 需求深挖与疑点质疑 (Adversarial Questioning)
在实现此功能前，我们需要确认以下几个核心设计边界：

### 疑问 1：注入方式 (Injection Method)
在执行部署脚本时，环境变量该如何注入？
- **方案 A（推荐）**：在部署目标服务器的 `release` 目录中自动生成一个 `.env` 文件。绝大多数框架（Node.js, Go, PHP）原生或通过库支持 `.env` 读取。同时在执行 SSH 脚本时也将其 `export` 到上下文。
- **方案 B**：仅仅在执行 `pre_deploy` / `post_deploy` 脚本时注入系统环境变量，不生成物理文件。
*攻击点*：如果不生成物理文件，Web 服务（如通过 systemd 或 pm2 启动）如何读取这些变量？必须通过物理 `.env` 落地才能保证应用层读取。

### 疑问 2：存储与加密机制 (Storage & Security)
- **方案 A（极简）**：直接以 JSON 格式附加在现有的 `EnvironmentModel` 中，如 `EnvVars string`。对于 `is_secret=true` 的值，前端展示为 `***`，但在 SQLite 中明文存储。适合内部安全网络。
- **方案 B（严格）**：采用对称加密算法存储在 SQLite 中，主密钥通过启动时的 ENV 注入。
*性能与复杂度对冲*：方案 B 会增加冷启动和管理的复杂度。若仅为内部小团队，方案 A 足够。

### 疑问 3：UI 交互形态 (UI/UX)
- 环境变量数量可能很多，在 `EnvironmentConfig.vue` 应当以键值对表格 (Key-Value Table) 形式展现。
- 支持单个键值对的增删改，并提供 `Secret` 勾选框以切换隐藏/显示。

## 3. 拟定核心设计与对现有功能的影响
1. **数据模型**：在 `EnvironmentModel` 中新增 `EnvVars` 字段（JSON存储），或者新增独立物理表 `EnvironmentVariable` (依赖 GORM AutoMigrate)。**建议使用 `EnvVars text` JSON 字段，侵入性最小，完全向后兼容**。
2. **部署引擎 (`DeployEngine`)**：
   - 增加一步：在 Git pull 后，循环环境配置，通过 SSH 往 `workspacePath`（目标 release 目录）写入 `.env` 文件。
   - 对现有逻辑 0 污染，原流程直接增加写入文件命令即可。

## 4. 确认请求
请用户审批上述方案：
1. 是否同意在物理目录自动生成 `.env` 文件？
2. 是否接受极简的 JSON 明文存储（前端脱敏展示）方案，以换取开发效率和最低的代码侵入性？
