package search

import (
	"testing"
)

func TestParseRgJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name: "single match",
			input: `{"type":"begin","data":{"path":{"text":"./main.go"}}}
{"type":"match","data":{"path":{"text":"./main.go"},"lines":{"text":"func main() {\n"},"line_number":10,"submatches":[{"match":{"text":"main"},"start":5,"end":9}]}}
{"type":"end","data":{"path":{"text":"./main.go"},"stats":{"matched_lines":1}}}`,
			want: 1,
		},
		{
			name: "multiple matches",
			input: `{"type":"match","data":{"path":{"text":"./a.go"},"lines":{"text":"foo bar\n"},"line_number":1,"submatches":[{"match":{"text":"foo"},"start":0,"end":3}]}}
{"type":"match","data":{"path":{"text":"./b.go"},"lines":{"text":"foo baz\n"},"line_number":5,"submatches":[{"match":{"text":"foo"},"start":0,"end":3}]}}`,
			want: 2,
		},
		{
			name:  "no matches",
			input: `{"type":"summary","data":{"stats":{"matched_lines":0}}}`,
			want:  0,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results, err := parseRgJSON("/tmp", tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRgJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(results) != tt.want {
				t.Errorf("parseRgJSON() got %d results, want %d", len(results), tt.want)
			}
		})
	}
}

func TestParseRgJSON_FieldValues(t *testing.T) {
	t.Parallel()

	input := `{"type":"match","data":{"path":{"text":"./src/handler.go"},"lines":{"text":"func HandleSearch() {\n"},"line_number":42,"submatches":[{"match":{"text":"Search"},"start":11,"end":17}]}}`

	results, err := parseRgJSON("/project", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.File != "src/handler.go" {
		t.Errorf("File = %q, want %q", r.File, "src/handler.go")
	}
	if r.Line != 42 {
		t.Errorf("Line = %d, want %d", r.Line, 42)
	}
	if r.Column != 11 {
		t.Errorf("Column = %d, want %d", r.Column, 11)
	}
	if r.Content != "func HandleSearch() {" {
		t.Errorf("Content = %q, want %q", r.Content, "func HandleSearch() {")
	}
}

func TestParseGrepOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "single match",
			input: "./main.go:10:func main() {",
			want:  1,
		},
		{
			name:  "multiple matches",
			input: "./a.go:1:foo bar\n./b.go:5:foo baz",
			want:  2,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
		{
			name:  "malformed line skipped",
			input: "no-colon-here",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := parseGrepOutput("/tmp", tt.input)
			if len(results) != tt.want {
				t.Errorf("parseGrepOutput() got %d results, want %d", len(results), tt.want)
			}
		})
	}
}

func TestParseGrepOutput_FieldValues(t *testing.T) {
	t.Parallel()

	input := "./src/handler.go:42:func HandleSearch() {"
	results := parseGrepOutput("/project", input)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.File != "src/handler.go" {
		t.Errorf("File = %q, want %q", r.File, "src/handler.go")
	}
	if r.Line != 42 {
		t.Errorf("Line = %d, want %d", r.Line, 42)
	}
	if r.Column != 0 {
		t.Errorf("Column = %d, want %d", r.Column, 0)
	}
	if r.Content != "func HandleSearch() {" {
		t.Errorf("Content = %q, want %q", r.Content, "func HandleSearch() {")
	}
}
