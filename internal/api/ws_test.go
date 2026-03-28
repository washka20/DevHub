package api

import (
	"encoding/json"
	"testing"
)

func TestClient_Subscribe(t *testing.T) {
	c := &client{
		projects: make(map[string]bool),
	}
	c.subscribe("myapp")

	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()

	if !c.projects["myapp"] {
		t.Error("expected myapp to be in subscriptions")
	}
}

func TestClient_Unsubscribe(t *testing.T) {
	c := &client{
		projects: make(map[string]bool),
	}
	c.subscribe("myapp")
	c.subscribe("other")
	c.unsubscribe("myapp")

	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()

	if c.projects["myapp"] {
		t.Error("expected myapp to be removed from subscriptions")
	}
	if !c.projects["other"] {
		t.Error("expected other to still be in subscriptions")
	}
}

func TestClient_SubscribedTo(t *testing.T) {
	c := &client{
		projects: make(map[string]bool),
	}
	c.subscribe("myapp")

	if !c.subscribedTo("myapp") {
		t.Error("expected subscribedTo(myapp) = true")
	}
	if c.subscribedTo("other") {
		t.Error("expected subscribedTo(other) = false")
	}
}

func TestClient_SubscribedTo_NoSubscriptions(t *testing.T) {
	c := &client{
		projects: make(map[string]bool),
	}

	// With no subscriptions, should receive all events
	if !c.subscribedTo("anyproject") {
		t.Error("expected subscribedTo to return true when no subscriptions")
	}
	if !c.subscribedTo("") {
		t.Error("expected subscribedTo to return true for empty project")
	}
}

func TestEvent_JSON(t *testing.T) {
	event := Event{
		Type:    "git_changed",
		Project: "myapp",
		Data:    "commit",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded["type"] != "git_changed" {
		t.Errorf("expected type git_changed, got %v", decoded["type"])
	}
	if decoded["project"] != "myapp" {
		t.Errorf("expected project myapp, got %v", decoded["project"])
	}
	if decoded["data"] != "commit" {
		t.Errorf("expected data commit, got %v", decoded["data"])
	}
	// cmd should be omitted (omitempty)
	if _, exists := decoded["cmd"]; exists {
		t.Error("expected cmd to be omitted when empty")
	}
}
