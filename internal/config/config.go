package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TerminalConfig holds terminal-related settings.
type TerminalConfig struct {
	MaxSessions int `yaml:"max_sessions"`
}

// Config holds application configuration.
type Config struct {
	Port           int            `yaml:"port"`
	ProjectsDir    string         `yaml:"projects_dir"`
	DefaultProject string         `yaml:"default_project"`
	Terminal       TerminalConfig `yaml:"terminal"`
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Port:           9000,
		ProjectsDir:    "~/project",
		DefaultProject: "cfa",
		Terminal: TerminalConfig{
			MaxSessions: 10,
		},
	}
}

// Load reads config from ~/.devhub.yaml. If the file doesn't exist,
// default values are used.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".devhub.yaml"))
	if err != nil {
		// File not found — use defaults
		cfg.ProjectsDir = expandHome(cfg.ProjectsDir)
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ProjectsDir = expandHome(cfg.ProjectsDir)
	return cfg, nil
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
