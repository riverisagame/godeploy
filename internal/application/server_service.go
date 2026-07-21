package application

import (
	"errors"
	"github.com/riverisagame/godeploy/internal/domain"
)

type ServerService struct {
	repo        domain.ServerRepository
	projectRepo domain.ProjectRepository
}

func NewServerService(repo domain.ServerRepository, projectRepo domain.ProjectRepository) *ServerService {
	return &ServerService{
		repo:        repo,
		projectRepo: projectRepo,
	}
}

func (s *ServerService) ListServers() ([]*domain.Server, error) {
	return s.repo.FindAll()
}

func (s *ServerService) CreateServer(name, ip string, port int, user, keyPath string) (*domain.Server, error) {
	server, err := domain.NewServer(name, ip, port, user, keyPath)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *ServerService) GetServerByID(id uint) (*domain.Server, error) {
	return s.repo.FindByID(id)
}

func (s *ServerService) DeleteServer(id uint) error {
	// 级联清理所有环境中的该 ServerID
	// @Ref: docs/sps/plans/20260721_phase4_ir.md Task 4.1 | @Date: 2026-07-21
	projects, err := s.projectRepo.FindAll()
	if err == nil {
		for _, prj := range projects {
			changed := false
			for _, env := range prj.Environments {
				newServerIDs := make([]uint, 0)
				for _, sid := range env.ServerIDs {
					if sid != id {
						newServerIDs = append(newServerIDs, sid)
					}
				}
				if len(newServerIDs) != len(env.ServerIDs) {
					env.ServerIDs = newServerIDs
					changed = true
				}
			}
			if changed {
				s.projectRepo.Save(prj)
			}
		}
	}

	return s.repo.Delete(id)
}

// @Ref: docs/sps/plans/20260721_project_server_edit_ir.md | @Date: 2026-07-21
func (s *ServerService) UpdateServer(id uint, name, ip string, port int, user, keyPath string) (*domain.Server, error) {
	server, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, errors.New("server not found")
	}

	if name != "" {
		server.Name = name
	}
	if ip != "" {
		server.IP = ip
	}
	if port > 0 {
		server.Port = port
	}
	if user != "" {
		server.User = user
	}
	if keyPath != "" {
		server.KeyPath = keyPath
	}

	if err := s.repo.Save(server); err != nil {
		return nil, err
	}
	return server, nil
}
