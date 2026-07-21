package persistence

import (
	"github.com/riverisagame/godeploy/internal/domain"
	"gorm.io/gorm"
)

type UserModel struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	Role         string
}

type SqliteUserRepository struct {
	db *gorm.DB
}

func NewSqliteUserRepository(db *gorm.DB) *SqliteUserRepository {
	return &SqliteUserRepository{db: db}
}

func (r *SqliteUserRepository) Save(u *domain.User) error {
	m := &UserModel{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
	}
	err := r.db.Save(m).Error
	if err == nil {
		u.ID = m.ID
	}
	return err
}

func (r *SqliteUserRepository) FindByUsername(username string) (*domain.User, error) {
	var m UserModel
	err := r.db.Where("username = ?", username).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		Role:         m.Role,
	}, nil
}
