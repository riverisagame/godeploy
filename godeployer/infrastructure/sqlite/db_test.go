package sqlite_test

import (
	"database/sql"
	"testing"
	"time"

	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/sqlite"

	_ "github.com/glebarez/go-sqlite"
)

func TestUserRepository_CreateAndGetUser(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	// 初始化表结构
	_, err = db.Exec(`
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		bound_git_authors TEXT DEFAULT '',
		restrict_git_authors BOOLEAN DEFAULT 0,
		permitted_projects TEXT DEFAULT ''
	);
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	repo := sqlite.NewUserRepository(db)

	user := &domain.UserResponse{
		Username:           "testuser",
		Role:               "Admin",
		BoundGitAuthors:    "test",
		RestrictGitAuthors: true,
		PermittedProjects:  "all",
		CreatedAt:          time.Now(),
	}

	// RED Phase: 尚未实现 CreateUser
	err = repo.CreateUser(user, "dummy_hash") // Note: we need hash to create
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	res, err := repo.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if res.Username != "testuser" {
		t.Fatalf("expected testuser, got %s", res.Username)
	}
}
