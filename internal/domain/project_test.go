package domain

import "testing"

func TestNewProject(t *testing.T) {
	_, err := NewProject("", "git@github.com:demo/demo.git")
	if err == nil || err.Error() != "project name cannot be empty" {
		t.Errorf("Expected error 'project name cannot be empty', got %v", err)
	}

	_, err = NewProject("demo", "")
	if err == nil || err.Error() != "repo URL cannot be empty" {
		t.Errorf("Expected error 'repo URL cannot be empty', got %v", err)
	}

	p, err := NewProject("demo", "git@github.com:demo/demo.git")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if p.Name != "demo" {
		t.Errorf("Expected name 'demo', got '%s'", p.Name)
	}
	if p.KeepReleases != 5 {
		t.Errorf("Expected keep_releases default 5, got %d", p.KeepReleases)
	}
}

func TestAddEnvironment(t *testing.T) {
	p, _ := NewProject("demo", "git@github.com:demo/demo.git")

	err := p.AddEnvironment("dev", "develop", "symlink")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(p.Environments) != 1 {
		t.Fatalf("Expected 1 environment, got %d", len(p.Environments))
	}

	err = p.AddEnvironment("", "develop", "symlink")
	if err == nil || err.Error() != "environment name cannot be empty" {
		t.Errorf("Expected error 'environment name cannot be empty', got %v", err)
	}
}

func TestEnvironmentEnvVars(t *testing.T) {
	// @Ref: docs/sps/plans/20260720_env_vars_ir.md | @Date: 2026-07-20
	p, _ := NewProject("demo", "git@github.com:demo/demo.git")
	_ = p.AddEnvironment("prod", "main", "symlink")
	env := p.Environments[0]

	if len(env.EnvVars) != 0 {
		t.Fatalf("Expected 0 env vars initially")
	}

	env.AddEnvVar("DB_HOST", "localhost", false)
	env.AddEnvVar("DB_PASS", "secret", true)

	if len(env.EnvVars) != 2 {
		t.Fatalf("Expected 2 env vars")
	}

	if env.EnvVars[0].Key != "DB_HOST" || env.EnvVars[1].IsSecret != true {
		t.Errorf("Env var fields mismatch")
	}
}
