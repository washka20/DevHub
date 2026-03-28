package scanner

import (
	"os"
	"path/filepath"
)

// Project describes a discovered project directory.
type Project struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsGit       bool   `json:"is_git"`
	HasMakefile bool   `json:"has_makefile"`
	HasDocker   bool   `json:"has_docker"`
}

// Scan walks projectsDir and returns a Project for each subdirectory.
// If a directory has no git/makefile/docker, it checks subdirectories (one level deeper).
func Scan(projectsDir string) ([]Project, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(projectsDir, entry.Name())
		p := scanDir(entry.Name(), dir)

		if p.IsGit || p.HasMakefile || p.HasDocker {
			projects = append(projects, p)
		} else {
			// No markers — check subdirectories
			subs := scanSubdirs(dir)
			if len(subs) > 0 {
				projects = append(projects, subs...)
			}
		}
	}

	return projects, nil
}

func scanDir(name, dir string) Project {
	return Project{
		Name:        name,
		Path:        dir,
		IsGit:       dirExists(filepath.Join(dir, ".git")),
		HasMakefile: fileExists(filepath.Join(dir, "Makefile")),
		HasDocker:   hasDockerCompose(dir),
	}
}

func scanSubdirs(parentDir string) []Project {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return nil
	}
	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		dir := filepath.Join(parentDir, entry.Name())
		p := scanDir(entry.Name(), dir)
		if p.IsGit || p.HasMakefile || p.HasDocker {
			projects = append(projects, p)
		}
	}
	return projects
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasDockerCompose checks for docker-compose files including environment-specific ones.
func hasDockerCompose(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "docker-compose*.yml"))
	if len(matches) > 0 {
		return true
	}
	matches, _ = filepath.Glob(filepath.Join(dir, "docker-compose*.yaml"))
	return len(matches) > 0
}

// FindComposeFile returns the best docker-compose file for a project.
func FindComposeFile(dir string) string {
	// Prefer exact match first
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p
		}
	}
	// Then development
	for _, name := range []string{"docker-compose.development.yml", "docker-compose.dev.yml"} {
		p := filepath.Join(dir, name)
		if fileExists(p) {
			return p
		}
	}
	// Any docker-compose*.yml
	matches, _ := filepath.Glob(filepath.Join(dir, "docker-compose*.yml"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
