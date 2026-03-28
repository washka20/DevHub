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
