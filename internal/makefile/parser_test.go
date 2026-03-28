package makefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCategorize_Docker(t *testing.T) {
	for _, name := range []string{"docker", "up", "down", "start", "stop", "docker-build"} {
		if cat := categorize(name); cat != "Docker" {
			t.Errorf("categorize(%q) = %q, want Docker", name, cat)
		}
	}
}

func TestCategorize_NPM(t *testing.T) {
	for _, name := range []string{"npm-install", "npm_build"} {
		if cat := categorize(name); cat != "NPM" {
			t.Errorf("categorize(%q) = %q, want NPM", name, cat)
		}
	}
}

func TestCategorize_Composer(t *testing.T) {
	for _, name := range []string{"composer-install", "composer_update"} {
		if cat := categorize(name); cat != "Composer" {
			t.Errorf("categorize(%q) = %q, want Composer", name, cat)
		}
	}
}

func TestCategorize_PHP(t *testing.T) {
	for _, name := range []string{"migrate", "cache", "winter", "migrate-fresh", "cache-clear"} {
		if cat := categorize(name); cat != "PHP" {
			t.Errorf("categorize(%q) = %q, want PHP", name, cat)
		}
	}
}

func TestCategorize_Git(t *testing.T) {
	for _, name := range []string{"pull", "submodules", "pull-all"} {
		if cat := categorize(name); cat != "Git" {
			t.Errorf("categorize(%q) = %q, want Git", name, cat)
		}
	}
}

func TestCategorize_Init(t *testing.T) {
	for _, name := range []string{"init", "fresh", "env", "init-dev"} {
		if cat := categorize(name); cat != "Init" {
			t.Errorf("categorize(%q) = %q, want Init", name, cat)
		}
	}
}

func TestCategorize_Other(t *testing.T) {
	for _, name := range []string{"test", "lint", "build", "deploy"} {
		if cat := categorize(name); cat != "Other" {
			t.Errorf("categorize(%q) = %q, want Other", name, cat)
		}
	}
}

func TestParse(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")

	content := `## Build the project
build:
	go build ./...

## Run tests
test:
	go test ./...

up:
	docker compose up -d

npm-install:
	npm install
`
	if err := os.WriteFile(makefilePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	commands, err := Parse(makefilePath)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(commands) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(commands))
	}

	expected := []struct {
		name     string
		desc     string
		category string
	}{
		{"build", "Build the project", "Other"},
		{"test", "Run tests", "Other"},
		{"up", "", "Docker"},
		{"npm-install", "", "NPM"},
	}

	for i, exp := range expected {
		if commands[i].Name != exp.name {
			t.Errorf("command[%d].Name = %q, want %q", i, commands[i].Name, exp.name)
		}
		if commands[i].Description != exp.desc {
			t.Errorf("command[%d].Description = %q, want %q", i, commands[i].Description, exp.desc)
		}
		if commands[i].Category != exp.category {
			t.Errorf("command[%d].Category = %q, want %q", i, commands[i].Category, exp.category)
		}
	}
}

func TestParse_WithDescriptions(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")

	content := `## Start all services
up:
	docker compose up -d

## Stop all services
down:
	docker compose down
`
	if err := os.WriteFile(makefilePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	commands, err := Parse(makefilePath)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	if commands[0].Description != "Start all services" {
		t.Errorf("expected description 'Start all services', got %q", commands[0].Description)
	}
	if commands[1].Description != "Stop all services" {
		t.Errorf("expected description 'Stop all services', got %q", commands[1].Description)
	}
}

func TestParse_InternalTargets(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")

	content := `.PHONY: all
_internal:
	echo "hidden"

build:
	go build
`
	if err := os.WriteFile(makefilePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	commands, err := Parse(makefilePath)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("expected 1 command (build only), got %d", len(commands))
	}
	if commands[0].Name != "build" {
		t.Errorf("expected build, got %q", commands[0].Name)
	}
}

func TestParse_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")

	if err := os.WriteFile(makefilePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	commands, err := Parse(makefilePath)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(commands))
	}
}

func TestParse_NonExistentFile(t *testing.T) {
	_, err := Parse("/nonexistent/Makefile")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
