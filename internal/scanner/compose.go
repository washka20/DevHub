package scanner

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ComposeFile describes a single docker-compose file detected for a project.
type ComposeFile struct {
	// Path is the filename (relative to the project root), e.g. "docker-compose.dev.yml".
	Path string `json:"path"`
	// Services is the list of top-level service names declared in this file.
	Services []string `json:"services"`
	// Profiles is the union of every `profiles:` array across services in this file.
	Profiles []string `json:"profiles"`
}

// composeDoc is the minimal YAML shape we need to inspect a compose file.
type composeDoc struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Profiles []string `yaml:"profiles"`
}

// FindComposeFiles returns every `docker-compose*.{yml,yaml}` file in dir with its
// services/profiles parsed. The canonical `docker-compose.yml` (or `.yaml`) is
// always first; the rest are sorted lexicographically for stable UI order.
// A broken YAML file is still returned with empty Services/Profiles so the user
// can see it exists rather than silently ignoring it.
func FindComposeFiles(dir string) []ComposeFile {
	seen := map[string]bool{}
	var matches []string

	for _, pattern := range []string{"docker-compose*.yml", "docker-compose*.yaml"} {
		found, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, f := range found {
			if !seen[f] {
				seen[f] = true
				matches = append(matches, f)
			}
		}
	}
	if len(matches) == 0 {
		return nil
	}

	// Stable order: docker-compose.yml first, then .yaml, then alphabetical for the rest.
	sort.SliceStable(matches, func(i, j int) bool {
		bi := filepath.Base(matches[i])
		bj := filepath.Base(matches[j])
		return composeRank(bi) < composeRank(bj) ||
			(composeRank(bi) == composeRank(bj) && bi < bj)
	})

	result := make([]ComposeFile, 0, len(matches))
	for _, full := range matches {
		rel := filepath.Base(full)
		result = append(result, parseComposeFile(full, rel))
	}
	return result
}

func composeRank(base string) int {
	switch base {
	case "docker-compose.yml":
		return 0
	case "docker-compose.yaml":
		return 1
	}
	return 10
}

func parseComposeFile(full, rel string) ComposeFile {
	cf := ComposeFile{Path: rel}

	raw, err := os.ReadFile(full)
	if err != nil {
		return cf
	}

	var doc composeDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return cf
	}

	// Collect service names and the union of their profiles.
	names := make([]string, 0, len(doc.Services))
	profileSet := map[string]struct{}{}
	for name, svc := range doc.Services {
		names = append(names, name)
		for _, p := range svc.Profiles {
			profileSet[p] = struct{}{}
		}
	}
	sort.Strings(names)
	cf.Services = names

	profiles := make([]string, 0, len(profileSet))
	for p := range profileSet {
		profiles = append(profiles, p)
	}
	sort.Strings(profiles)
	cf.Profiles = profiles

	return cf
}
