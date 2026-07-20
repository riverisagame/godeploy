package application

import (
	"errors"
	"pdeploy/internal/domain"
)

type DeployService struct {
	repo        domain.DeploymentRepository
	projectRepo domain.ProjectRepository
	gitClient   GitClient
}

func NewDeployService(repo domain.DeploymentRepository, projectRepo domain.ProjectRepository, gitClient GitClient) *DeployService {
	return &DeployService{
		repo:        repo,
		projectRepo: projectRepo,
		gitClient:   gitClient,
	}
}

func (s *DeployService) TriggerDeploy(envID, userID uint, commitHash string) (*domain.Deployment, error) {
	d, err := domain.NewDeployment(envID, userID, commitHash)
	if err != nil {
		return nil, err
	}

	err = s.repo.Save(d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (s *DeployService) CompleteDeploy(id uint, success bool, log string, releaseName string) error {
	d, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("deployment not found")
	}

	if success {
		// @Ref: docs/sps/plans/20260720_ui_rollback_history_ir.md | @Date: 2026-07-20
		d.MarkSuccess(log, releaseName)
	} else {
		d.MarkFailed(log)
	}

	return s.repo.Save(d)
}

func (s *DeployService) GetDeploymentsByEnv(envID uint) ([]*domain.Deployment, error) {
	return s.repo.FindByEnvID(envID)
}

func (s *DeployService) GetEnvironmentDiff(projectID uint, envName string) ([]domain.CommitInfo, error) {
	proj, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, errors.New("project not found")
	}

	var env *domain.Environment
	for _, e := range proj.Environments {
		if e.Name == envName {
			env = e
			break
		}
	}
	if env == nil {
		return nil, errors.New("environment not found")
	}

	// 查找最后一次成功的部署
	deps, err := s.repo.FindByEnvID(env.ID) 
	if err != nil {
		return nil, err
	}

	var fromCommit string
	for i := len(deps) - 1; i >= 0; i-- {
		if deps[i].Status == "success" {
			fromCommit = deps[i].CommitHash
			break
		}
	}

	// Fetch diff
	return s.gitClient.FetchAndGetCommits(proj.RepoURL, env.Branch, proj.Name, fromCommit)
}
