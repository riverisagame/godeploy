package db

import (
	"testing"
	"time"

	"deploy/godeployer/domain"
)

func TestTaskRepository(t *testing.T) {
	db, err := InitGORM("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	repo := NewTaskRepository(db)

	task := &domain.DeployTask{
		ProjectID:      "proj_1",
		EnvID:          "env_1",
		CommitID:       "abc1234",
		Status:         "pending",
		ReleaseName:    "rel_1",
		UserID:         1,
		Username:       "admin",
		ConfigSnapshot: "{}",
		CreatedAt:      time.Now(),
	}

	if err := repo.InsertTask(task); err != nil {
		t.Fatalf("InsertTask failed: %v", err)
	}

	if task.ID == 0 {
		t.Error("expected ID to be set after insert")
	}

	fetched, err := repo.GetTaskByID(task.ID)
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if fetched.CommitID != "abc1234" {
		t.Errorf("expected commit abc1234, got %s", fetched.CommitID)
	}

	tasks, err := repo.GetTasksByEnv("proj_1", "env_1", 10)
	if err != nil {
		t.Fatalf("GetTasksByEnv failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}
