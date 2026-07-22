package domain

type ProjectRepository interface {
	Save(p *Project) error
	FindByID(id uint) (*Project, error)
	FindAll() ([]*Project, error)
	Delete(id uint) error
	FindProjectByEnvID(envID uint) (*Project, error)
}

type EnvironmentRepository interface {
	Save(e *Environment) error
	FindByProjectID(projectID uint) ([]*Environment, error)
}

type DeploymentRepository interface {
	Save(d *Deployment) error
	FindByID(id uint) (*Deployment, error)
	FindByEnvID(envID uint) ([]*Deployment, error)
	FindByStatus(status string) ([]*Deployment, error)
}
