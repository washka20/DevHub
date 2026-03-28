package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Session struct {
	ID        string
	Cmd       *exec.Cmd
	Pty       *os.File
	LogFile   *os.File // persistent log of all PTY output
	CreatedAt time.Time
	CWD       string
	mu        sync.Mutex
	closed    bool
	readerCh  chan struct{} // closed to signal current reader goroutine to stop
}

// StopReader signals the current PTY reader goroutine to exit.
func (s *Session) StopReader() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readerCh != nil {
		close(s.readerCh)
		s.readerCh = nil
	}
}

// StartReader creates a stop channel for a new reader goroutine.
// Must be called before starting the goroutine.
func (s *Session) StartReader() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan struct{})
	s.readerCh = ch
	return ch
}

func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("session %s is closed", s.ID)
	}
	return pty.Setsize(s.Pty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true

	if s.Cmd.Process != nil {
		s.Cmd.Process.Signal(syscall.SIGHUP)

		done := make(chan struct{})
		go func() {
			s.Cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			s.Cmd.Process.Kill()
			<-done
		}
	}

	s.Pty.Close()
	if s.LogFile != nil {
		s.LogFile.Close()
	}
}

type SessionInfo struct {
	ID        string `json:"id"`
	CWD       string `json:"cwd"`
	CreatedAt string `json:"created_at"`
}

type Manager struct {
	sessions    map[string]*Session
	mu          sync.RWMutex
	maxSessions int
}

// ErrMaxSessions is returned when the session limit is reached.
var ErrMaxSessions = errors.New("max sessions limit reached")

func NewManager(maxSessions int) *Manager {
	return &Manager{
		sessions:    make(map[string]*Session),
		maxSessions: maxSessions,
	}
}

func (m *Manager) Create(id, shell, cwd string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, ErrMaxSessions
	}

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	cmd := exec.Command(shell)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}

	// Create log file for persistent session output
	var logFile *os.File
	logDir := filepath.Join(os.TempDir(), "devhub-terminal-logs")
	if err := os.MkdirAll(logDir, 0700); err == nil {
		logFile, _ = os.Create(filepath.Join(logDir, id+".log"))
	}

	sess := &Session{
		ID: id, Cmd: cmd, Pty: ptmx, LogFile: logFile, CreatedAt: time.Now(), CWD: cwd,
	}
	m.sessions[id] = sess
	return sess, nil
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *Manager) Destroy(id string) {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if ok {
		sess.Close()
	}
}

func (m *Manager) DestroyAll() {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, s := range sessions {
		s.Close()
	}
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

func (m *Manager) MaxSessions() int {
	return m.maxSessions
}

func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, SessionInfo{
			ID: s.ID, CWD: s.CWD, CreatedAt: s.CreatedAt.Format(time.RFC3339),
		})
	}
	return list
}
