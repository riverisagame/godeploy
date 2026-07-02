// ============================================================
// 文件：repository.go
// 作用：📚 定义"数据存储"的接口！
//
// Repository（仓库）模式是 DDD（领域驱动设计）中最重要的模式之一。
// 它把"数据怎么存"的问题抽象出来：
//
// - 接口（Interface）：定义"能做什么操作"，比如"保存用户"、"查询任务"
// - 实现（Implementation）：具体怎么存（SQLite、MySQL、甚至文件），由基础设施层负责
//
// 就像一个"外卖柜"：
// - 你可以放进去、拿出来、查询里面有什么
// - 但你不用管柜子内部是怎么运作的（弹簧？电子锁？）
//
// 这里定义了 3 个仓库接口：
// 1. UserRepository：操作用户数据（增删改查）
// 2. ProjectRepository：查询项目配置
// 3. TaskRepository：操作部署任务数据
// ============================================================

package domain

// ============================================================
// 👤 UserRepository：用户仓库接口
//
// 定义了"怎么操作用户数据"的方法集合。
// 包括：查一个用户、查全部用户、创建、更新、删除。
//
// 为什么叫 Repository 不叫 DAO？
// DAO（Data Access Object）只关注"读写数据"，
// Repository 更关注"怎么从业务角度操作数据集合"，
// 这个概念区别有点微妙，简单理解：
// Repository ≈ DAO + 一些业务相关的查询~
// ============================================================

// UserRepository 定义了对 User 的持久化操作接口
type UserRepository interface {
	// GetUserByUsername 根据用户名查找用户
	// 返回用户信息，如果找不到返回 nil, error
	GetUserByUsername(username string) (*UserResponse, error)

	// CreateUser 创建一个新用户
	// 需要用户名、角色、密码哈希等信息
	CreateUser(user *UserResponse, passwordHash string) error

	// UpdateUser 更新用户信息（比如改角色、改权限）
	UpdateUser(user *UserResponse, passwordHash string) error

	// GetUsers 获取所有用户的列表
	GetUsers() ([]UserResponse, error)

	// DeleteUser 删除指定用户名的用户
	DeleteUser(username string) error
}

// ============================================================
// 📁 ProjectRepository：项目配置仓库接口
//
// 项目的配置存在 YAML 文件里，而不是数据库里。
// 但为了方便统一查询，我们通过这个接口来访问项目配置。
//
// 目前只有一个方法：获取所有项目的摘要列表。
// 以后可以加：按 ID 查询、按环境查询等~
// ============================================================

// ProjectRepository 定义对项目配置的查询接口。
// Config 本身通过 YAML 管理，此接口提供聚合查询能力。
type ProjectRepository interface {
	// GetAllProjects 获取所有项目的摘要信息
	GetAllProjects(config *Config) []ProjectSummary
}

// ProjectSummary 项目摘要，用于 API 列表展示。
// 不像 ProjectConfig 那么详细（没有服务器地址、环境配置等敏感信息），
// 只包含前端展示需要的基本信息。
type ProjectSummary struct {
	ID          string `json:"id"`          // 🆔 项目 ID
	Name        string `json:"name"`        // 📛 项目名称
	Description string `json:"description"` // 📝 项目描述（暂时可能为空）
}

// ============================================================
// 📋 TaskRepository：部署任务仓库接口
//
// 这是最核心的仓库接口！部署任务是最常操作的数据。
// 每次部署都要：
// 1. 创建任务（InsertTask）
// 2. 更新状态（UpdateTaskStatus）
// 3. 查询历史（GetTasksByEnv）
// 4. 清理旧数据（DeleteTasks）
// ============================================================

// TaskRepository 定义了对部署任务的记录持久化操作接口
type TaskRepository interface {
	// InsertTask 插入一条新的部署任务记录
	InsertTask(task *DeployTask) error

	// GetTaskByID 根据 ID 查询任务详情
	GetTaskByID(id int) (*DeployTask, error)

	// GetTasksByEnv 查询某个项目指定环境的最新 N 条任务
	// limit 参数：要返回多少条（比如最新的 10 条）
	GetTasksByEnv(projectID, envID string, limit int) ([]DeployTask, error)

	// DeleteTasks 批量删除指定 ID 的任务
	DeleteTasks(ids []int) error

	// UpdateTaskStatus 更新单个任务的状态
	// 比如：pending → deploying → success
	UpdateTaskStatus(id int, status DeployStatus) error

	// GetStalledTasks 获取"卡住"的任务
	// 比如部署中但很久没更新的任务（可能是程序崩溃了）
	GetStalledTasks() ([]DeployTask, error)

	// UpdateTaskStatusBatch 批量更新任务状态
	// 用于一次性处理多个卡住的任务
	UpdateTaskStatusBatch(ids []int, status DeployStatus) error

	// CountTasksByEnv 统计某个项目环境下有多少条任务记录
	CountTasksByEnv(projectID, envID string) (int, error)

	// GetTasksByEnvAsc 按时间正序查询任务（最旧的在前）
	// 跟 GetTasksByEnv（最新的在前）相反
	// 用于清理最旧的那些任务
	GetTasksByEnvAsc(projectID, envID string, limit int) ([]DeployTask, error)
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 什么是"接口"（Interface）？
//    A: 一个方法列表的约定。就像"能飞"这个接口——飞机能飞、鸟能飞、
//       但它们的实现完全不同。接口只关心"做什么"，不关心"怎么做"~
//
// 2. Q: 为什么要有仓库（Repository）？
//    A: 拆分管数据的代码！你的模型（实体）不用知道自己怎么存数据库，
//       仓库帮你搞定一切，就像你只管点餐，不用管厨房怎么做~
//
// 中级（面试常考）：
// 3. Q: TaskRepository 的方法命名有什么规律？
//    A: ByXxx 表示按什么条件查询（ByEnv = 按环境查）
//       Asc/无后缀 表示排序方式（Asc = 正序，默认倒序）
//       Batch 表示批量操作（Batch = 一次处理多条）
//
// 4. Q: 把 UserRepository 和 TaskRepository 分开而不是合并的原因？
//    A: 单一职责原则！每个接口只负责一种"聚合根"的操作。
//       User 和 Task 是两个不同的业务概念，分开更清晰~
//
// 高级（架构师级别）：
// 5. Q: 为什么 ProjectRepository 和 Config 是分开的？
//    A: 配置的"存储方式"和"使用方式"不同！Config 来自 YAML 文件，
//       但展示给前端需要的是摘要信息。接口在这里起到"适配"作用——
//       把原始的 YAML 数据转换成前端需要的格式~
//
// 6. Q: GetStalledTasks 是做什么的？
//    A: 应对"系统崩溃"的兜底机制！如果部署过程中程序突然挂了，
//       数据库里的任务状态还是"部署中"。系统重启后，通过这个函数
//       找出这些"卡住"的任务，把它们标记为失败~
// ============================================================
