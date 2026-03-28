package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_GitMakefileDocker(t *testing.T) {
	dir := t.TempDir()

	// Create a project with git, makefile, and docker-compose
	proj := filepath.Join(dir, "myapp")
	os.MkdirAll(filepath.Join(proj, ".git"), 0755)
	os.WriteFile(filepath.Join(proj, "Makefile"), []byte("build:\n\tgo build"), 0644)
	os.WriteFile(filepath.Join(proj, "docker-compose.yml"), []byte("version: '3'"), 0644)

	projects, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	p := projects[0]
	if p.Name != "myapp" {
		t.Errorf("expected name myapp, got %s", p.Name)
	}
	if !p.IsGit {
		t.Error("expected IsGit true")
	}
	if !p.HasMakefile {
		t.Error("expected HasMakefile true")
	}
	if !p.HasDocker {
		t.Error("expected HasDocker true")
	}
}

func TestScan_DockerVariants(t *testing.T) {
	dir := t.TempDir()

	// docker-compose.yaml (yaml extension)
	proj1 := filepath.Join(dir, "proj-yaml")
	os.MkdirAll(proj1, 0755)
	os.WriteFile(filepath.Join(proj1, "docker-compose.yaml"), []byte("version: '3'"), 0644)
	os.MkdirAll(filepath.Join(proj1, ".git"), 0755) // need at least one marker

	// docker-compose.dev.yml
	proj2 := filepath.Join(dir, "proj-dev")
	os.MkdirAll(proj2, 0755)
	os.WriteFile(filepath.Join(proj2, "docker-compose.dev.yml"), []byte("version: '3'"), 0644)
	os.MkdirAll(filepath.Join(proj2, ".git"), 0755)

	projects, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	for _, p := range projects {
		if !p.HasDocker {
			t.Errorf("project %s expected HasDocker true", p.Name)
		}
	}
}

func TestScan_NestedProjects(t *testing.T) {
	dir := t.TempDir()

	// Parent dir has no markers, but subdirs do
	parent := filepath.Join(dir, "company")
	os.MkdirAll(parent, 0755)

	sub1 := filepath.Join(parent, "frontend")
	os.MkdirAll(filepath.Join(sub1, ".git"), 0755)

	sub2 := filepath.Join(parent, "backend")
	os.MkdirAll(filepath.Join(sub2, ".git"), 0755)
	os.WriteFile(filepath.Join(sub2, "Makefile"), []byte("build:\n\tgo build"), 0644)

	projects, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 nested projects, got %d", len(projects))
	}
}

func TestScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	projects, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestScan_SkipFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file (not a directory) -- should be skipped
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0644)

	projects, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("expected 0 projects (files should be skipped), got %d", len(projects))
	}
}

func TestFindComposeFile_Exact(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)

	result := FindComposeFile(dir)
	expected := filepath.Join(dir, "docker-compose.yml")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindComposeFile_Yaml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yaml"), []byte("version: '3'"), 0644)

	result := FindComposeFile(dir)
	expected := filepath.Join(dir, "docker-compose.yaml")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindComposeFile_Dev(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.dev.yml"), []byte("version: '3'"), 0644)

	result := FindComposeFile(dir)
	expected := filepath.Join(dir, "docker-compose.dev.yml")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindComposeFile_PreferExact(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
	os.WriteFile(filepath.Join(dir, "docker-compose.dev.yml"), []byte("version: '3'"), 0644)

	result := FindComposeFile(dir)
	expected := filepath.Join(dir, "docker-compose.yml")
	if result != expected {
		t.Errorf("expected exact match %s, got %s", expected, result)
	}
}

func TestFindComposeFile_NotFound(t *testing.T) {
	dir := t.TempDir()

	result := FindComposeFile(dir)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}
