package executor

import (
	"context"
	"testing"
	"time"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()
	outCh, errCh := Execute(ctx, "", "echo", "hello")

	var lines []string
	for line := range outCh {
		lines = append(lines, line)
	}

	err := <-errCh
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [hello], got %v", lines)
	}
}

func TestMultilineOutput(t *testing.T) {
	ctx := context.Background()
	outCh, errCh := Execute(ctx, "", "printf", "a\nb\nc\n")

	var lines []string
	for line := range outCh {
		lines = append(lines, line)
	}

	err := <-errCh
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("expected [a b c], got %v", lines)
	}
}

func TestExitError(t *testing.T) {
	ctx := context.Background()
	outCh, errCh := Execute(ctx, "", "false")

	// Drain output
	for range outCh {
	}

	err := <-errCh
	if err == nil {
		t.Error("expected error for `false` command, got nil")
	}
}

func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	outCh, errCh := Execute(ctx, "", "sleep", "10")

	// Drain output
	for range outCh {
	}

	err := <-errCh
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
}

func TestInvalidCommand(t *testing.T) {
	ctx := context.Background()
	outCh, errCh := Execute(ctx, "", "nonexistent_command_xyz_12345")

	// Drain output
	for range outCh {
	}

	err := <-errCh
	if err == nil {
		t.Error("expected error for invalid command, got nil")
	}
}

func TestWorkingDir(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	outCh, errCh := Execute(ctx, dir, "pwd")

	var lines []string
	for line := range outCh {
		lines = append(lines, line)
	}

	err := <-errCh
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(lines) != 1 || lines[0] != dir {
		t.Errorf("expected working dir %s, got %v", dir, lines)
	}
}
