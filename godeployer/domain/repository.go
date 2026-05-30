package domain

// UserRepository 定义了对 User 的持久化操作接口
type UserRepository interface {
	GetUserByUsername(username string) (*UserResponse, error)
	CreateUser(user *UserResponse, passwordHash string) error
	UpdateUser(user *UserResponse, passwordHash string) error
	GetUsers() ([]UserResponse, error)
	DeleteUser(username string) error
}

// ProjectRepository 定义了对项目部署状态的持久化操作接口
// 注意 Config 本身是通过 YAML 管理的，但如果涉及 SQLite 中的附加信息，可在此定义。
type ProjectRepository interface {
}

// TaskRepository 定义了对部署任务的记录持久化操作接口
type TaskRepository interface {
	InsertTask(task *DeployTask) error
	GetTaskByID(id int) (*DeployTask, error)
	GetTasksByEnv(projectID, envID string, limit int) ([]DeployTask, error)
	DeleteTasks(ids []int) error
	UpdateTaskStatus(id int, status string) error
	GetStalledTasks() ([]DeployTask, error)
	UpdateTaskStatusBatch(ids []int, status string) error
	CountTasksByEnv(projectID, envID string) (int, error)
	GetTasksByEnvAsc(projectID, envID string, limit int) ([]DeployTask, error)
}
