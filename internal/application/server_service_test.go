package application_test

import (
	"pdeploy/internal/application"
	"pdeploy/internal/infrastructure/persistence"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestServerService_CreateServer(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&persistence.ServerModel{}, &persistence.ProjectModel{}, &persistence.EnvironmentModel{})
	repo := persistence.NewSqliteServerRepository(db)
	projectRepo := persistence.NewSqliteProjectRepository(db)
	svc := application.NewServerService(repo, projectRepo)

	server, err := svc.CreateServer("Test Server", "127.0.0.1", 22, "deploy", "~/.ssh/id_rsa")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if server.ID == 0 {
		t.Errorf("expected ID > 0, got %d", server.ID)
	}
	if server.User != "deploy" {
		t.Errorf("expected User 'deploy', got '%s'", server.User)
	}

	servers, err := svc.ListServers()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}
}

func TestServerService_GetServer(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&persistence.ServerModel{}, &persistence.ProjectModel{}, &persistence.EnvironmentModel{})
	repo := persistence.NewSqliteServerRepository(db)
	projectRepo := persistence.NewSqliteProjectRepository(db)
	svc := application.NewServerService(repo, projectRepo)

	created, _ := svc.CreateServer("Test", "1.1.1.1", 22, "", "")
	
	srv, err := svc.GetServerByID(created.ID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if srv.Name != "Test" {
		t.Errorf("expected Test, got %s", srv.Name)
	}
}

func TestServerService_DeleteServer(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&persistence.ServerModel{}, &persistence.ProjectModel{}, &persistence.EnvironmentModel{})
	repo := persistence.NewSqliteServerRepository(db)
	projectRepo := persistence.NewSqliteProjectRepository(db)
	svc := application.NewServerService(repo, projectRepo)

	created, _ := svc.CreateServer("Test Delete", "1.1.1.2", 22, "", "")
	
	// Create a project that references this server
	projectSvc := application.NewProjectService(projectRepo)
	prj, err := projectSvc.CreateProject("Test Project", "git@github.com:test/test.git")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	_, err = projectSvc.AddEnvironment(prj.ID, "prod", "main", "static")
	if err != nil {
		t.Fatalf("failed to add env: %v", err)
	}
	// Fetch it to update
	prjs, _ := projectSvc.GetProjects()
	prjs[0].Environments[0].ServerIDs = []uint{created.ID}
	projectRepo.Save(prjs[0])

	var em persistence.EnvironmentModel
	db.First(&em)
	t.Logf("Environment from DB: %+v", em)

	prjsBefore, _ := projectSvc.GetProjects()
	if len(prjsBefore[0].Environments[0].ServerIDs) != 1 {
		t.Fatalf("Setup failed: expected 1 server in environment, got %d, DB JSON: %s", len(prjsBefore[0].Environments[0].ServerIDs), em.ServerIDs)
	}
	
	err = svc.DeleteServer(created.ID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	servers, _ := svc.ListServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after delete, got %d", len(servers))
	}
	
	// Verify it was removed from the environment
	prjsAfter, _ := projectSvc.GetProjects()
	if len(prjsAfter[0].Environments[0].ServerIDs) != 0 {
		t.Errorf("expected 0 servers in environment after delete, got %d", len(prjsAfter[0].Environments[0].ServerIDs))
	} else {
		t.Logf("Unexpected success: ServerIDs is %v", prjsAfter[0].Environments[0].ServerIDs)
	}
}

func TestServerService_UpdateServer(t *testing.T) {
	// [RED] Edge Cases Test for UpdateServer
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&persistence.ServerModel{}, &persistence.ProjectModel{}, &persistence.EnvironmentModel{})
	repo := persistence.NewSqliteServerRepository(db)
	projectRepo := persistence.NewSqliteProjectRepository(db)
	svc := application.NewServerService(repo, projectRepo)

	created, _ := svc.CreateServer("Old Server", "1.1.1.1", 22, "root", "/key")
	
	updated, err := svc.UpdateServer(created.ID, "New Server", "2.2.2.2", 2222, "admin", "/newkey")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	
	if updated.Name != "New Server" || updated.IP != "2.2.2.2" || updated.Port != 2222 || updated.User != "admin" || updated.KeyPath != "/newkey" {
		t.Errorf("expected server to be fully updated, got: %v", updated)
	}
	
	// Edge Case: Invalid server
	_, err = svc.UpdateServer(999, "New Server", "2.2.2.2", 2222, "admin", "/newkey")
	if err == nil {
		t.Errorf("expected error for non-existent server")
	}
}
