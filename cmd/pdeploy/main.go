package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverisagame/godeploy"
	"github.com/riverisagame/godeploy/internal/application"
	"github.com/riverisagame/godeploy/internal/config"
	"github.com/riverisagame/godeploy/internal/infrastructure/git"
	"github.com/riverisagame/godeploy/internal/infrastructure/persistence"
	"github.com/riverisagame/godeploy/internal/infrastructure/ssh"
	"github.com/riverisagame/godeploy/internal/interfaces/api"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// 1. Initialize Infrastructure
	// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 2.3 | @Date: 2026-07-21
	dsn := cfg.DBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1) // Avoid database is locked errors
	}

	err = db.AutoMigrate(
		&persistence.ProjectModel{},
		&persistence.EnvironmentModel{},
		&persistence.ServerModel{},
		&persistence.DeploymentModel{},
		&persistence.UserModel{},
	)
	if err != nil {
		log.Fatal("Failed to auto-migrate:", err)
	}

	projectRepo := persistence.NewSqliteProjectRepository(db)
	serverRepo := persistence.NewSqliteServerRepository(db)
	deployRepo := persistence.NewSqliteDeploymentRepository(db)

	// 2. Initialize Application Services
	projectSvc := application.NewProjectService(projectRepo)
	sshClient := ssh.NewClient()
	gitClient := git.NewClient(cfg.WorkspaceDir)

	userRepo := persistence.NewSqliteUserRepository(db)
	authSvc := application.NewAuthService(userRepo, cfg.JWTSecret)

	// Create default admin user if none exists
	if err := authSvc.InitAdminUser("admin", "admin123"); err != nil {
		log.Printf("Failed to init default admin user: %v", err)
	}

	serverSvc := application.NewServerService(serverRepo, projectRepo)

	deploySvc := application.NewDeployService(deployRepo, projectRepo, gitClient)
	deployEngine := application.NewDeployEngine(sshClient, gitClient, serverSvc, deploySvc)

	// 3. Initialize Interfaces
	router := api.NewRouter(projectSvc, serverSvc, deploySvc, deployEngine, authSvc, pdeploy.StaticFS, cfg.JWTSecret)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("pdeploy server starting on %s...\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
