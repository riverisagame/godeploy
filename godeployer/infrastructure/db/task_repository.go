package db

import (
	"deploy/godeployer/domain"
	"gorm.io/gorm"
)

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务资源库
func NewTaskRepository(db *gorm.DB) domain.TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) InsertTask(task *domain.DeployTask) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetTaskByID(id int) (*domain.DeployTask, error) {
	var task domain.DeployTask
	if err := r.db.First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) GetTasksByEnv(projectID, envID string, limit int) ([]domain.DeployTask, error) {
	var tasks []domain.DeployTask
	query := r.db.Where("project_id = ? AND env_id = ?", projectID, envID).Order("id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepository) DeleteTasks(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&domain.DeployTask{}).Error
}

func (r *taskRepository) UpdateTaskStatus(id int, status string) error {
	return r.db.Model(&domain.DeployTask{}).Where("id = ?", id).Update("status", status).Error
}

func (r *taskRepository) GetStalledTasks() ([]domain.DeployTask, error) {
	var tasks []domain.DeployTask
	if err := r.db.Where("status IN ?", []string{"pending", "deploying"}).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepository) UpdateTaskStatusBatch(ids []int, status string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&domain.DeployTask{}).Where("id IN ?", ids).Update("status", status).Error
}

func (r *taskRepository) CountTasksByEnv(projectID, envID string) (int, error) {
	var count int64
	if err := r.db.Model(&domain.DeployTask{}).Where("project_id = ? AND env_id = ?", projectID, envID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

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
