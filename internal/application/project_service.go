package application

import (
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
	
	if err := project.AddEnvironment(name, branch, deployType); err != nil {
		return nil, err
	}
	
	if err := s.repo.Save(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) UpdateEnvironment(projectID uint, envName, preDeploy, postDeploy, deployPath string, serverIDs []uint) (*domain.Project, error) {
	project, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, err
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
			break
		}
	}
	
	if err := s.repo.Save(project); err != nil {
		return nil, err
	}
	return project, nil
}
