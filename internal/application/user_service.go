package application

import (
	"errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/riverisagame/godeploy/internal/domain"
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// @Ref: docs/sps/plans/20260721_v2.5_refactoring_ir.md Task 3.1 | @Date: 2026-07-22
func (s *UserService) CreateUser(username, password, role string) error {
	if role != "admin" && role != "developer" {
		return errors.New("invalid role")
	}
	existing, _ := s.repo.FindByUsername(username)
	if existing != nil {
		return errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}
	return s.repo.Save(user)
}

func (s *UserService) ListUsers() ([]domain.User, error) {
	// Requires UserRepository.FindAll()
	var users []domain.User
	// TODO: We need FindAll method in repo
	if repoWithFindAll, ok := s.repo.(interface{ FindAll() ([]*domain.User, error) }); ok {
		models, err := repoWithFindAll.FindAll()
		if err != nil {
			return nil, err
		}
		for _, m := range models {
			users = append(users, domain.User{
				ID:       m.ID,
				Username: m.Username,
				Role:     m.Role,
			})
		}
	} else {
		return nil, errors.New("FindAll not supported by repository")
	}
	return users, nil
}
