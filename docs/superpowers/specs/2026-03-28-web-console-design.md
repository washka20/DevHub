# Web Console -- Full System Terminal in DevHub

## Problem

DevHub manages projects (git, docker, makefile commands) but lacks a terminal. Users must switch between the browser and a separate terminal app. Adding a web-based terminal lets users execute arbitrary shell commands and run interactive tools (including Claude Code CLI) directly from DevHub.

## Requirements

- Full system terminal in the browser (equivalent to a native terminal)
- PTY-backed: `isatty()=true`, full ANSI/color/cursor support, interactive programs work (vim, htop, claude)
- Tabbed sessions with optional split panels (horizontal/vertical)
- CWD defaults to the selected project's directory
- Sessions persist when navigating to other DevHub tabs (Git, Docker) and back
- Max 10 concurrent sessions (configurable)

## Approach

**Backend**: `creack/pty` for PTY allocation + dedicated WebSocket endpoint for bidirectional binary I/O.

**Frontend**: `xterm.js` terminal emulator + `splitpanes` for drag-resizable split panels + custom tab bar + Pinia store for session persistence.

The terminal WebSocket endpoint is **separate** from the existing Hub (`/api/ws`) because the Hub broadcasts JSON events (one-to-many) while the terminal needs 1:1 binary frame streaming.

## Architecture

```
Browser (xterm.js per pane)
    |
    | Binary WS frames: keystrokes / PTY output
    | Text WS frames: JSON control (resize, exit)
    |
Go Backend
    |
    | /api/terminal/ws/{sessionID}  (separate from /api/ws)
    |
terminal.Manager
    |
    | PTY sessions (creack/pty)
    |
bash / zsh / claude / any program
```

## Backend Design

### PTY Session Manager -- `internal/terminal/terminal.go`

```go
type Session struct {
    ID        string
    Cmd       *exec.Cmd
    Pty       *os.File   // master side from creack/pty
    CreatedAt time.Time
    CWD       string
    mu        sync.Mutex
    closed    bool
}

type Manager struct {
    sessions    map[string]*Session
    mu          sync.RWMutex
    maxSessions int
}
```

**Manager API:**
- `Create(id, shell, cwd, cols, rows)` -- `pty.StartWithSize()`, env: `TERM=xterm-256color`, `COLORTERM=truecolor`
- `Get(id)` -- lookup session
- `Resize(id, cols, rows)` -- `pty.Setsize()`
- `Destroy(id)` -- SIGHUP -> wait 2s -> SIGKILL -> close PTY fd
- `DestroyAll()` -- called on server shutdown
- `Count()` -- for enforcing max sessions limit

### Terminal WebSocket Handler -- `internal/api/terminal_ws.go`

Endpoint: `/api/terminal/ws/{sessionID}`

**Protocol:**

| Direction | Frame Type | Content |
|-----------|-----------|---------|
| Client -> Server | Binary | Raw keystrokes (UTF-8 bytes -> PTY stdin) |
| Client -> Server | Text | `{"type":"resize","cols":120,"rows":40}` |
| Server -> Client | Binary | Raw PTY output (bytes -> xterm.js write) |
| Server -> Client | Text | `{"type":"exit","code":0}` |

**Goroutines per connection:**
1. **PTY reader**: `pty.Read(buf[4096])` -> binary WS frame. NOT line-buffered. On PTY EOF: send exit event, close WS.
2. **WS reader**: binary WS frame -> `pty.Write()`. Text JSON -> parse control messages (resize).

**Cleanup**: When WS disconnects, destroy the associated PTY session as a safety net (handles browser crash, tab close). Explicit tab close uses REST `DELETE /api/terminal/sessions/{id}`. With `keep-alive`, WS connections stay alive during route navigation -- no reconnection needed.

### REST Endpoints -- `internal/api/terminal_handlers.go`

- `POST /api/terminal/sessions` -- Create session. Body: `{cols, rows, cwd?}`. Returns `{session_id, shell}`.
- `GET /api/terminal/sessions` -- List active sessions.
- `DELETE /api/terminal/sessions/{id}` -- Destroy session.

### Server Changes -- `internal/server/server.go`

- Create `terminal.NewManager(maxSessions)` in `New()`
- Register routes on `apiRouter`:
  - `/terminal/sessions` (GET, POST)
  - `/terminal/sessions/{id}` (DELETE)
  - `/terminal/ws/{id}` (WebSocket upgrade)

### Main Changes -- `cmd/main.go`

- Add signal handling (SIGINT/SIGTERM) for `manager.DestroyAll()` on shutdown

### Config Changes -- `internal/config/config.go`

```go
type TerminalConfig struct {
    MaxSessions int `yaml:"max_sessions"` // default: 10
}
```

### Dependencies

- Add `github.com/creack/pty` to `go.mod`

## Frontend Design

### Component Tree

```
ConsoleView.vue (/console)
  ├── TerminalTabBar.vue        -- [bash] [claude] [+] [x] | [Split H] [Split V]
  └── TerminalPanels.vue        -- manages split layout
      └── Splitpanes
          ├── Pane -> WebTerminal.vue  (xterm.js #1)
          └── Pane -> WebTerminal.vue  (xterm.js #2)
```

