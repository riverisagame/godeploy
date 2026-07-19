package persistence

import (
	"pdeploy/internal/domain"
	"gorm.io/gorm"
)

type ServerModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string
	IP   string
	Port int
}

type SqliteServerRepository struct {
	db *gorm.DB
}

func NewSqliteServerRepository(db *gorm.DB) *SqliteServerRepository {
	return &SqliteServerRepository{db: db}
}

func (r *SqliteServerRepository) Save(s *domain.Server) error {
	m := &ServerModel{
		ID:   s.ID,
		Name: s.Name,
		IP:   s.IP,
		Port: s.Port,
	}

	if err := r.db.Save(m).Error; err != nil {
		return err
	}
	s.ID = m.ID
	return nil
}

func (r *SqliteServerRepository) FindAll() ([]*domain.Server, error) {
	var models []ServerModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}

	var res []*domain.Server
	for _, m := range models {
		res = append(res, &domain.Server{
			ID:   m.ID,
			Name: m.Name,
			IP:   m.IP,
			Port: m.Port,
		})
	}
	return res, nil
}

func (r *SqliteServerRepository) FindByID(id uint) (*domain.Server, error) {
	var m ServerModel
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &domain.Server{
		ID:   m.ID,
		Name: m.Name,
		IP:   m.IP,
		Port: m.Port,
	}, nil
}
