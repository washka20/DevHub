package git

import (
	"testing"
)

// TestBuildGraphRows_LinearHistory проверяет линейную историю из 3 коммитов.
// Все коммиты должны быть на колонке 0.
func TestBuildGraphRows_LinearHistory(t *testing.T) {
	commits := []Commit{
		{Hash: "aaa", Parents: []string{"bbb"}},
		{Hash: "bbb", Parents: []string{"ccc"}},
		{Hash: "ccc", Parents: nil},
	}

	result, err := BuildGraphRows(commits)
	if err != nil {
		t.Fatalf("BuildGraphRows вернул ошибку: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("ожидалось 3 коммита, получено %d", len(result))
	}

	for i, c := range result {
		if c.GraphData == nil {
			t.Errorf("коммит %d (%s): GraphData == nil", i, c.Hash)
			continue
		}
		if c.GraphData.Column != 0 {
			t.Errorf("коммит %d (%s): ожидалась колонка 0, получена %d", i, c.Hash, c.GraphData.Column)
		}
		if c.GraphData.Color == "" {
			t.Errorf("коммит %d (%s): цвет не должен быть пустым", i, c.Hash)
		}
	}
}

// TestBuildGraphRows_WithBranch проверяет мерж-коммит с двумя родителями.
// У мерж-коммита должны быть линии для визуализации слияния.
func TestBuildGraphRows_WithBranch(t *testing.T) {
	// Коммит "merge" сливает "branch" и "main" — это создаёт ответвление в графе.
	commits := []Commit{
		{Hash: "merge", Parents: []string{"main", "branch"}},
		{Hash: "branch", Parents: []string{"base"}},
		{Hash: "main", Parents: []string{"base"}},
		{Hash: "base", Parents: nil},
	}

	result, err := BuildGraphRows(commits)
	if err != nil {
		t.Fatalf("BuildGraphRows вернул ошибку: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("ожидалось 4 коммита, получено %d", len(result))
	}

	// Проверяем, что все коммиты получили GraphData
	for i, c := range result {
		if c.GraphData == nil {
			t.Errorf("коммит %d (%s): GraphData == nil", i, c.Hash)
		}
	}

	// У мерж-коммита должны быть линии
	mergeCommit := result[0]
	if mergeCommit.GraphData != nil && len(mergeCommit.GraphData.Lines) == 0 {
		t.Errorf("мерж-коммит должен иметь линии для визуализации слияния")
	}
}

// TestBuildGraphRows_Empty проверяет, что nil-вход возвращает пустой результат.
func TestBuildGraphRows_Empty(t *testing.T) {
	result, err := BuildGraphRows(nil)
	if err != nil {
		t.Fatalf("BuildGraphRows вернул ошибку для nil: %v", err)
	}

	if result != nil {
		t.Errorf("ожидался nil, получено %v", result)
	}
}

// --- Тесты для BuildFullGraph ---

func TestBuildFullGraph_LinearHistory(t *testing.T) {
	topo := []TopologyNode{
		{Hash: "aaa", Parents: []string{"bbb"}},
		{Hash: "bbb", Parents: []string{"ccc"}},
		{Hash: "ccc", Parents: nil},
	}
	result, err := BuildFullGraph(topo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}
	for i, n := range result.Nodes {
		if n.GraphData.Color == "" {
			t.Errorf("node %d: expected non-empty color", i)
		}
		if n.GraphData.Column != 0 {
			t.Errorf("node %d: expected column 0, got %d", i, n.GraphData.Column)
		}
	}
}

func TestBuildFullGraph_WithMerge(t *testing.T) {
	topo := []TopologyNode{
		{Hash: "m1", Parents: []string{"a1", "b1"}},
		{Hash: "a1", Parents: []string{"root"}},
		{Hash: "b1", Parents: []string{"root"}},
		{Hash: "root", Parents: nil},
	}
	result, err := BuildFullGraph(topo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(result.Nodes))
	}
	// Merge коммит должен иметь линии
	if len(result.Nodes[0].GraphData.Lines) == 0 {
		t.Error("merge commit should have lines")
	}
	// MaxWidth должен быть > 0 (ветвление)
	if result.MaxWidth == 0 {
		t.Error("expected MaxWidth > 0 for branching graph")
	}
}

func TestBuildFullGraph_Empty(t *testing.T) {
	result, err := BuildFullGraph(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(result.Nodes))
	}
}
