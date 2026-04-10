package docker

import (
	"testing"

	"devhub/internal/testutil"
)

func TestContainers_JSONArray(t *testing.T) {
	jsonOutput := `[{"Name":"myapp-web-1","Image":"nginx:latest","Status":"Up 2 hours","State":"running","Service":"web","Ports":"0.0.0.0:80->80/tcp"},{"Name":"myapp-db-1","Image":"postgres:15","Status":"Up 2 hours","State":"running","Service":"db","Ports":"5432/tcp"}]`

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: jsonOutput},
	}}

	svc := NewDockerService(mock)
	containers, err := svc.Containers("/project/docker-compose.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].Name != "web" {
		t.Errorf("expected service name web, got %s", containers[0].Name)
	}
	if containers[0].State != "running" {
		t.Errorf("expected state running, got %s", containers[0].State)
	}
	if containers[1].Name != "db" {
		t.Errorf("expected service name db, got %s", containers[1].Name)
	}
}

func TestContainers_JSONLines(t *testing.T) {
	jsonOutput := `{"Name":"myapp-web-1","Image":"nginx","Status":"Up","State":"running","Service":"web","Ports":"80/tcp"}
{"Name":"myapp-redis-1","Image":"redis:7","Status":"Up","State":"running","Service":"redis","Ports":"6379/tcp"}`

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: jsonOutput},
	}}

	svc := NewDockerService(mock)
	containers, err := svc.Containers("/project/docker-compose.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].Name != "web" || containers[1].Name != "redis" {
		t.Errorf("unexpected names: %s, %s", containers[0].Name, containers[1].Name)
	}
}

func TestContainers_Empty(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	containers, err := svc.Containers("/project/docker-compose.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containers != nil {
		t.Errorf("expected nil containers, got %v", containers)
	}
}

func TestAction_Start(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "web", "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_Stop(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "web", "stop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_Restart(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "web", "restart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_Up(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "web", "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_StartAll(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "", "start-all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_StopAll(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: ""},
	}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "", "stop-all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAction_Invalid(t *testing.T) {
	mock := &testutil.MockRunner{Calls: []testutil.MockCall{}}

	svc := NewDockerService(mock)
	err := svc.Action("/project/docker-compose.yml", "web", "invalid-action")
	if err == nil {
		t.Error("expected error for invalid action, got nil")
	}
}

func TestLogs(t *testing.T) {
	logOutput := "2024-01-15 10:00:00 Starting server...\n2024-01-15 10:00:01 Server ready\n"

	mock := &testutil.MockRunner{Calls: []testutil.MockCall{
		{Output: logOutput},
	}}

	svc := NewDockerService(mock)
	logs, err := svc.Logs("/project/docker-compose.yml", "web", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs != logOutput {
		t.Errorf("unexpected logs output")
	}
}
