package makefile

import (
	"bufio"
	"os"
	"strings"
)

// Command represents a single make target.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Parse reads a Makefile and extracts available targets with descriptions.
func Parse(makefilePath string) ([]Command, error) {
	f, err := os.Open(makefilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var commands []Command
	var pendingComment string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Capture description comments (## comment)
		if strings.HasPrefix(line, "##") {
			pendingComment = strings.TrimSpace(strings.TrimPrefix(line, "##"))
			continue
		}

		// Match target lines: "targetname:" or "targetname: deps"
		if idx := strings.Index(line, ":"); idx > 0 {
			target := line[:idx]

			// Skip if line starts with tab/space (recipe line) or has = (variable)
			if strings.ContainsAny(target, " \t=") {
				pendingComment = ""
				continue
			}

			// Skip internal targets starting with . or _
			if strings.HasPrefix(target, ".") || strings.HasPrefix(target, "_") {
				pendingComment = ""
				continue
			}

			commands = append(commands, Command{
				Name:        target,
				Description: pendingComment,
				Category:    categorize(target),
			})
			pendingComment = ""
			continue
		}

		pendingComment = ""
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commands, nil
}

// categorize determines the category based on the target name prefix.
func categorize(name string) string {
	switch {
	case strings.HasPrefix(name, "npm-") || strings.HasPrefix(name, "npm_"):
		return "NPM"
	case strings.HasPrefix(name, "composer-") || strings.HasPrefix(name, "composer_"):
		return "Composer"
	case name == "docker" || name == "up" || name == "down" || name == "start" || name == "stop":
		return "Docker"
	case strings.HasPrefix(name, "docker"):
		return "Docker"
	case name == "migrate" || name == "cache" || name == "winter" ||
		strings.HasPrefix(name, "migrate") || strings.HasPrefix(name, "cache") || strings.HasPrefix(name, "winter"):
		return "PHP"
	case name == "pull" || name == "submodules" ||
		strings.HasPrefix(name, "pull") || strings.HasPrefix(name, "submodules"):
		return "Git"
	case name == "init" || name == "fresh" || name == "env" ||
		strings.HasPrefix(name, "init") || strings.HasPrefix(name, "fresh") || strings.HasPrefix(name, "env"):
		return "Init"
	default:
		return "Other"
	}
}