### Pinia Store -- `stores/terminal.ts`

```typescript
interface TerminalSession {
  id: string           // UUID from backend
  label: string        // "bash", "claude", etc.
  cwd: string
}

interface TerminalPane {
  id: string
  sessionId: string
}

interface TerminalTab {
  id: string
  label: string
  panes: TerminalPane[]
  splitDirection: 'horizontal' | 'vertical' | null
}

// State
sessions: Map<string, TerminalSession>
tabs: TerminalTab[]
activeTabId: string | null
```

Store is global (Pinia) so sessions survive navigation between DevHub routes. WebSocket connections are managed per-component (connect on mount, disconnect on unmount, reconnect on re-mount).

### WebTerminal.vue

Props: `sessionId: string`

- Mount: create xterm.js Terminal, FitAddon, WebLinksAddon, WebglAddon. Connect WS to `/api/terminal/ws/{sessionId}`.
- `term.onData` -> WS binary frame (keystrokes)
- WS binary -> `term.write()` (PTY output)
- FitAddon + ResizeObserver -> WS text frame `{type:"resize",cols,rows}`
- Unmount (tab/pane closed by user): dispose xterm instance, disconnect WS. Backend session destroyed via REST DELETE from store action or by WS disconnect safety net.

Theme matches DevHub:
```typescript
theme: {
  background: '#0d1117',
  foreground: '#c9d1d9',
  cursor: '#58a6ff',
  selectionBackground: 'rgba(88,166,255,0.3)',
}
```

### TerminalTabBar.vue

- Renders tabs from store
- Active tab highlighted (border-bottom matches terminal bg)
- "+" button creates new session (POST /api/terminal/sessions)
- "x" on each tab destroys session (DELETE) and removes from store
- Right side: Split H / Split V buttons

### Split Panels

Uses `splitpanes` npm package for drag-resizable panes.

- Default: single pane (no split)
- "Split H": adds second pane to the right
- "Split V": adds second pane below
- Each pane has its own session and WebTerminal instance
- Close button on pane header removes pane (and destroys its session)
- When only 1 pane remains, split mode deactivates

### Navigation Persistence

`ConsoleView` wrapped in `<keep-alive>` in App.vue router-view to prevent unmount on route change. Alternative: manage WS connections in store and reconnect on mount.

Chosen approach: **keep-alive** -- simpler, xterm.js instances stay alive, no reconnection needed.

### New Dependencies (package.json)

```
@xterm/xterm
@xterm/addon-fit
@xterm/addon-web-links
@xterm/addon-webgl
splitpanes
```

### Route & Sidebar

- Route: `/console` -> `ConsoleView.vue`
- Sidebar: "Console" nav item after Docker

### Vite Proxy

```typescript
'/api/terminal/ws': { target: 'ws://localhost:9000', ws: true }
```

Must be before the generic `/api` proxy rule.

## Security

- DevHub binds to `127.0.0.1` only -- local access, same threat model as an open terminal
- Max sessions limit prevents resource exhaustion
- PTY cleanup on WS disconnect and server shutdown
- No authentication needed (localhost dev tool)

## File Summary

| File | Type | ~Lines |
|------|------|--------|
| `internal/terminal/terminal.go` | New | 200 |
| `internal/api/terminal_ws.go` | New | 180 |
| `internal/api/terminal_handlers.go` | New | 100 |
| `internal/server/server.go` | Modified | +20 |
| `cmd/main.go` | Modified | +15 |
| `internal/config/config.go` | Modified | +10 |
| `go.mod` | Modified | +1 |
| `frontend/src/stores/terminal.ts` | New | 150 |
| `frontend/src/composables/useTerminal.ts` | New | 120 |
| `frontend/src/components/WebTerminal.vue` | New | 150 |
| `frontend/src/components/TerminalTabBar.vue` | New | 120 |
| `frontend/src/components/TerminalPanels.vue` | New | 100 |
| `frontend/src/views/ConsoleView.vue` | New | 80 |
| `frontend/src/router/index.ts` | Modified | +5 |
| `frontend/src/components/AppSidebar.vue` | Modified | +4 |
| `frontend/vite.config.ts` | Modified | +4 |
| `frontend/package.json` | Modified | +5 |
| **Total** | | **~1,264 new + ~64 modified** |

## Verification

1. `make dev` -- start project
2. Navigate to `/console` -- terminal opens in project CWD
3. Run `ls`, `echo hello` -- basic commands work
4. Run `vim` or `htop` -- interactive TUI works
5. Run `claude` -- full Claude Code TUI with colors, prompts, markdown
6. Click "Split H" -- second pane appears with drag-resize handle
7. Create new tab via "+" -- switches to new session
8. Navigate to /git and back to /console -- sessions preserved
9. Close tab via "x" -- session destroyed on backend
10. Resize browser window -- terminal re-fits correctly
