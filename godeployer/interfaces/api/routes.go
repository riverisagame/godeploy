// ============================================================
// 文件：routes.go
// 作用：🗺️ 路由地图——所有 API 接口的"交通指引"！
//
// 这个文件定义了所有 HTTP 接口的路径和对应的处理函数。
// 就像一张地图：/api/login → 登录处理函数
//
// 还定义了不同角色的权限：
// - admin（管理员）：什么都能干
// - deployer（部署者）：可以部署和查看
// - viewer（查看者）：只能看，不能操作
//
// 给初二小白的比喻：
// 路由就像学校的指示牌🏫：
// - 大门（/api/login）→ 任何人都能进
// - 教室（/api/projects）→ 学生和老师都能进
// - 校长室（/api/users）→ 只有校长能进
// - 机房（/api/tasks POST）→ 管理员和运维能进
// ============================================================

package api

import (
	"database/sql"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"

	"github.com/gin-gonic/gin"
)

// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
// diffSemaphore 用于控制并发 diff 请求的数量（最多 5 个）
var diffSemaphore = make(chan struct{}, 5)

// APIHandler 结构体——所有 API 处理函数的"工具包"
// 它持有所有需要的依赖：
// - config：系统配置（项目列表、全局设置）
// - db：数据库连接（用户、任务等数据）
// - taskRepo：任务仓库（操作部署任务）
// - executor：节点执行器（操作远程服务器）
// - engine：部署引擎（执行部署）
type APIHandler struct {
	config   *domain.Config       // ⚙️ 系统配置
	db       *sql.DB              // 💾 数据库连接
	taskRepo domain.TaskRepository // 📋 任务仓库
	executor domain.NodeExecutor   // 🖥️ 远程执行器
	engine   *application.DeployEngine // 🏭 部署引擎
}

// SetupRoutes 创建路由——这是给正常启动用的
// 参数：config（配置）、db（数据库）、taskRepo（任务仓库）、engine（部署引擎）
// 返回：Gin 路由器
func SetupRoutes(config *domain.Config, db *sql.DB, taskRepo domain.TaskRepository, engine *application.DeployEngine) *gin.Engine {
	// executor 传 nil，内部会用默认的 SSH 实现
	return SetupRoutesWithExecutor(config, db, taskRepo, nil, engine)
}

