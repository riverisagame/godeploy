package api

import (
	"database/sql"

	"deploy/godeployer/application"
	"deploy/godeployer/domain"

	"github.com/gin-gonic/gin"
)

// @Ref: docs/sps/plans/20260530_sqlite_purego_and_performance_gate_plan.md | @Date: 2026-05-30
var diffSemaphore = make(chan struct{}, 5)

type APIHandler struct {
	config   *domain.Config
	db       *sql.DB
	taskRepo domain.TaskRepository
	executor domain.NodeExecutor
	engine   *application.DeployEngine
}

func SetupRoutes(config *domain.Config, db *sql.DB, taskRepo domain.TaskRepository, engine *application.DeployEngine) *gin.Engine {
	return SetupRoutesWithExecutor(config, db, taskRepo, nil, engine)
}

// SetupRoutesWithExecutor 允许传入模拟 Executor 以支持测试驱动开发 (TDD)
// @Ref: docs/sps/plans/20260527_nanoplan_tdd_enhanced.md | @Date: 2026-05-27
func SetupRoutesWithExecutor(config *domain.Config, db *sql.DB, taskRepo domain.TaskRepository, executor domain.NodeExecutor, engine *application.DeployEngine) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if config.Global.DiffMaxSizeKB <= 0 {
		config.Global.DiffMaxSizeKB = 5120
	}
	if config.Global.DiskMinSpaceMB <= 0 {
		config.Global.DiskMinSpaceMB = 500
	}

	handler := &APIHandler{
		config:   config,
		db:       db,
		taskRepo: taskRepo,
		executor: executor,
		engine:   engine,
	}

	// 开放接口
	r.POST("/api/login", handler.HandleLogin)
	r.POST("/api/webhooks/github/:project_id/:env_id", handler.HandleGithubWebhook)

	// WebSocket 路由
	r.GET("/api/ws/tasks/:id/log", handler.HandleWSLog)

	// 受保护接口
	protected := r.Group("/api")
	protected.Use(application.AuthMiddleware(config.Global.JWTSecret))
	{
		viewerGrp := protected.Group("/")
		viewerGrp.Use(application.RoleMiddleware("admin", "deployer", "viewer"))
		{
			viewerGrp.GET("/projects", handler.HandleGetProjects)
			viewerGrp.GET("/projects/:id/refs", handler.HandleGetProjectRefs)
			viewerGrp.GET("/projects/:id/commits", handler.HandleGetProjectCommits)
			viewerGrp.GET("/projects/:id/preview_diff", handler.HandleGetProjectPreviewDiff)
			viewerGrp.GET("/tasks", handler.HandleGetTasks)
			viewerGrp.GET("/tasks/:id", handler.HandleGetTaskDetail)
			viewerGrp.GET("/tasks/:id/log", handler.HandleGetTaskLog)
			viewerGrp.GET("/tasks/:id/diff", handler.HandleGetTaskDiff)
		}

		adminGrp := protected.Group("/")
		adminGrp.Use(application.RoleMiddleware("admin"))
		{
			adminGrp.GET("/users", handler.HandleGetUsers)
			adminGrp.POST("/users", handler.HandleCreateUser)
			adminGrp.PUT("/users/:username", handler.HandleUpdateUser)
			adminGrp.DELETE("/users/:username", handler.HandleDeleteUser)
			adminGrp.GET("/users/:username/git_binding", handler.HandleGetUserGitBinding)
			adminGrp.PUT("/users/:username/git_binding", handler.HandleUpdateUserGitBinding)
			adminGrp.PUT("/users/:username/permissions", handler.HandleUpdateUserPermissions)
			adminGrp.POST("/system/prune", handler.HandleSystemPrune)
		}

		deployerGrp := protected.Group("/")
		deployerGrp.Use(application.RoleMiddleware("admin", "deployer"))
		{
			deployerGrp.POST("/tasks", handler.HandleCreateTask)
			deployerGrp.POST("/tasks/:id/rollback", handler.HandleTriggerRollback)
		}
	}

	return r
}
