package application

import (
	"context"
	"github.com/riverisagame/godeploy/internal/domain"
)

type DeployScheduler struct {
	repo        domain.DeploymentRepository
	projectRepo domain.ProjectRepository
	engine      *DeployEngine
	queue       chan uint
}

func NewDeployScheduler(repo domain.DeploymentRepository, projectRepo domain.ProjectRepository, engine *DeployEngine) *DeployScheduler {
	return &DeployScheduler{
		repo:        repo,
		projectRepo: projectRepo,
		engine:      engine,
		queue:       make(chan uint, 100),
	}
}

func (s *DeployScheduler) Recover() {
	// @Ref: docs/sps/plans/20260722_v3.0_async_ir.md | @Date: 2026-07-22
	deployments, err := s.repo.FindByStatus("running")
	if err != nil {
		return
	}

	for _, d := range deployments {
		d.MarkFailed("deployment crashed due to application restart or scheduler interruption.")
		s.repo.Save(d)
	}
}

func (s *DeployScheduler) Start(ctx context.Context) {
	// @Ref: docs/sps/plans/20260722_v3.0_async_ir.md | @Date: 2026-07-22
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case deployID := <-s.queue:
				s.processDeployment(deployID)
			}
		}
	}()
}

func (s *DeployScheduler) processDeployment(deployID uint) {
	d, err := s.repo.FindByID(deployID)
	if err != nil || d == nil {
		return
	}

	p, err := s.projectRepo.FindProjectByEnvID(d.EnvID)
	if err != nil || p == nil {
		d.MarkFailed("project or environment not found")
		s.repo.Save(d)
		return
	}

	var env *domain.Environment
	for _, e := range p.Environments {
		if e.ID == d.EnvID {
			env = e
			break
		}
	}
	if env == nil {
		d.MarkFailed("environment not found in project")
		s.repo.Save(d)
		return
	}

	// Execute the deployment or rollback
	if s.engine != nil {
		d.SetPhase("running")
		s.repo.Save(d)
		if len(d.CommitHash) > 12 && d.CommitHash[:12] == "ROLLBACK_TO_" {
			targetRelease := d.CommitHash[12:]
			s.engine.Rollback(d, env, targetRelease)
		} else {
			s.engine.StartDeploy(d, p, env) 
		}
	}
}

func (s *DeployScheduler) Notify(deployID uint) {
	select {
	case s.queue <- deployID:
	default:
		// Queue full
	}
}
