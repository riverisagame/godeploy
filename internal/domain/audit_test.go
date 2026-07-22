package domain_test

import (
	"testing"
	"time"

	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogCreation(t *testing.T) {
	now := time.Now()
	log := &domain.AuditLog{
		ID:        1,
		UserID:    100,
		Username:  "admin",
		Method:    "POST",
		Path:      "/api/servers",
		Details:   "{}",
		CreatedAt: now,
	}

	assert.Equal(t, int64(1), log.ID)
	assert.Equal(t, int64(100), log.UserID)
	assert.Equal(t, "admin", log.Username)
	assert.Equal(t, "POST", log.Method)
	assert.Equal(t, "/api/servers", log.Path)
	assert.Equal(t, "{}", log.Details)
	assert.Equal(t, now, log.CreatedAt)
}
