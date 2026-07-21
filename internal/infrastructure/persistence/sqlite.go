package persistence

import (
	"encoding/json"
	"pdeploy/internal/domain"
	"gorm.io/gorm"
)

// Persistence Models (Infrastructure Layer only)
type ProjectModel struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"uniqueIndex"`
	RepoURL       string
	KeepReleases  int
	WebhookSecret string
	Environments  []EnvironmentModel `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE;"`
}

type EnvironmentModel struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectID  uint   `gorm:"uniqueIndex:idx_project_env_name;index"`
	Name       string `gorm:"uniqueIndex:idx_project_env_name"`
	Branch       string
	DeployType   string
	BuildCommand string
	PreDeploy    string
	PostDeploy   string
	SharedDirs   string
	SharedFiles  string
	ServerIDs     string // JSON 序列化的 []uint
	DeployPath    string
	EnvVars       string // JSON 序列化的 []domain.EnvVar
	NotifyWebhook string
}

type SqliteProjectRepository struct {
	db *gorm.DB
}

func NewSqliteProjectRepository(db *gorm.DB) *SqliteProjectRepository {
	return &SqliteProjectRepository{
		db: db,
	}
}

func toDomainProject(pm *ProjectModel) *domain.Project {
	p := &domain.Project{
		ID:            pm.ID,
		Name:          pm.Name,
		RepoURL:       pm.RepoURL,
		KeepReleases:  pm.KeepReleases,
		WebhookSecret: pm.WebhookSecret,
		Environments:  make([]*domain.Environment, 0),
	}
	for _, em := range pm.Environments {
		var serverIDs []uint
		if em.ServerIDs != "" {
			json.Unmarshal([]byte(em.ServerIDs), &serverIDs)
		}
		if serverIDs == nil {
			serverIDs = make([]uint, 0)
		}
		var envVars []domain.EnvVar
		if em.EnvVars != "" {
			json.Unmarshal([]byte(em.EnvVars), &envVars)
		}
		if envVars == nil {
			envVars = make([]domain.EnvVar, 0)
		}
		p.Environments = append(p.Environments, &domain.Environment{
			ID:            em.ID,
			Name:          em.Name,
			Branch:        em.Branch,
			DeployType:    em.DeployType,
			BuildCommand:  em.BuildCommand,
			PreDeploy:     em.PreDeploy,
			PostDeploy:    em.PostDeploy,
			SharedDirs:    em.SharedDirs,
			SharedFiles:   em.SharedFiles,
			ServerIDs:     serverIDs,
			DeployPath:    em.DeployPath,
			EnvVars:       envVars,
			NotifyWebhook: em.NotifyWebhook,
		})
	}
	return p
}

func toProjectModel(p *domain.Project) *ProjectModel {
	pm := &ProjectModel{
		ID:            p.ID,
		Name:          p.Name,
		RepoURL:       p.RepoURL,
		KeepReleases:  p.KeepReleases,
		WebhookSecret: p.WebhookSecret,
		Environments:  make([]EnvironmentModel, 0),
	}
	for _, env := range p.Environments {
		srvJSON, _ := json.Marshal(env.ServerIDs)
		envVarsJSON, _ := json.Marshal(env.EnvVars)
		pm.Environments = append(pm.Environments, EnvironmentModel{
			ID:            env.ID,
			Name:          env.Name,
			Branch:        env.Branch,
			DeployType:    env.DeployType,
			BuildCommand:  env.BuildCommand,
			PreDeploy:     env.PreDeploy,
			PostDeploy:    env.PostDeploy,
			SharedDirs:    env.SharedDirs,
			SharedFiles:   env.SharedFiles,
			ServerIDs:     string(srvJSON),
			DeployPath:    env.DeployPath,
			EnvVars:       string(envVarsJSON),
			NotifyWebhook: env.NotifyWebhook,
		})
	}
	return pm
}

func (r *SqliteProjectRepository) Save(p *domain.Project) error {
	pm := toProjectModel(p)
	err := r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(pm).Error
	if err == nil {
		p.ID = pm.ID // populate ID back to domain entity
	}
	return err
}

func (r *SqliteProjectRepository) FindByID(id uint) (*domain.Project, error) {
	var pm ProjectModel
	err := r.db.Preload("Environments").First(&pm, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return toDomainProject(&pm), nil
}

func (r *SqliteProjectRepository) FindAll() ([]*domain.Project, error) {
	var pModels []ProjectModel
	if err := r.db.Preload("Environments").Find(&pModels).Error; err != nil {
		return nil, err
	}

	var projects []*domain.Project
	for _, pm := range pModels {
		projects = append(projects, toDomainProject(&pm))
	}
	return projects, nil
}

// @Ref: docs/sps/plans/20260721_project_server_edit_ir.md | @Date: 2026-07-21
func (r *SqliteProjectRepository) Delete(id uint) error {
	// GORM's OnDelete:CASCADE on ProjectModel.Environments will handle associated records
	return r.db.Delete(&ProjectModel{}, id).Error
}
