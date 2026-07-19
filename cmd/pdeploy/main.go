package main

import (
	"log"
	"net/http"
	"pdeploy"
	"pdeploy/internal/application"
	"pdeploy/internal/infrastructure/git"
	"pdeploy/internal/infrastructure/persistence"
	"pdeploy/internal/infrastructure/ssh"
	"pdeploy/internal/interfaces/api"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Initialize Infrastructure
	db, err := gorm.Open(sqlite.Open("pdeploy.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	err = db.AutoMigrate(
		&persistence.ProjectModel{}, 
		&persistence.EnvironmentModel{},
		&persistence.ServerModel{},
		&persistence.DeploymentModel{},
	)
	if err != nil {
		log.Fatal("Failed to auto-migrate:", err)
	}

	projectRepo := persistence.NewSqliteProjectRepository(db)
	serverRepo := persistence.NewSqliteServerRepository(db)
	deployRepo := persistence.NewSqliteDeploymentRepository(db)

	// 2. Initialize Application Services
	projectSvc := application.NewProjectService(projectRepo)
	deploySvc := application.NewDeployService(deployRepo)

	sshClient := ssh.NewClient()
	gitClient := git.NewClient("./workspace")
	deployEngine := application.NewDeployEngine(sshClient, gitClient, serverRepo, deploySvc)

	// 3. Initialize Interfaces
	router := api.NewRouter(projectSvc, serverRepo, deploySvc, deployEngine, pdeploy.StaticFS)

	log.Println("pdeploy server starting on :8080...")
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal("Server failed:", err)
	}
}
