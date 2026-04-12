package config

import (
	"bufio"
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

// GitLabConfig holds GitLab API connection settings.
type GitLabConfig struct {
	URL     string `yaml:"url"`
	Token   string `yaml:"token"`
	Enabled bool   `yaml:"enabled"`
}

// ServicesConfig holds external service integrations.
type ServicesConfig struct {
	GitLab GitLabConfig `yaml:"gitlab"`
}

// Config holds application configuration.
type Config struct {
	Host           string         `yaml:"host"`
	Port           int            `yaml:"port"`
	ProjectsDir    string         `yaml:"projects_dir"`
	DefaultProject string         `yaml:"default_project"`
	Terminal       TerminalConfig `yaml:"terminal"`
	Services       ServicesConfig `yaml:"services"`
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
		Services: ServicesConfig{
			GitLab: GitLabConfig{
				Enabled: false,
			},
		},
	}
}

// loadDotEnv loads KEY=VALUE pairs from a .env file into os environment.
// Existing env vars are NOT overwritten.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
	}
}

// Load reads config from ~/.devhub.yaml. If the file doesn't exist,
// default values are used.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Load .env from working directory or parent (Makefile does cd cmd/)
	loadDotEnv(".env")
	loadDotEnv("../.env")

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".devhub.yaml"))
	if err != nil {
		// File not found — use defaults
		cfg.ProjectsDir = ExpandHome(cfg.ProjectsDir)
		applyEnvOverrides(cfg)
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ProjectsDir = ExpandHome(cfg.ProjectsDir)
	applyEnvOverrides(cfg)
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

// applyEnvOverrides overrides config values from environment variables.
// Env vars take precedence over YAML config.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DEVHUB_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("DEVHUB_GITLAB_URL"); v != "" {
		cfg.Services.GitLab.URL = v
	}
	if v := os.Getenv("DEVHUB_GITLAB_TOKEN"); v != "" {
		cfg.Services.GitLab.Token = v
		cfg.Services.GitLab.Enabled = true
	}
	if os.Getenv("DEVHUB_GITLAB_ENABLED") == "true" {
		cfg.Services.GitLab.Enabled = true
	}
	if os.Getenv("DEVHUB_GITLAB_ENABLED") == "false" {
		cfg.Services.GitLab.Enabled = false
	}
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
