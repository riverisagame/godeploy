package persistence

import (
	"context"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
	"gorm.io/gorm"
)

type AuditLogModel struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64 `gorm:"index"`
	Username  string
	Method    string
	Path      string
	Details   string
	CreatedAt time.Time `gorm:"index"`
}

type SqliteAuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *SqliteAuditRepository {
	return &SqliteAuditRepository{db: db}
}

func (r *SqliteAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	model := &AuditLogModel{
		UserID:    log.UserID,
		Username:  log.Username,
		Method:    log.Method,
		Path:      log.Path,
		Details:   log.Details,
		CreatedAt: log.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}

	log.ID = model.ID
	return nil
}

func (r *SqliteAuditRepository) List(ctx context.Context, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	var models []AuditLogModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&AuditLogModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	var logs []*domain.AuditLog
	for _, m := range models {
		logs = append(logs, &domain.AuditLog{
			ID:        m.ID,
			UserID:    m.UserID,
			Username:  m.Username,
			Method:    m.Method,
			Path:      m.Path,
			Details:   m.Details,
			CreatedAt: m.CreatedAt,
		})
	}

	return logs, total, nil
}

func (r *SqliteAuditRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&AuditLogModel{}).Error
}
