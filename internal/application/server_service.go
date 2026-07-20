package application

import "pdeploy/internal/domain"

type ServerService struct {
	repo domain.ServerRepository
}

func NewServerService(repo domain.ServerRepository) *ServerService {
	return &ServerService{repo: repo}
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
	return s.repo.Delete(id)
}
