package terminal

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager(5)
	if m.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", m.Count())
	}
}

func TestCreateAndGet(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("test-1", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if sess.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", sess.ID)
	}

	got, ok := m.Get("test-1")
	if !ok {
		t.Fatal("Get returned false for existing session")
	}
	if got.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", got.ID)
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("Get returned true for nonexistent session")
	}

	m.Destroy("test-1")
}

func TestMaxSessions(t *testing.T) {
	m := NewManager(2)
	_, err := m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create s1 failed: %v", err)
	}
	_, err = m.Create("s2", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create s2 failed: %v", err)
	}
	_, err = m.Create("s3", "/bin/sh", t.TempDir(), 80, 24)
	if err == nil {
		t.Error("expected error when exceeding max sessions, got nil")
	}

	m.DestroyAll()
}

func TestDestroyAll(t *testing.T) {
	m := NewManager(5)
	m.Create("s1", "/bin/sh", t.TempDir(), 80, 24)
	m.Create("s2", "/bin/sh", t.TempDir(), 80, 24)

	m.DestroyAll()

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after DestroyAll, got %d", m.Count())
	}
}

func TestPtyReadWrite(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("rw", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer m.Destroy("rw")

	_, err = sess.Pty.Write([]byte("echo hello_pty_test\n"))
	if err != nil {
		t.Fatalf("Write to PTY failed: %v", err)
	}

	buf := make([]byte, 4096)
	deadline := time.Now().Add(3 * time.Second)
	var totalRead int
	for time.Now().Before(deadline) {
		sess.Pty.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := sess.Pty.Read(buf[totalRead:])
		totalRead += n
		if err != nil {
			break
		}
		if totalRead > 0 {
			output := string(buf[:totalRead])
			if containsSubstring(output, "hello_pty_test") {
				return
			}
		}
	}
	t.Errorf("expected output containing hello_pty_test, got %q", string(buf[:totalRead]))
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestResize(t *testing.T) {
	m := NewManager(5)
	sess, err := m.Create("resize", "/bin/sh", t.TempDir(), 80, 24)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer m.Destroy("resize")

	err = sess.Resize(120, 40)
	if err != nil {
		t.Errorf("Resize failed: %v", err)
	}
}

func TestListSessions(t *testing.T) {
	m := NewManager(5)
	m.Create("a", "/bin/sh", t.TempDir(), 80, 24)
	m.Create("b", "/bin/sh", t.TempDir(), 80, 24)

	list := m.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions in list, got %d", len(list))
	}

	m.DestroyAll()
}