// SetupRoutesWithExecutor 创建路由——这是给测试用的（可以注入模拟的 Executor）
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func SetupRoutesWithExecutor(config *domain.Config, db *sql.DB,
	taskRepo domain.TaskRepository, executor domain.NodeExecutor,
	engine *application.DeployEngine) *gin.Engine {

	r := gin.New()
	// 使用 Gin 默认的日志和恢复中间件
	r.Use(gin.Logger(), gin.Recovery())

	// 设置默认的 diff 大小限制
	if config.Global.DiffMaxSizeKB <= 0 {
		config.Global.DiffMaxSizeKB = 5120 // 默认 5MB
	}
	if config.Global.DiskMinSpaceMB <= 0 {
		config.Global.DiskMinSpaceMB = 500 // 默认 500MB 空间
	}

	// 创建 API 处理器
	handler := &APIHandler{
		config:   config,
		db:       db,
		taskRepo: taskRepo,
		executor: executor,
		engine:   engine,
	}

	// ============================================================
	// 🌐 公开接口（不需要登录）
	// ============================================================

	// POST /api/login — 用户登录，获取 JWT 令牌
	r.POST("/api/login", handler.HandleLogin)

	// POST /api/webhooks/github/:project_id/:env_id — GitHub Webhook 自动部署
	// 不需要鉴权（通过 HMAC 签名验证身份）
	r.POST("/api/webhooks/github/:project_id/:env_id", handler.HandleGithubWebhook)

	// GET /api/ws/tasks/:id/log — WebSocket 实时日志推送
	r.GET("/api/ws/tasks/:id/log", handler.HandleWSLog)

	// ============================================================
	// 🔐 受保护接口（需要 JWT 令牌）
	// ============================================================
	protected := r.Group("/api")
	protected.Use(application.AuthMiddleware(config.Global.JWTSecret))
	{
		// 📖 查看者及以上角色可以访问（viewer, deployer, admin）
		viewerGrp := protected.Group("/")
		viewerGrp.Use(application.RoleMiddleware("admin", "deployer", "viewer"))
		{
			viewerGrp.GET("/projects", handler.HandleGetProjects)                     // 📁 项目列表
			viewerGrp.GET("/projects/:id/refs", handler.HandleGetProjectRefs)         // 🌿 项目分支/标签
			viewerGrp.GET("/projects/:id/commits", handler.HandleGetProjectCommits)   // 📜 项目提交记录
			viewerGrp.GET("/projects/:id/preview_diff", handler.HandleGetProjectPreviewDiff) // 🔍 部署前预览 diff
			viewerGrp.GET("/tasks", handler.HandleGetTasks)                           // 📋 任务列表
			viewerGrp.GET("/tasks/:id", handler.HandleGetTaskDetail)                  // 📋 任务详情
			viewerGrp.GET("/tasks/:id/log", handler.HandleGetTaskLog)                 // 📝 任务日志
			viewerGrp.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)               // 🔍 任务 diff
		}

		// 👑 管理员专用接口
		adminGrp := protected.Group("/")
		adminGrp.Use(application.RoleMiddleware("admin"))
		{
			adminGrp.GET("/users", handler.HandleGetUsers)                     // 👤 用户列表
			adminGrp.POST("/users", handler.HandleCreateUser)                  // ➕ 创建用户
			adminGrp.PUT("/users/:username", handler.HandleUpdateUser)         // ✏️ 更新用户
			adminGrp.DELETE("/users/:username", handler.HandleDeleteUser)      // 🗑️ 删除用户
			adminGrp.GET("/users/:username/git_binding", handler.HandleGetUserGitBinding)    // 🔗 Git 绑定查看
			adminGrp.PUT("/users/:username/git_binding", handler.HandleUpdateUserGitBinding) // 🔗 Git 绑定更新
			adminGrp.PUT("/users/:username/permissions", handler.HandleUpdateUserPermissions) // 🔐 权限更新
			adminGrp.POST("/system/prune", handler.HandleSystemPrune)          // 🧹 系统清理
		}

		// 🚀 部署者及以上角色可以访问（deployer, admin）
		deployerGrp := protected.Group("/")
		deployerGrp.Use(application.RoleMiddleware("admin", "deployer"))
		{
			deployerGrp.POST("/tasks", handler.HandleCreateTask)              // ➕ 创建部署任务
			deployerGrp.POST("/tasks/:id/rollback", handler.HandleTriggerRollback) // ↩️ 触发回滚
		}
	}

	return r
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是 API 路由？
//    A: URL 路径和处理函数的对应关系。
//       就像外卖地址和送餐员的对应关系~
//
// 2. Q: GET、POST、PUT、DELETE 有什么区别？
//    A: GET = 查（看看有什么）
//       POST = 增（创建一个新的）
//       PUT = 改（更新现有的）
//       DELETE = 删（删除）
//       这就是"增删改查"！
//
// 中级：
// 3. Q: 为什么路由要分组？
//    A: 方便统一加中间件！同一组路由共享同一个权限检查，
//       不用在每个 handler 里重复写权限判断代码~
//
// 4. Q: 什么是"中间件"（Middleware）？
//    A: 在请求到达实际处理函数前先经过的一道"关卡"。
//       AuthMiddleware = 检查有没有登录
//       RoleMiddleware = 检查有没有权限
//       gin.Logger = 记录日志
//       gin.Recovery = 捕获 panic 防止程序崩溃
//
// 高级：
// 5. Q: 为什么 WebSocket 路由（/api/ws/...）放在公开路由里？
//    A: WebSocket 的认证方式跟 HTTP 不同！
//       WebSocket 建立连接后通过第一条消息来鉴权，
//       所以不能使用 HTTP 的 AuthMiddleware~
// ============================================================
