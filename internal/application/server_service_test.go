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
	db.AutoMigrate(&persistence.ServerModel{})
	repo := persistence.NewSqliteServerRepository(db)
	svc := application.NewServerService(repo)

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
	db.AutoMigrate(&persistence.ServerModel{})
	repo := persistence.NewSqliteServerRepository(db)
	svc := application.NewServerService(repo)

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
	db.AutoMigrate(&persistence.ServerModel{})
	repo := persistence.NewSqliteServerRepository(db)
	svc := application.NewServerService(repo)

	created, _ := svc.CreateServer("Test Delete", "1.1.1.2", 22, "", "")
	
	err := svc.DeleteServer(created.ID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	servers, _ := svc.ListServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after delete, got %d", len(servers))
	}
}
