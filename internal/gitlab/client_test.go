package gitlab

import (
	"encoding/json"
	"testing"
)

func TestTodoJSONParsing(t *testing.T) {
	raw := `[
		{
			"id": 123,
			"project_id": 5,
			"action_name": "assigned",
			"target_type": "Issue",
			"target": {
				"id": 10,
				"iid": 42,
				"title": "Fix login bug",
				"state": "opened",
				"web_url": "https://gitlab.example.com/group/project/-/issues/42"
			},
			"author": {
				"id": 1,
				"username": "johndoe",
				"name": "John Doe",
				"avatar_url": "https://gitlab.example.com/uploads/-/system/user/avatar/1/avatar.png"
			},
			"body": "Fix login bug",
			"state": "pending",
			"created_at": "2026-04-10T12:00:00.000Z"
		}
	]`

	var todos []Todo
	if err := json.Unmarshal([]byte(raw), &todos); err != nil {
		t.Fatalf("failed to unmarshal todos: %v", err)
	}

	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}

	todo := todos[0]

	t.Run("top-level fields", func(t *testing.T) {
		if todo.ID != 123 {
			t.Errorf("ID = %d, want 123", todo.ID)
		}
		if todo.ProjectID != 5 {
			t.Errorf("ProjectID = %d, want 5", todo.ProjectID)
		}
		if todo.ActionName != "assigned" {
			t.Errorf("ActionName = %q, want %q", todo.ActionName, "assigned")
		}
		if todo.TargetType != "Issue" {
			t.Errorf("TargetType = %q, want %q", todo.TargetType, "Issue")
		}
		if todo.State != "pending" {
			t.Errorf("State = %q, want %q", todo.State, "pending")
		}
	})

	t.Run("target fields", func(t *testing.T) {
		if todo.Target.IID != 42 {
			t.Errorf("Target.IID = %d, want 42", todo.Target.IID)
		}
		if todo.Target.Title != "Fix login bug" {
			t.Errorf("Target.Title = %q, want %q", todo.Target.Title, "Fix login bug")
		}
		if todo.Target.State != "opened" {
			t.Errorf("Target.State = %q, want %q", todo.Target.State, "opened")
		}
		if todo.Target.WebURL == "" {
			t.Error("Target.WebURL is empty")
		}
	})

	t.Run("author fields", func(t *testing.T) {
		if todo.Author.Username != "johndoe" {
			t.Errorf("Author.Username = %q, want %q", todo.Author.Username, "johndoe")
		}
		if todo.Author.Name != "John Doe" {
			t.Errorf("Author.Name = %q, want %q", todo.Author.Name, "John Doe")
		}
	})
}

func TestTodoEmptyArrayParsing(t *testing.T) {
	raw := `[]`

	var todos []Todo
	if err := json.Unmarshal([]byte(raw), &todos); err != nil {
		t.Fatalf("failed to unmarshal empty todos: %v", err)
	}

	if len(todos) != 0 {
		t.Errorf("expected 0 todos, got %d", len(todos))
	}
}
