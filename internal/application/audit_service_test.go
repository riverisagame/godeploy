package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAuditRepository) List(ctx context.Context, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*domain.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	args := m.Called(ctx, cutoff)
	return args.Error(0)
}

func TestAuditService(t *testing.T) {
	mockRepo := new(MockAuditRepository)
	svc := application.NewAuditService(mockRepo)
	ctx := context.Background()

	t.Run("RecordAction", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.AuditLog")).Return(nil).Once()
		mockRepo.On("DeleteOlderThan", mock.Anything, mock.Anything).Return(nil).Maybe()
		
		err := svc.RecordAction(ctx, 1, "user1", "DELETE", "/api/item/1", "{}")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetLogs", func(t *testing.T) {
		logs := []*domain.AuditLog{{ID: 1}}
		mockRepo.On("List", ctx, 1, 10).Return(logs, int64(1), nil).Once()

		resLogs, total, err := svc.GetLogs(ctx, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, resLogs, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("PruneLogs", func(t *testing.T) {
		mockRepo.On("DeleteOlderThan", mock.Anything, mock.Anything).Return(nil).Maybe()
		
		err := svc.PruneLogs(ctx)
		assert.NoError(t, err)
	})
}
