package application

import (
	"errors"
	"pdeploy/internal/domain"
)

type DeployService struct {
	repo domain.DeploymentRepository
}

func NewDeployService(repo domain.DeploymentRepository) *DeployService {
	return &DeployService{
		repo: repo,
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

func (s *DeployService) CompleteDeploy(id uint, success bool, log string) error {
	d, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("deployment not found")
	}

	if success {
		d.MarkSuccess(log)
	} else {
		d.MarkFailed(log)
	}

	return s.repo.Save(d)
}
