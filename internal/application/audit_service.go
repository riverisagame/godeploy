package application

import (
	"context"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
)

type AuditService interface {
	RecordAction(ctx context.Context, userID int64, username, method, path, details string) error
	GetLogs(ctx context.Context, page, pageSize int) ([]*domain.AuditLog, int64, error)
	PruneLogs(ctx context.Context) error
}

type auditService struct {
	repo domain.AuditRepository
}

func NewAuditService(repo domain.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) RecordAction(ctx context.Context, userID int64, username, method, path, details string) error {
	log := &domain.AuditLog{
		UserID:    userID,
		Username:  username,
		Method:    method,
		Path:      path,
		Details:   details,
		CreatedAt: time.Now(),
	}

	err := s.repo.Create(ctx, log)

	// 每次记录有极小几率触发清理 (也可以用定时任务，但为了简单内聚此处用随机触发)
	if time.Now().UnixNano()%100 == 0 {
		go func() {
			_ = s.repo.DeleteOlderThan(context.Background(), time.Now().AddDate(0, 0, -90))
		}()
	}

	return err
}

func (s *auditService) GetLogs(ctx context.Context, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}

func (s *auditService) PruneLogs(ctx context.Context) error {
	return s.repo.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -90))
}
