package search

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"devhub/internal/runner"
)

// SearchResult represents a single match found in a file.
type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Content string `json:"content"`
}

// SearchService performs text search across project files.
type SearchService struct {
	runner runner.CommandRunner
}

// NewSearchService creates a new SearchService.
func NewSearchService(r runner.CommandRunner) *SearchService {
	return &SearchService{runner: r}
}

const maxResults = 100

// Search performs a text search in dir using ripgrep (preferred) or grep as fallback.
func (s *SearchService) Search(dir, query, glob string) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}

	if _, err := exec.LookPath("rg"); err == nil {
		return s.searchRg(dir, query, glob)
	}
	return s.searchGrep(dir, query, glob)
}

func (s *SearchService) searchRg(dir, query, glob string) ([]SearchResult, error) {
	args := []string{
		"--json",
		"-m", fmt.Sprintf("%d", maxResults),
		"--glob", "!.git",
		"--glob", "!node_modules",
		"--glob", "!vendor",
		"--glob", "!dist",
	}
	if glob != "" {
		args = append(args, "--glob", glob)
	}
	args = append(args, query, ".")

	out, err := s.runner.Run(dir, "rg", args...)
	if err != nil {
		// rg exits 1 when no matches found — that's not an error
		if out == "" || strings.TrimSpace(out) == "" {
			return []SearchResult{}, nil
		}
	}

	return parseRgJSON(dir, out)
}

// rgMessage represents a single JSON line from rg --json output.
type rgMessage struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		LineNumber int `json:"line_number"`
		Lines      struct {
			Text string `json:"text"`
		} `json:"lines"`
		Submatches []struct {
			Start int `json:"start"`
		} `json:"submatches"`
	} `json:"data"`
}

func parseRgJSON(dir string, out string) ([]SearchResult, error) {
	var results []SearchResult
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg rgMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Type != "match" {
			continue
		}

		file := msg.Data.Path.Text
		// Strip leading "./" from paths
		file = strings.TrimPrefix(file, "./")

		col := 0
		if len(msg.Data.Submatches) > 0 {
			col = msg.Data.Submatches[0].Start
		}

		content := strings.TrimRight(msg.Data.Lines.Text, "\n\r")

		results = append(results, SearchResult{
			File:    file,
			Line:    msg.Data.LineNumber,
			Column:  col,
			Content: content,
		})

		if len(results) >= maxResults {
			break
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	return results, nil
}

func (s *SearchService) searchGrep(dir, query, glob string) ([]SearchResult, error) {
	args := []string{"-rn", "-m", fmt.Sprintf("%d", maxResults)}
	if glob != "" {
		args = append(args, "--include", glob)
	}
	args = append(args, query, ".")

	out, err := s.runner.Run(dir, "grep", args...)
	if err != nil {
		if out == "" || strings.TrimSpace(out) == "" {
			return []SearchResult{}, nil
		}
	}

	return parseGrepOutput(dir, out), nil
}

func parseGrepOutput(dir string, out string) []SearchResult {
	var results []SearchResult
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: ./path/to/file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		file := strings.TrimPrefix(parts[0], "./")
		lineNum := 0
		fmt.Sscanf(parts[1], "%d", &lineNum)
		content := parts[2]

		// Make path relative to dir
		if filepath.IsAbs(file) {
			if rel, err := filepath.Rel(dir, file); err == nil {
				file = rel
			}
		}

		results = append(results, SearchResult{
			File:    file,
			Line:    lineNum,
			Column:  0,
			Content: content,
		})

		if len(results) >= maxResults {
			break
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	return results
}
