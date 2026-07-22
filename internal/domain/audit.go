package domain

import (
	"context"
	"time"
)

type AuditLog struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"index"`
	Username  string    `json:"username"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, page, pageSize int) ([]*AuditLog, int64, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) error
}
