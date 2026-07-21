package domain

import "testing"

func TestNewDeployment(t *testing.T) {
	_, err := NewDeployment(0, 1, "hash123")
	if err == nil || err.Error() != "environment ID must be > 0" {
		t.Errorf("Expected environment ID error, got %v", err)
	}
	
	_, err = NewDeployment(1, 0, "hash123")
	if err != nil {
		t.Fatalf("Expected no error for user ID 0 (webhook), got %v", err)
	}
	
	d, err := NewDeployment(1, 1, "hash123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if d.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", d.Status)
	}
	if d.Phase != "init" {
		t.Errorf("Expected phase 'init', got '%s'", d.Phase)
	}
	
	d.SetPhase("pre_deploy")
	if d.Phase != "pre_deploy" {
		t.Errorf("Expected phase 'pre_deploy', got '%s'", d.Phase)
	}
	
	d.MarkSuccess("deployed ok", "release_v1")
	if d.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", d.Status)
	}
	if d.Log != "deployed ok" {
		t.Errorf("Expected log 'deployed ok', got '%s'", d.Log)
	}
	if d.ReleaseName != "release_v1" {
		t.Errorf("Expected release name 'release_v1', got '%s'", d.ReleaseName)
	}
}
