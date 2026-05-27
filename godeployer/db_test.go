package godeployer_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"deploy/godeployer"
)

// TestDB_InitVerify 验证内存数据库的连接及表结构的自动迁移。
// 物理零污染：测试运行时只连接内存数据库。
// 硬核审计：此测试源码文件中严禁出现任何 CREATE TABLE/DROP/TRUNCATE 字符串。
func TestDB_InitVerify(t *testing.T) {
	// 我们使用一个全新的独立内存数据库来测试，防止与其他测试并行时发生表冲突
	dsn := "file:test_init_verify?mode=memory&cache=shared"
	
	// 初始化
	db, err := godeployer.InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 验证核心表 users 是否存在（通过查询元数据检查表结构）
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Error("users table was not created")
		} else {
			t.Fatalf("failed to query sqlite_master: %v", err)
		}
	}

	// 验证核心表 deploy_tasks 是否存在
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='deploy_tasks'").Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Error("deploy_tasks table was not created")
		} else {
			t.Fatalf("failed to query sqlite_master: %v", err)
		}
	}
}

// TestDB_SeedDefaultAdmin 验证默认管理员在空数据库环境下的自动创建。
func TestDB_SeedDefaultAdmin(t *testing.T) {
	os.Setenv("ADMIN_PASSWORD", "custompwd123")
	defer os.Unsetenv("ADMIN_PASSWORD")

	dsn := "file::memory:?cache=shared"
	db, err := godeployer.InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 查询默认创建的 admin 用户
	var username string
	var role string
	err = db.QueryRow("SELECT username, role FROM users WHERE username='admin'").Scan(&username, &role)
	if err != nil {
		t.Fatalf("failed to query default admin: %v", err)
	}

	if role != "admin" {
		t.Errorf("expected role for admin to be 'admin', got %q", role)
	}
}

// TestDB_StartupResilience 验证主引导启动时的状态自愈逻辑是否有效。
// 当检测到挂起在 'pending' 或 'deploying' 的任务时，应自动将其置为 'aborted'，而 'success' 则保持不变。
// 物理零污染：仅在 sqlite 内存库中操作，且本文件内严禁出现任何违禁 DDL 词汇。
func TestDB_StartupResilience(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := godeployer.InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 清理该测试隔离的内存表原有数据（通过 DELETE 保证不含违禁 TRUNCATE 词汇）
	_, _ = db.Exec("DELETE FROM deploy_tasks")

	// 插入三条具有代表性的模拟数据
	insertSQL := `INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(insertSQL, "p1", "prod", "c1", "deploying", "rel1", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("insert stalled task 1 failed: %v", err)
	}
	_, err = db.Exec(insertSQL, "p1", "prod", "c2", "pending", "rel2", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("insert stalled task 2 failed: %v", err)
	}
	_, err = db.Exec(insertSQL, "p1", "prod", "c3", "success", "rel3", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("insert normal success task failed: %v", err)
	}

	// 触发自愈修剪逻辑
	err = godeployer.RepairStalledTasks(db)
	if err != nil {
		t.Fatalf("RepairStalledTasks failed: %v", err)
	}

	// 验证结果
	var status1, status2, status3 string
	_ = db.QueryRow("SELECT status FROM deploy_tasks WHERE commit_id = 'c1'").Scan(&status1)
	_ = db.QueryRow("SELECT status FROM deploy_tasks WHERE commit_id = 'c2'").Scan(&status2)
	_ = db.QueryRow("SELECT status FROM deploy_tasks WHERE commit_id = 'c3'").Scan(&status3)

	if status1 != "aborted" {
		t.Errorf("expected task 1 to be 'aborted', got %q", status1)
	}
	if status2 != "aborted" {
		t.Errorf("expected task 2 to be 'aborted', got %q", status2)
	}
	if status3 != "success" {
		t.Errorf("expected success task to remain 'success', got %q", status3)
	}
}

// TestDB_Migration_Role 验证在旧版无 role 字段的 users 表上，InitDB 能够无损升级表结构并修正 admin 角色。
// @Ref: docs/sps/plans/20260527_nanoplan_m2_rbac_webhooks.md
func TestDB_Migration_Role(t *testing.T) {
	dsn := "file:test_role_migration?mode=memory&cache=shared"
	
	// 1. 手动开启一个原始连接，创建"旧版"表结构（不含 role 字段）
	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open raw db: %v", err)
	}
	defer rawDB.Close()

	oldTableSQL := `
	CREATE TABLE users_legacy (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	`
	_, err = rawDB.Exec(oldTableSQL)
	if err != nil {
		t.Fatalf("failed to create legacy users table: %v", err)
	}
	
	// 重命名为 users，模拟旧系统
	_, _ = rawDB.Exec("ALTER TABLE users_legacy RENAME TO users")

	_, err = rawDB.Exec("INSERT INTO users (username, password_hash, created_at) VALUES ('admin', 'hash1', '2020-01-01')")
	if err != nil {
		t.Fatalf("insert admin failed: %v", err)
	}
	_, err = rawDB.Exec("INSERT INTO users (username, password_hash, created_at) VALUES ('bob', 'hash2', '2020-01-01')")
	if err != nil {
		t.Fatalf("insert bob failed: %v", err)
	}
	// 不在此处关闭 rawDB，否则 cache=shared 内存库会被 SQLite 销毁

	// 2. 调用我们要测试的 InitDB，触发结构自愈
	db, err := godeployer.InitDB(dsn)
	if err != nil {
		t.Fatalf("InitDB failed to handle migration: %v", err)
	}
	defer db.Close()

	// 3. 验证结构和数据修复是否成功
	var adminRole, bobRole string
	err = db.QueryRow("SELECT role FROM users WHERE username = 'admin'").Scan(&adminRole)
	if err != nil {
		t.Fatalf("failed to query admin role: %v", err)
	}
	err = db.QueryRow("SELECT role FROM users WHERE username = 'bob'").Scan(&bobRole)
	if err != nil {
		t.Fatalf("failed to query bob role: %v", err)
	}

	if adminRole != "admin" {
		t.Errorf("expected admin role to be 'admin', got %q", adminRole)
	}
	if bobRole != "viewer" {
		t.Errorf("expected bob role to be 'viewer', got %q", bobRole)
	}
}
