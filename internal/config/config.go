package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TerminalConfig holds terminal-related settings.
type TerminalConfig struct {
	MaxSessions int    `yaml:"max_sessions"`
	Shell       string `yaml:"shell"`
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
			Shell:       "", // empty = auto-detect from $SHELL
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
		cfg.ProjectsDir = ExpandHome(cfg.ProjectsDir)
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ProjectsDir = ExpandHome(cfg.ProjectsDir)
	return cfg, nil
}

// Save writes the current config back to ~/.devhub.yaml.
// Before saving, home directory paths are unexpanded back to ~ for readability.
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Make a copy with unexpanded paths for storage
	toSave := *c
	toSave.ProjectsDir = unexpandHome(toSave.ProjectsDir, home)

	data, err := yaml.Marshal(&toSave)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(home, ".devhub.yaml"), data, 0644)
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// unexpandHome replaces the home directory prefix with ~ for readability.
func unexpandHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~/" + path[len(home)+1:]
	}
	return path
}
