package domain

// UserRepository 定义了对 User 的持久化操作接口
type UserRepository interface {
	GetUserByUsername(username string) (*UserResponse, error)
	CreateUser(user *UserResponse, passwordHash string) error
	UpdateUser(user *UserResponse, passwordHash string) error
	GetUsers() ([]UserResponse, error)
	DeleteUser(username string) error
}

// ProjectRepository 定义对项目配置的查询接口。
// Config 本身通过 YAML 管理，此接口提供聚合查询能力。
type ProjectRepository interface {
	GetAllProjects(config *Config) []ProjectSummary
}

// ProjectSummary 项目摘要，用于 API 列表展示。
type ProjectSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TaskRepository 定义了对部署任务的记录持久化操作接口
type TaskRepository interface {
	InsertTask(task *DeployTask) error
	GetTaskByID(id int) (*DeployTask, error)
	GetTasksByEnv(projectID, envID string, limit int) ([]DeployTask, error)
	DeleteTasks(ids []int) error
	UpdateTaskStatus(id int, status DeployStatus) error
	GetStalledTasks() ([]DeployTask, error)
	UpdateTaskStatusBatch(ids []int, status DeployStatus) error
	CountTasksByEnv(projectID, envID string) (int, error)
	GetTasksByEnvAsc(projectID, envID string, limit int) ([]DeployTask, error)
}
