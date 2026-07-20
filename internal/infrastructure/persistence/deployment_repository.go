package persistence

import (
	"pdeploy/internal/domain"
	"gorm.io/gorm"
)

type DeploymentModel struct {
	gorm.Model
	ProjectID   uint
	EnvID       uint   `gorm:"index"`
	UserID      uint   `gorm:"index"`
	CommitHash  string `gorm:"index"`
	Status      string `gorm:"index"`
	Phase       string
	Log         string `gorm:"type:text"`
	ReleaseName string

	Environment EnvironmentModel `gorm:"foreignKey:EnvID;constraint:OnDelete:CASCADE;"`
}

type SqliteDeploymentRepository struct {
	db *gorm.DB
}

func NewSqliteDeploymentRepository(db *gorm.DB) *SqliteDeploymentRepository {
	return &SqliteDeploymentRepository{db: db}
}

func (r *SqliteDeploymentRepository) Save(d *domain.Deployment) error {
	model := &DeploymentModel{
		EnvID:      d.EnvID,
		UserID:     d.UserID,
		CommitHash: d.CommitHash,
		Status:     d.Status,
		Phase:      d.Phase,
		Log:        d.Log,
		ReleaseName: d.ReleaseName,
	}
	if d.ID != 0 {
		model.ID = d.ID
	}
	if err := r.db.Save(model).Error; err != nil {
		return err
	}
	d.ID = model.ID
	return nil
}

func (r *SqliteDeploymentRepository) FindByID(id uint) (*domain.Deployment, error) {
	var model DeploymentModel
	if err := r.db.First(&model, id).Error; err != nil {
		return nil, err
	}
	return &domain.Deployment{
		ID:         model.ID,
		EnvID:      model.EnvID,
		UserID:     model.UserID,
		CommitHash: model.CommitHash,
		Status:     model.Status,
		Phase:      model.Phase,
		Log:        model.Log,
		ReleaseName: model.ReleaseName,
		CreatedAt:  model.CreatedAt,
	}, nil
}

func (r *SqliteDeploymentRepository) FindByEnvID(envID uint) ([]*domain.Deployment, error) {
	var models []DeploymentModel
	if err := r.db.Where("env_id = ?", envID).Order("id desc").Limit(20).Find(&models).Error; err != nil {
		return nil, err
	}
	var res []*domain.Deployment
	for _, m := range models {
		res = append(res, &domain.Deployment{
			ID:         m.ID,
			EnvID:      m.EnvID,
			UserID:     m.UserID,
			CommitHash: m.CommitHash,
			Status:     m.Status,
			Phase:      m.Phase,
			Log:        m.Log,
			ReleaseName: m.ReleaseName,
			CreatedAt:  m.CreatedAt,
		})
	}
	return res, nil
}
