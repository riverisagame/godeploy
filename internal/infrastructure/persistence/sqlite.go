package persistence

import (
	"pdeploy/internal/domain"
	"gorm.io/gorm"
)

// Persistence Models (Infrastructure Layer only)
type ProjectModel struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"uniqueIndex"`
	RepoURL      string
	KeepReleases int
	Environments []EnvironmentModel `gorm:"foreignKey:ProjectID"`
}

type EnvironmentModel struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectID  uint   `gorm:"index"`
	Name       string
	Branch     string
	DeployType string
	PreDeploy  string
	PostDeploy string
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
		ID:           pm.ID,
		Name:         pm.Name,
		RepoURL:      pm.RepoURL,
		KeepReleases: pm.KeepReleases,
		Environments: make([]*domain.Environment, 0),
	}
	for _, em := range pm.Environments {
		p.Environments = append(p.Environments, &domain.Environment{
			Name:       em.Name,
			Branch:     em.Branch,
			DeployType: em.DeployType,
			PreDeploy:  em.PreDeploy,
			PostDeploy: em.PostDeploy,
		})
	}
	return p
}

func toProjectModel(p *domain.Project) *ProjectModel {
	pm := &ProjectModel{
		ID:           p.ID,
		Name:         p.Name,
		RepoURL:      p.RepoURL,
		KeepReleases: p.KeepReleases,
		Environments: make([]EnvironmentModel, 0),
	}
	for _, env := range p.Environments {
		pm.Environments = append(pm.Environments, EnvironmentModel{
			Name:       env.Name,
			Branch:     env.Branch,
			DeployType: env.DeployType,
			PreDeploy:  env.PreDeploy,
			PostDeploy: env.PostDeploy,
		})
	}
	return pm
}

func (r *SqliteProjectRepository) Save(p *domain.Project) error {
	pm := toProjectModel(p)
	err := r.db.Save(pm).Error
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
	var pms []ProjectModel
	err := r.db.Preload("Environments").Find(&pms).Error
	if err != nil {
		return nil, err
	}

	var projects []*domain.Project
	for i := range pms {
		projects = append(projects, toDomainProject(&pms[i]))
	}
	return projects, nil
}
