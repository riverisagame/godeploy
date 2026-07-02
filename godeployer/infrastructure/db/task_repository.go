// ============================================================
// 文件：task_repository.go
// 作用：📋 任务仓库——实现 domain.TaskRepository 接口！
//
// 什么是"仓库"（Repository）？
// 仓库 = 专门用来操作数据库的"中间人"。
// 领域层（domain）定义了"要做什么"（接口），
// 基础设施层（infrastructure）实现"怎么做"（具体 SQL）。
//
// 这个文件负责部署任务（DeployTask）的所有数据库操作：
// - 创建任务（InsertTask）
// - 查询任务（GetTaskByID、GetTasksByEnv）
// - 更新状态（UpdateTaskStatus）
// - 删除任务（DeleteTasks）
// - 批量操作（UpdateTaskStatusBatch、GetStalledTasks）
//
// 所有操作都是通过 GORM 完成的——不用写 SQL，只调用函数！
// ============================================================

package db

import (
	"deploy/godeployer/domain" // 📋 领域接口和实体
	"gorm.io/gorm"             // 🏗️ GORM 数据库工具
)

// taskRepository 结构体——实现了 domain.TaskRepository 接口
// 注意名字是小写开头！小写 = 包内私有，外部不能直接访问
// 只能通过 NewTaskRepository() 函数创建
type taskRepository struct {
	db *gorm.DB // 💾 数据库连接
}

// NewTaskRepository 创建任务资源库
// 参数：GORM 数据库连接
// 返回值：TaskRepository 接口（不是具体类型！面向接口编程）
func NewTaskRepository(db *gorm.DB) domain.TaskRepository {
	return &taskRepository{db: db}
}

// InsertTask 插入一条新任务到数据库
// 就像在 Excel 里新增一行记录
func (r *taskRepository) InsertTask(task *domain.DeployTask) error {
	return r.db.Create(task).Error // GORM 的 Create = INSERT INTO
}

// GetTaskByID 根据 ID 查询某个任务
// 返回 nil, nil = 没有找到（不是错误！）
// 返回 nil, err = 查询出错了
// 返回 &task, nil = 找到了！
func (r *taskRepository) GetTaskByID(id int) (*domain.DeployTask, error) {
	var task domain.DeployTask
	// First = 查询第一条匹配的记录
	// 相当于 SELECT * FROM deploy_tasks WHERE id = ? LIMIT 1
	if err := r.db.First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // ⚠️ 没找到不是错误，返回 nil
		}
		return nil, err // ❌ 真的出错了
	}
	return &task, nil
}

// GetTasksByEnv 查询某个项目+环境的最新 N 条任务
// 按 ID 倒序排列（最新的在前）
func (r *taskRepository) GetTasksByEnv(projectID, envID string, limit int) ([]domain.DeployTask, error) {
	var tasks []domain.DeployTask
	// Where = 筛选条件，Order = 排序，Limit = 限制条数
	query := r.db.Where("project_id = ? AND env_id = ?", projectID, envID).Order("id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// DeleteTasks 批量删除指定 ID 的任务
// 用于清理过期的旧任务记录
func (r *taskRepository) DeleteTasks(ids []int) error {
	if len(ids) == 0 {
		return nil // 空列表 = 什么都不做
	}
	// DELETE FROM deploy_tasks WHERE id IN (?)
	return r.db.Where("id IN ?", ids).Delete(&domain.DeployTask{}).Error
}

// UpdateTaskStatus 更新单个任务的状态
// 比如：pending → deploying → success
func (r *taskRepository) UpdateTaskStatus(id int, status domain.DeployStatus) error {
	// UPDATE deploy_tasks SET status = ? WHERE id = ?
	return r.db.Model(&domain.DeployTask{}).Where("id = ?", id).Update("status", string(status)).Error
}

// GetStalledTasks 获取"卡住"的任务
// 卡住 = 状态是 pending（等待中）或 deploying（部署中），
// 但实际已经没有协程在处理了（程序可能崩溃过）
func (r *taskRepository) GetStalledTasks() ([]domain.DeployTask, error) {
	var tasks []domain.DeployTask
	if err := r.db.Where("status IN ?",
		[]domain.DeployStatus{domain.StatusPending, domain.StatusDeploying},
	).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTaskStatusBatch 批量更新任务状态
// 一次性把多个卡住的任务都标记为失败/取消
func (r *taskRepository) UpdateTaskStatusBatch(ids []int, status domain.DeployStatus) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&domain.DeployTask{}).Where("id IN ?", ids).Update("status", string(status)).Error
}

// CountTasksByEnv 统计某个项目+环境有多少条任务
func (r *taskRepository) CountTasksByEnv(projectID, envID string) (int, error) {
	var count int64
	if err := r.db.Model(&domain.DeployTask{}).
		Where("project_id = ? AND env_id = ?", projectID, envID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetTasksByEnvAsc 按时间正序查询任务（最老的在前）
// 跟 GetTasksByEnv（最新的在前）相反
// 用于查找最老的记录，以便清理
func (r *taskRepository) GetTasksByEnvAsc(projectID, envID string, limit int) ([]domain.DeployTask, error) {
	var tasks []domain.DeployTask
	query := r.db.Where("project_id = ? AND env_id = ?", projectID, envID).Order("id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: What is Object Relational Mapping (ORM)?
//    A: A way to interact with your database using your programming language instead of SQL.
//       For example, db.Create(&task) becomes INSERT INTO deploy_tasks ...
//
// 2. Q: 为什么有些结构体名小写开头（taskRepository）？
//    A: Go 语言中，小写开头 = 包内私有，外部看不见。
//       NewTaskRepository() 返回接口，外部只能通过接口操作~
//
// 中级：
// 3. Q: GetTaskByID 为什么返回 nil, nil 表示"没找到"？
//    A: Go 中常用这种做法——"没找到"不是错误，只是一种正常情况。
//       调用者通过检查返回值是否为 nil 来判断~
//
// 4. Q: 为什么更新状态时要把 DeployStatus 转成 string？
//    A: GORM 默认存 string，DeployStatus 是自定义类型，
//       转成 string 确保数据库能正确存储~
//
// 高级：
// 5. Q: Repository 模式和直接使用 GORM 有什么不同？
//    A: Repository 是"抽象层"——领域层不知到 GORM 存在！
//       如果以后从 GORM 换成别的 ORM，只需改这个文件。
//       领域层代码完全不用动~（依赖反转原则）
// ============================================================
