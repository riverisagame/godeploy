package db

import (
	"deploy/godeployer/domain"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户资源库
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByUsername(username string) (*domain.UserResponse, error) {
	var user domain.UserResponse
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CreateUser(user *domain.UserResponse, passwordHash string) error {
	user.PasswordHash = passwordHash
	return r.db.Create(user).Error
}

func (r *userRepository) UpdateUser(user *domain.UserResponse, passwordHash string) error {
	updates := map[string]interface{}{
		"role":                 user.Role,
		"bound_git_authors":    user.BoundGitAuthors,
		"restrict_git_authors": user.RestrictGitAuthors,
		"permitted_projects":   user.PermittedProjects,
	}
	if passwordHash != "" {
		updates["password_hash"] = passwordHash
	}
	return r.db.Model(user).Updates(updates).Error
}

func (r *userRepository) GetUsers() ([]domain.UserResponse, error) {
	var users []domain.UserResponse
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) DeleteUser(username string) error {
	return r.db.Where("username = ?", username).Delete(&domain.UserResponse{}).Error
}
