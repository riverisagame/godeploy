package application

import (
	"errors"
	"pdeploy/internal/domain"
)

type ProjectService struct {
	repo domain.ProjectRepository
}

func NewProjectService(repo domain.ProjectRepository) *ProjectService {
	return &ProjectService{
		repo: repo,
	}
}

func (s *ProjectService) CreateProject(name, repoURL string) (*domain.Project, error) {
	project, err := domain.NewProject(name, repoURL)
	if err != nil {
		return nil, err
	}

	err = s.repo.Save(project)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *ProjectService) GetProjects() ([]*domain.Project, error) {
	return s.repo.FindAll()
}

func (s *ProjectService) AddEnvironment(projectID uint, name, branch, deployType string) (*domain.Project, error) {
	project, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	if err := project.AddEnvironment(name, branch, deployType); err != nil {
		return nil, err
	}
	
	if err := s.repo.Save(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) UpdateEnvironment(projectID uint, envName, preDeploy, postDeploy, deployPath string, serverIDs []uint, envVars []domain.EnvVar) (*domain.Project, error) {
	project, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	for _, env := range project.Environments {
		if env.Name == envName {
			env.PreDeploy = preDeploy
			env.PostDeploy = postDeploy
			if deployPath != "" {
				env.DeployPath = deployPath
			}
			if serverIDs != nil {
				env.ServerIDs = serverIDs
			}
			if envVars != nil {
				env.EnvVars = envVars
			}
			break
		}
	}
	
	if err := s.repo.Save(project); err != nil {
		return nil, err
	}
	return project, nil
}

// @Ref: docs/sps/plans/20260721_project_server_edit_ir.md | @Date: 2026-07-21
func (s *ProjectService) UpdateProject(id uint, name, repoURL string) (*domain.Project, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	if name != "" {
		project.Name = name
	}
	if repoURL != "" {
		project.RepoURL = repoURL
	}

	if err := s.repo.Save(project); err != nil {
		return nil, err
	}
	return project, nil
}

// @Ref: docs/sps/plans/20260721_project_server_edit_ir.md | @Date: 2026-07-21
func (s *ProjectService) DeleteProject(id uint) error {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}
	
	// Delegate soft-delete/cascading to the repository
	return s.repo.Delete(id)
}
