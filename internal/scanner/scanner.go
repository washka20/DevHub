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
	Group       string `json:"group,omitempty"`
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
			// Directory is a project itself — always show it
			projects = append(projects, p)
		}

		// Also check for sub-projects (grouped by parent name)
		subs := scanSubdirs(entry.Name(), dir)
		if len(subs) > 0 {
			projects = append(projects, subs...)
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

func scanSubdirs(groupName, parentDir string) []Project {
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
			p.Group = groupName
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

// FindComposeFile returns the absolute path of the best docker-compose file for
// a project. Kept as a thin wrapper over FindComposeFiles for backward
// compatibility with callers that only need a single file.
func FindComposeFile(dir string) string {
	files := FindComposeFiles(dir)
	if len(files) == 0 {
		// Fallback to development-named files for projects that only ship them.
		for _, name := range []string{"docker-compose.development.yml", "docker-compose.dev.yml"} {
			p := filepath.Join(dir, name)
			if fileExists(p) {
				return p
			}
		}
		return ""
	}
	return filepath.Join(dir, files[0].Path)
}

// DefaultComposeFiles returns the files that should be used by default when the
// caller did not pass an explicit stack. Prefers `docker-compose.yml` alone; if
// it's not there, falls back to the first detected compose file.
func DefaultComposeFiles(dir string) []string {
	files := FindComposeFiles(dir)
	if len(files) == 0 {
		return nil
	}
	return []string{files[0].Path}
}
