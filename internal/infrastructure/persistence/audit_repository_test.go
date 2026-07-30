package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/riverisagame/godeploy/internal/domain"
	"github.com/riverisagame/godeploy/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAuditRepository(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&persistence.AuditLogModel{})
	assert.NoError(t, err)

	repo := persistence.NewAuditRepository(db)
	ctx := context.Background()

	// Test Create
	log := &domain.AuditLog{
		UserID:    1,
		Username:  "testuser",
		Method:    "PUT",
		Path:      "/api/projects/1",
		Details:   `{"name": "newname"}`,
		CreatedAt: time.Now(),
	}

	err = repo.Create(ctx, log)
	assert.NoError(t, err)
	assert.True(t, log.ID > 0)

	// Test List
	logs, total, err := repo.List(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "testuser", logs[0].Username)

	// Test DeleteOlderThan
	oldLog := &domain.AuditLog{
		UserID:    1,
		Username:  "olduser",
		Method:    "GET",
		Path:      "/api/old",
		CreatedAt: time.Now().AddDate(0, 0, -100), // 100 days ago
	}
	err = repo.Create(ctx, oldLog)
	assert.NoError(t, err)

	err = repo.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -90))
	assert.NoError(t, err)

	logs, total, err = repo.List(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total) // Should only have 1 log remaining
	// @Ref: docs/sps/plans/20260730_bugfix_plan.md | @Date: 2026-07-30
	assert.Len(t, logs, 1)
}
