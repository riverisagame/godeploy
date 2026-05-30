package db

import (

	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"

	"deploy/godeployer/domain"
)

var DB *gorm.DB

// InitGORM initializes GORM database instance for given driver and DSN
func InitGORM(driverName, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch driverName {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	case "sqlite":
		fallthrough
	default:
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// For SQLite, enforce single connection if possible (though sqlite driver might handle it)
	if driverName == "sqlite" {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}

	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	if err := createDefaultAdmin(db); err != nil {
		return nil, fmt.Errorf("failed to seed admin: %w", err)
	}

	if err := repairStalledTasks(db); err != nil {
		return nil, fmt.Errorf("failed to auto-repair stalled tasks: %w", err)
	}

	DB = db
	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.UserResponse{},
		&domain.DeployTask{},
	)
}

func createDefaultAdmin(db *gorm.DB) error {
	var count int64
	if err := db.Model(&domain.UserResponse{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		pwd := os.Getenv("ADMIN_PASSWORD")
		if pwd == "" {
			pwd = "admin123"
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		admin := &domain.UserResponse{
			Username:     "admin",
			PasswordHash: string(hash),
			Role:         "admin",
			CreatedAt:    time.Now(),
		}
		if err := db.Create(admin).Error; err != nil {
			return err
		}
	} else {
		// Ensure admin user has admin role
		db.Model(&domain.UserResponse{}).Where("username = ?", "admin").Update("role", "admin")
	}
	return nil
}

func repairStalledTasks(db *gorm.DB) error {
	return db.Model(&domain.DeployTask{}).
		Where("status IN ?", []string{"pending", "deploying"}).
		Update("status", "aborted").Error
}

// InitTestDB provides a unified helper for tests to get both sql.DB and TaskRepository
func InitTestDB(dsn string) (*sql.DB, domain.TaskRepository, error) {
	gormDB, err := InitGORM("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, err
	}
	taskRepo := NewTaskRepository(gormDB)
	return sqlDB, taskRepo, nil
}
