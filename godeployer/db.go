package godeployer

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// InitDB 连接 SQLite 并自动创建必要的表结构。
func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 强制单连接模式，规避并发写时的 database is locked 错误
	db.SetMaxOpenConns(1)

	// 创建用户表
	createUserTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`
	if _, err := db.Exec(createUserTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	// 尝试向可能存在的旧版 users 表中追加列（若已存在，则忽略错误）
	// @Ref: docs/sps/plans/20260527_nanoplan_m2_rbac_webhooks.md
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'viewer'")
	
	// @Ref: docs/sps/plans/20260527_nanoplan_git_binding.md
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN bound_git_authors TEXT DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN restrict_git_authors BOOLEAN DEFAULT 0")

	// @Ref: docs/sps/plans/20260528_project_permissions_plan.md
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN permitted_projects TEXT DEFAULT '*'")


	// 无损数据修复：确保现存的 admin 用户具备管理员权限
	_, err = db.Exec("UPDATE users SET role = 'admin' WHERE username = 'admin'")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate admin role: %w", err)
	}

	// 创建部署任务表
	createTaskTable := `
	CREATE TABLE IF NOT EXISTS deploy_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id TEXT NOT NULL,
		env_id TEXT NOT NULL,
		commit_id TEXT NOT NULL,
		status TEXT NOT NULL,
		release_name TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		config_snapshot TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`
	if _, err := db.Exec(createTaskTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create deploy_tasks table: %w", err)
	}

	DB = db

	// @Ref: docs/sps/plans/20260529_deploy_enhancements_plan.md | @Date: 2026-05-29
	// 平滑升级：检查 deploy_tasks 字段是否存在，不存在则动态添加
	var hasDescription, hasExtraExclude bool
	rows, err := db.Query("PRAGMA table_info(deploy_tasks)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, dType string
			var notnull, pk int
			var dfltValNull sql.NullString
			if err := rows.Scan(&cid, &name, &dType, &notnull, &dfltValNull, &pk); err == nil {
				if name == "description" {
					hasDescription = true
				}
				if name == "extra_exclude" {
					hasExtraExclude = true
				}
			}
		}
		rows.Close()
	}

	if !hasDescription {
		if _, err := db.Exec("ALTER TABLE deploy_tasks ADD COLUMN description TEXT DEFAULT ''"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to add description column: %w", err)
		}
	}
	if !hasExtraExclude {
		if _, err := db.Exec("ALTER TABLE deploy_tasks ADD COLUMN extra_exclude TEXT DEFAULT ''"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to add extra_exclude column: %w", err)
		}
	}

	// @Ref: docs/sps/plans/20260530_goal_perfect_diff_plan.md | @Date: 2026-05-30
	var hasTargetType bool
	rows2, err := db.Query("PRAGMA table_info(deploy_tasks)")
	if err == nil {
		for rows2.Next() {
			var cid int
			var name, dType string
			var notnull, pk int
			var dfltValNull sql.NullString
			if err := rows2.Scan(&cid, &name, &dType, &notnull, &dfltValNull, &pk); err == nil {
				if name == "target_type" {
					hasTargetType = true
				}
			}
		}
		rows2.Close()
	}
	if !hasTargetType {
		if _, err := db.Exec("ALTER TABLE deploy_tasks ADD COLUMN target_type TEXT DEFAULT ''"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to add target_type column: %w", err)
		}
	}

	// 自动创建默认管理员
	if err := createDefaultAdmin(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to seed admin: %w", err)
	}

	// 启动自愈机制，清退上次异常停机残留的历史锁
	// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
	if err := RepairStalledTasks(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to auto-repair stalled tasks: %w", err)
	}

	return db, nil
}

func createDefaultAdmin(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	// 只有在表为空时才插入默认管理员
	if count == 0 {
		pwd := os.Getenv("ADMIN_PASSWORD")
		if pwd == "" {
			pwd = "admin123"
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		insertSQL := `INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)`
		_, err = db.Exec(insertSQL, "admin", string(hash), "admin", time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

// RepairStalledTasks 修复挂起的任务。每次系统冷启动时调用，将 pending 和 deploying 的任务自动改为 aborted，以安全清锁。
// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
func RepairStalledTasks(db *sql.DB) error {
	query := `UPDATE deploy_tasks SET status = 'aborted' WHERE status IN ('pending', 'deploying')`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to repair stalled tasks: %w", err)
	}
	return nil
}
