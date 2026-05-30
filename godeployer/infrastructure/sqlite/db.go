package sqlite

import (
	"database/sql"
	"errors"

	"deploy/godeployer/domain"
)

type userRepository struct {
	db *sql.DB
}

// NewUserRepository 创建资源库
func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByUsername(username string) (*domain.UserResponse, error) {
	var user domain.UserResponse
	err := r.db.QueryRow(`
		SELECT id, username, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects 
		FROM users WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.Role, &user.CreatedAt,
		&user.BoundGitAuthors, &user.RestrictGitAuthors, &user.PermittedProjects,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Or a specific error
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CreateUser(user *domain.UserResponse, passwordHash string) error {
	res, err := r.db.Exec(`
		INSERT INTO users (username, password_hash, role, created_at, bound_git_authors, restrict_git_authors, permitted_projects) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, user.Username, passwordHash, user.Role, user.CreatedAt, user.BoundGitAuthors, user.RestrictGitAuthors, user.PermittedProjects)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		user.ID = int(id)
	}
	return nil
}

func (r *userRepository) UpdateUser(user *domain.UserResponse, passwordHash string) error {
	return errors.New("not implemented")
}

func (r *userRepository) GetUsers() ([]domain.UserResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *userRepository) DeleteUser(username string) error {
	return errors.New("not implemented")
}

// Note: CreateUser with password hash isn't in the domain interface yet, we should add it or pass password in another way.
