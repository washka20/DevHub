package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
	if cfg.ProjectsDir != "~/project" {
		t.Errorf("expected projects_dir ~/project, got %s", cfg.ProjectsDir)
	}
	if cfg.DefaultProject != "cfa" {
		t.Errorf("expected default_project cfa, got %s", cfg.DefaultProject)
	}
}

func TestDefaultConfig_TerminalMaxSessions(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Terminal.MaxSessions != 10 {
		t.Errorf("expected MaxSessions=10, got %d", cfg.Terminal.MaxSessions)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := []byte("port: 8080\nprojects_dir: /tmp/myprojects\ndefault_project: myapp\n")

	// Override HOME to point to our temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := os.WriteFile(filepath.Join(dir, ".devhub.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.ProjectsDir != "/tmp/myprojects" {
		t.Errorf("expected projects_dir /tmp/myprojects, got %s", cfg.ProjectsDir)
	}
	if cfg.DefaultProject != "myapp" {
		t.Errorf("expected default_project myapp, got %s", cfg.DefaultProject)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected default port 9000, got %d", cfg.Port)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := os.WriteFile(filepath.Join(dir, ".devhub.yaml"), []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	result := ExpandHome("~/project")
	expected := filepath.Join(home, "project")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestExpandHomeNoTilde(t *testing.T) {
	result := ExpandHome("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("expected /absolute/path, got %s", result)
	}
}

func TestPortOverride(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	yamlContent := []byte("port: 3000\n")
	if err := os.WriteFile(filepath.Join(dir, ".devhub.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
}

func TestProjectsDirOverride(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	yamlContent := []byte("projects_dir: /opt/dev\n")
	if err := os.WriteFile(filepath.Join(dir, ".devhub.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ProjectsDir != "/opt/dev" {
		t.Errorf("expected /opt/dev, got %s", cfg.ProjectsDir)
	}
}
