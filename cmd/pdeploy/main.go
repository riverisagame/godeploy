package main

import (
	"log"
	"net/http"
	"pdeploy"
	"pdeploy/internal/application"
	"pdeploy/internal/config"
	"pdeploy/internal/infrastructure/git"
	"pdeploy/internal/infrastructure/persistence"
	"pdeploy/internal/infrastructure/ssh"
	"pdeploy/internal/interfaces/api"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// 1. Initialize Infrastructure
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to open DB:", err)
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
	serverSvc := application.NewServerService(serverRepo)

	deploySvc := application.NewDeployService(deployRepo, projectRepo, gitClient)
	deployEngine := application.NewDeployEngine(sshClient, gitClient, serverSvc, deploySvc)

	// 3. Initialize Interfaces
	router := api.NewRouter(projectSvc, serverSvc, deploySvc, deployEngine, authSvc, pdeploy.StaticFS)

	addr := ":" + cfg.Port
	log.Printf("pdeploy server starting on %s...\n", addr)
	err = http.ListenAndServe(addr, router)
	if err != nil {
		log.Fatal("Server failed:", err)
	}
}
