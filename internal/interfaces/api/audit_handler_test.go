package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/riverisagame/godeploy/internal/interfaces/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) RecordAction(ctx context.Context, userID int64, username, method, path, details string) error {
	args := m.Called(ctx, userID, username, method, path, details)
	return args.Error(0)
}

func (m *MockAuditService) GetLogs(ctx context.Context, page, pageSize int) ([]*domain.AuditLog, int64, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]*domain.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuditService) PruneLogs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestGetAuditLogs(t *testing.T) {
	mockSvc := new(MockAuditService)
	handler := api.NewAuditHandler(mockSvc)

	req, _ := http.NewRequest("GET", "/api/audit-logs?page=1&pageSize=10", nil)
	rr := httptest.NewRecorder()

	logs := []*domain.AuditLog{
		{ID: 1, Username: "admin", Method: "POST"},
	}
	mockSvc.On("GetLogs", mock.Anything, 1, 10).Return(logs, int64(1), nil)

	handler.GetAuditLogs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var res map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), res["total"])

	data := res["data"].([]interface{})
	assert.Len(t, data, 1)

	mockSvc.AssertExpectations(t)
}
