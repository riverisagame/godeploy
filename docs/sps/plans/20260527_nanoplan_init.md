# TASK-003: GoDeployer 核心后端框架初始化纳米计划

本计划旨在初始化 GoDeployer 后端的核心框架，包括配置加载、数据库建立、多用户 JWT 认证以及事件通知总线的骨架，确保后续能够进行测试驱动开发。

---

## 1. 拟修改/创建文件清单

- **[NEW]** `godeployer/main.go` - 应用程序主入口
- **[NEW]** `godeployer/config.go` - 配置模型与环境变量解析
- **[NEW]** `godeployer/db.go` - SQLite 连接初始化与表迁移
- **[NEW]** `godeployer/auth.go` - JWT 鉴权机制与密码哈希处理
- **[NEW]** `godeployer/notifier.go` - 异步通知事件管道与接口定义

---

## 2. 纳米级任务细分 (代码量限制在 10-20 行/步)

### 2.1 任务 1: 配置文件解析与环境变量过滤
*文件路径*: `godeployer/config.go`

- **[x] [Sub-Task 1.1]** 定义全局配置结构体 `Config` 和 `ProjectConfig`。
  - *改动逻辑*：定义对应 YAML 节点的字段。包含 SQLite 路径、日志路径、项目目录等。
- **[x] [Sub-Task 1.2]** 编写 `ExpandEnvFunc` 辅助函数，实现 YAML 读取后的环境变量求值。
  - *代码设计*：调用 `os.ExpandEnv` 对配置文本进行占位符替换，返回替换后的内容。
- **[x] [Sub-Task 1.3]** 编写 `LoadConfig` 函数，解析 `config.yaml` 主配置并扫描项目配置目录 `projects.d` 加载所有子项目配置。

### 2.2 任务 2: SQLite 数据库迁移与初始化
*文件路径*: `godeployer/db.go`

- **[x] [Sub-Task 2.1]** 建立全局变量 `DB *sql.DB` 并编写 `InitDB` 函数，连接 SQLite。
- **[x] [Sub-Task 2.2]** 编写 DDL 迁移逻辑。新建 `users` 表和 `deploy_tasks` 表，禁止包含 DROP/TRUNCATE 动作，只在不存在时创建。
- **[x] [Sub-Task 2.3]** 编写 `CreateDefaultAdmin`，检查 `users` 表。若为空，根据环境变量 `ADMIN_PASSWORD`（默认密码为 `admin123`）哈希后插入首个 admin 用户。

### 2.3 任务 3: 用户登录及 JWT 鉴权
*文件路径*: `godeployer/auth.go`

- **[x] [Sub-Task 3.1]** 编写 `HashPassword` 与 `CheckPasswordHash` 函数，封装 bcrypt 的哈希与校验。
- **[x] [Sub-Task 3.2]** 编写 `GenerateToken` 函数，根据用户名生成有限期的 JWT Token。
- **[x] [Sub-Task 3.3]** 编写 Gin JWT 鉴权中间件 `AuthMiddleware`，拦截请求、提取 Header 中的 Token 并解析、保存用户信息至 Context。

### 2.4 任务 4: 事件通知总线
*文件路径*: `godeployer/notifier.go`

- **[x] [Sub-Task 4.1]** 定义部署事件类型 `DeployEvent` 及 `Notifier` 接口。
- **[x] [Sub-Task 4.2]** 编写全局事件管理器 `EventBus`，维护事件 Channel 队列和已注册的 Notifier 列表。
- **[x] [Sub-Task 4.3]** 编写后台消费 Goroutine `StartEventConsumer`，监听 Channel 并异步调用所有已配置的通知发送器。

---

## 3. 验证与单元测试规划 (物理零污染)

根据“硬核审计”原则，测试绝不破坏任何物理表和现有数据：
- **测试环境**：每次运行测试时使用内存数据库（连接串使用 `file::memory:?cache=shared`），杜绝污染任何实际的 `.db` 文件。
- **单元测试覆盖**：
  1. `TestConfig_Load`：编写 YAML 字符串测试，测试环境变量 `${TEST_ENV}` 成功被替换。
  2. `TestDB_Init`：验证内存数据库建立，默认 admin 用户插入，且密码哈希可校验。
  3. `TestAuth_JWT`：生成 JWT 令牌后测试鉴权中间件解密正确性。
  4. `TestNotifier_Async`：注入 Mock Notifier，抛送事件并断言通知被异步消费。
