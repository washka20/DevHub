package terminal

import (
	"errors"
	"fmt"
	"io"
	"log"
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

	mu     sync.Mutex
	closed bool

	// Output routing: pump writes PTY data via this function.
	outputMu sync.Mutex
	outputFn func([]byte)

	// Shell lifecycle: exitCh is closed when the shell process exits.
	exitOnce sync.Once
	exitCh   chan struct{}
	exitCode int
}

// startPump runs a goroutine that reads PTY output, writes to the log file,
// and forwards to the attached output function. Runs for the session lifetime.
func (s *Session) startPump() {
	s.exitCh = make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := s.Pty.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])

				// Always write to log first (data survives WS disconnects)
				if s.LogFile != nil {
					s.LogFile.Write(data)
				}

				s.outputMu.Lock()
				fn := s.outputFn
				s.outputMu.Unlock()
				if fn != nil {
					fn(data)
				}
			}
			if err != nil {
				code := 0
				if err != io.EOF {
					code = 1
				}
				s.signalExit(code)
				return
			}
		}
	}()
}

func (s *Session) signalExit(code int) {
	s.exitOnce.Do(func() {
		s.exitCode = code
		close(s.exitCh)
	})
}

// AttachOutput sets the function called for each PTY output chunk.
// The previous output (if any) is replaced immediately.
func (s *Session) AttachOutput(fn func([]byte)) {
	s.outputMu.Lock()
	s.outputFn = fn
	s.outputMu.Unlock()
}

// DetachOutput clears the output function so PTY data only goes to the log.
func (s *Session) DetachOutput() {
	s.outputMu.Lock()
	s.outputFn = nil
	s.outputMu.Unlock()
}

// ExitCh returns a channel that is closed when the shell process exits.
func (s *Session) ExitCh() <-chan struct{} {
	return s.exitCh
}

// ExitCode returns the exit code (0=clean, 1=error). Valid after ExitCh closes.
func (s *Session) ExitCode() int {
	return s.exitCode
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
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	s.DetachOutput()

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
	// Ensure exitCh is closed even if the pump hasn't detected the exit yet
	s.signalExit(1)

	if s.LogFile != nil {
		logPath := s.LogFile.Name()
		s.LogFile.Close()
		os.Remove(logPath) // best-effort cleanup
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
	sess.startPump()
	m.sessions[id] = sess

	log.Printf("terminal: session %s created (shell=%s, cwd=%s)", id, shell, cwd)
	return sess, nil
}

// CreateWithCommand creates a session running an arbitrary command with args.
func (m *Manager) CreateWithCommand(id, cwd string, cols, rows uint16, name string, args ...string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, ErrMaxSessions
	}

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}

	var logFile *os.File
	logDir := filepath.Join(os.TempDir(), "devhub-terminal-logs")
	if err := os.MkdirAll(logDir, 0700); err == nil {
		logFile, _ = os.Create(filepath.Join(logDir, id+".log"))
	}

	sess := &Session{
		ID: id, Cmd: cmd, Pty: ptmx, LogFile: logFile, CreatedAt: time.Now(), CWD: cwd,
	}
	sess.startPump()
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
