# Settings Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Settings page to DevHub that lets users configure General (projects dir, default project, port), Terminal (shell, font, scrollback, cursor blink, max sessions), and Theme (7 presets with live preview) settings.

**Architecture:** Backend settings stored in `~/.devhub.yaml` (already exists), UI-only settings (font, theme) in localStorage via Pinia store. Two new API endpoints: `GET /api/settings` and `PUT /api/settings` for server-side config. `GET /api/settings/shells` returns available shells. Frontend reads settings store in WebTerminal.vue instead of hardcoded values.

**Tech Stack:** Go (gorilla/mux), Vue 3 (Pinia, vue-router), TypeScript, YAML

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/config/config.go` | Modify | Add `Shell` field to TerminalConfig, add `Save()` method |
| `internal/api/settings_handlers.go` | Create | GET/PUT /api/settings, GET /api/settings/shells |
| `internal/server/server.go` | Modify | Register settings routes |
| `internal/api/terminal_handlers.go` | Modify | Read shell from config instead of $SHELL |
| `frontend/src/types/index.ts` | Modify | Add settings + theme types |
| `frontend/src/data/terminal-themes.ts` | Create | 7 theme presets (colors) |
| `frontend/src/stores/settings.ts` | Create | Pinia store: fetch/save settings, localStorage for UI prefs |
| `frontend/src/views/SettingsView.vue` | Create | Settings page UI |
| `frontend/src/components/WebTerminal.vue` | Modify | Read from settings store |
| `frontend/src/router/index.ts` | Modify | Add /settings route |
| `frontend/src/components/AppSidebar.vue` | Modify | Add Settings nav item |

---

### Task 1: Backend — Extend Config + Save method

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add Shell field and Save method**

```go
// In TerminalConfig, add Shell:
type TerminalConfig struct {
	MaxSessions int    `yaml:"max_sessions"`
	Shell       string `yaml:"shell"`
}

// Update DefaultConfig to include shell default:
func DefaultConfig() *Config {
	return &Config{
		Port:           9000,
		ProjectsDir:    "~/project",
		DefaultProject: "cfa",
		Terminal: TerminalConfig{
			MaxSessions: 10,
			Shell:       "",  // empty = auto-detect from $SHELL
		},
	}
}

// Add Save method that writes config back to ~/.devhub.yaml:
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create a copy with unexpanded paths for saving
	toSave := *c
	if strings.HasPrefix(c.ProjectsDir, home) {
		toSave.ProjectsDir = "~" + c.ProjectsDir[len(home):]
	}

	data, err := yaml.Marshal(&toSave)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(home, ".devhub.yaml"), data, 0644)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Success, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(settings): добавить Shell в конфиг и метод Save()"
```

---

### Task 2: Backend — Settings API handlers

**Files:**
- Create: `internal/api/settings_handlers.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Create settings handlers**

Create `internal/api/settings_handlers.go`:

```go
package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"devhub/internal/config"
)

type SettingsHandlers struct {
	Cfg *config.Config
}

type settingsResponse struct {
	Port           int    `json:"port"`
	ProjectsDir    string `json:"projects_dir"`
	DefaultProject string `json:"default_project"`
	Terminal       terminalSettingsResponse `json:"terminal"`
}

type terminalSettingsResponse struct {
	MaxSessions int    `json:"max_sessions"`
	Shell       string `json:"shell"`
}

type updateSettingsRequest struct {
	Port           *int    `json:"port"`
	ProjectsDir    *string `json:"projects_dir"`
	DefaultProject *string `json:"default_project"`
	Terminal       *updateTerminalRequest `json:"terminal"`
}

type updateTerminalRequest struct {
	MaxSessions *int    `json:"max_sessions"`
	Shell       *string `json:"shell"`
}

// GetSettings handles GET /api/settings.
func (sh *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	resp := settingsResponse{
		Port:           sh.Cfg.Port,
		ProjectsDir:    sh.Cfg.ProjectsDir,
		DefaultProject: sh.Cfg.DefaultProject,
		Terminal: terminalSettingsResponse{
			MaxSessions: sh.Cfg.Terminal.MaxSessions,
			Shell:       sh.Cfg.Terminal.Shell,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateSettings handles PUT /api/settings.
func (sh *SettingsHandlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Port != nil {
		sh.Cfg.Port = *body.Port
	}
	if body.ProjectsDir != nil {
		sh.Cfg.ProjectsDir = config.ExpandHome(*body.ProjectsDir)
	}
	if body.DefaultProject != nil {
		sh.Cfg.DefaultProject = *body.DefaultProject
	}
	if body.Terminal != nil {
		if body.Terminal.MaxSessions != nil {
			sh.Cfg.Terminal.MaxSessions = *body.Terminal.MaxSessions
		}
		if body.Terminal.Shell != nil {
			sh.Cfg.Terminal.Shell = *body.Terminal.Shell
		}
	}

	if err := sh.Cfg.Save(); err != nil {
		jsonError(w, "failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ListShells handles GET /api/settings/shells.
func (sh *SettingsHandlers) ListShells(w http.ResponseWriter, r *http.Request) {
	shells := []string{}
	f, err := os.Open("/etc/shells")
	if err != nil {
		// fallback
		shells = append(shells, "/bin/bash")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shells)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		shells = append(shells, line)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shells)
}
```

- [ ] **Step 2: Export ExpandHome in config.go**

Rename `expandHome` to `ExpandHome` in `internal/config/config.go` (change the function name and all call sites within the file).

- [ ] **Step 3: Register routes in server.go**

Add after the terminal routes block in `internal/server/server.go`:

```go
	// Settings
	settingsH := &api.SettingsHandlers{Cfg: cfg}
	apiRouter.HandleFunc("/settings", settingsH.GetSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", settingsH.UpdateSettings).Methods("PUT")
	apiRouter.HandleFunc("/settings/shells", settingsH.ListShells).Methods("GET")
```

- [ ] **Step 4: Use config shell in terminal_handlers.go**

In `internal/api/terminal_handlers.go`, change `CreateSession` to accept config. Update `TerminalHandlers` struct:

```go
type TerminalHandlers struct {
	Manager *terminal.Manager
	Cfg     *config.Config
}
```

Replace the shell detection in `CreateSession`:

```go
	shell := th.Cfg.Terminal.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/bash"
	}
```

Update `server.go` where `TerminalHandlers` is created:

```go
	th := &api.TerminalHandlers{Manager: termManager, Cfg: cfg}
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: Success.

- [ ] **Step 6: Commit**

```bash
git add internal/api/settings_handlers.go internal/api/terminal_handlers.go internal/server/server.go internal/config/config.go
git commit -m "feat(settings): API endpoints GET/PUT /api/settings, GET /api/settings/shells"
```

---

### Task 3: Frontend — Types + Theme presets

**Files:**
- Modify: `frontend/src/types/index.ts`
- Create: `frontend/src/data/terminal-themes.ts`

- [ ] **Step 1: Add settings types**

Append to `frontend/src/types/index.ts`:

```typescript
// Settings
export interface ServerSettings {
  port: number
  projects_dir: string
  default_project: string
  terminal: {
    max_sessions: number
    shell: string
  }
}

export interface TerminalTheme {
  background: string
  foreground: string
  cursor: string
  selectionBackground: string
  black: string
  red: string
  green: string
  yellow: string
  blue: string
  magenta: string
  cyan: string
  white: string
  brightBlack: string
  brightRed: string
  brightGreen: string
  brightYellow: string
  brightBlue: string
  brightMagenta: string
  brightCyan: string
  brightWhite: string
}

export interface UISettings {
  fontSize: number
  fontFamily: string
  scrollback: number
  cursorBlink: boolean
  themeName: string
}
```

- [ ] **Step 2: Create terminal themes**

Create `frontend/src/data/terminal-themes.ts`:

```typescript
import type { TerminalTheme } from '../types'

export const terminalThemes: Record<string, TerminalTheme> = {
  'github-dark': {
    background: '#0d1117',
    foreground: '#c9d1d9',
    cursor: '#58a6ff',
    selectionBackground: 'rgba(88, 166, 255, 0.3)',
    black: '#484f58', red: '#ff7b72', green: '#3fb950', yellow: '#d29922',
    blue: '#58a6ff', magenta: '#bc8cff', cyan: '#39d353', white: '#b1bac4',
    brightBlack: '#6e7681', brightRed: '#ffa198', brightGreen: '#56d364', brightYellow: '#e3b341',
    brightBlue: '#79c0ff', brightMagenta: '#d2a8ff', brightCyan: '#56d364', brightWhite: '#f0f6fc',
  },
  'dracula': {
    background: '#282a36',
    foreground: '#f8f8f2',
    cursor: '#f8f8f2',
    selectionBackground: 'rgba(68, 71, 90, 0.5)',
    black: '#21222c', red: '#ff5555', green: '#50fa7b', yellow: '#f1fa8c',
    blue: '#bd93f9', magenta: '#ff79c6', cyan: '#8be9fd', white: '#f8f8f2',
    brightBlack: '#6272a4', brightRed: '#ff6e6e', brightGreen: '#69ff94', brightYellow: '#ffffa5',
    brightBlue: '#d6acff', brightMagenta: '#ff92df', brightCyan: '#a4ffff', brightWhite: '#ffffff',
  },
  'one-dark': {
    background: '#282c34',
    foreground: '#abb2bf',
    cursor: '#528bff',
    selectionBackground: 'rgba(82, 139, 255, 0.3)',
    black: '#3f4451', red: '#e06c75', green: '#98c379', yellow: '#e5c07b',
    blue: '#61afef', magenta: '#c678dd', cyan: '#56b6c2', white: '#abb2bf',
    brightBlack: '#4f5666', brightRed: '#be5046', brightGreen: '#98c379', brightYellow: '#d19a66',
    brightBlue: '#61afef', brightMagenta: '#c678dd', brightCyan: '#56b6c2', brightWhite: '#ffffff',
  },
  'nord': {
    background: '#2e3440',
    foreground: '#d8dee9',
    cursor: '#d8dee9',
    selectionBackground: 'rgba(136, 192, 208, 0.3)',
    black: '#3b4252', red: '#bf616a', green: '#a3be8c', yellow: '#ebcb8b',
    blue: '#81a1c1', magenta: '#b48ead', cyan: '#88c0d0', white: '#e5e9f0',
    brightBlack: '#4c566a', brightRed: '#bf616a', brightGreen: '#a3be8c', brightYellow: '#ebcb8b',
    brightBlue: '#81a1c1', brightMagenta: '#b48ead', brightCyan: '#8fbcbb', brightWhite: '#eceff4',
  },
  'monokai': {
    background: '#272822',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    selectionBackground: 'rgba(73, 72, 62, 0.5)',
    black: '#272822', red: '#f92672', green: '#a6e22e', yellow: '#f4bf75',
    blue: '#66d9ef', magenta: '#ae81ff', cyan: '#a1efe4', white: '#f8f8f2',
    brightBlack: '#75715e', brightRed: '#f92672', brightGreen: '#a6e22e', brightYellow: '#f4bf75',
    brightBlue: '#66d9ef', brightMagenta: '#ae81ff', brightCyan: '#a1efe4', brightWhite: '#f9f8f5',
  },
  'solarized-dark': {
    background: '#002b36',
    foreground: '#839496',
    cursor: '#839496',
    selectionBackground: 'rgba(7, 54, 66, 0.5)',
    black: '#073642', red: '#dc322f', green: '#859900', yellow: '#b58900',
    blue: '#268bd2', magenta: '#d33682', cyan: '#2aa198', white: '#eee8d5',
    brightBlack: '#586e75', brightRed: '#cb4b16', brightGreen: '#586e75', brightYellow: '#657b83',
    brightBlue: '#839496', brightMagenta: '#6c71c4', brightCyan: '#93a1a1', brightWhite: '#fdf6e3',
  },
  'tokyo-night': {
    background: '#1a1b26',
    foreground: '#c0caf5',
    cursor: '#c0caf5',
    selectionBackground: 'rgba(51, 59, 91, 0.5)',
    black: '#15161e', red: '#f7768e', green: '#9ece6a', yellow: '#e0af68',
    blue: '#7aa2f7', magenta: '#bb9af7', cyan: '#7dcfff', white: '#a9b1d6',
    brightBlack: '#414868', brightRed: '#f7768e', brightGreen: '#9ece6a', brightYellow: '#e0af68',
    brightBlue: '#7aa2f7', brightMagenta: '#bb9af7', brightCyan: '#7dcfff', brightWhite: '#c0caf5',
  },
}

export const themeNames: Record<string, string> = {
  'github-dark': 'GitHub Dark',
  'dracula': 'Dracula',
  'one-dark': 'One Dark',
  'nord': 'Nord',
  'monokai': 'Monokai',
  'solarized-dark': 'Solarized Dark',
  'tokyo-night': 'Tokyo Night',
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/data/terminal-themes.ts
git commit -m "feat(settings): типы настроек и 7 терминальных тем"
```

---

### Task 4: Frontend — Settings Pinia store

**Files:**
- Create: `frontend/src/stores/settings.ts`

- [ ] **Step 1: Create settings store**

Create `frontend/src/stores/settings.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ServerSettings, UISettings, TerminalTheme } from '../types'
import { terminalThemes } from '../data/terminal-themes'

const UI_SETTINGS_KEY = 'devhub-ui-settings'

const defaultUI: UISettings = {
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
  scrollback: 10000,
  cursorBlink: true,
  themeName: 'github-dark',
}

function loadUI(): UISettings {
  try {
    const raw = localStorage.getItem(UI_SETTINGS_KEY)
    if (raw) return { ...defaultUI, ...JSON.parse(raw) }
  } catch { /* ignore */ }
  return { ...defaultUI }
}

export const useSettingsStore = defineStore('settings', () => {
  // Server settings (from backend)
  const server = ref<ServerSettings>({
    port: 9000,
    projects_dir: '~/project',
    default_project: 'cfa',
    terminal: { max_sessions: 10, shell: '' },
  })

  // UI settings (localStorage)
  const ui = ref<UISettings>(loadUI())

  // Available shells (from backend)
  const shells = ref<string[]>([])

  // Computed: current theme object
  const currentTheme = computed<TerminalTheme>(() => {
    return terminalThemes[ui.value.themeName] || terminalThemes['github-dark']
  })

  // Fetch server settings
  async function fetchSettings() {
    const res = await fetch('/api/settings')
    if (res.ok) {
      server.value = await res.json()
    }
  }

  // Save server settings
  async function saveSettings(updates: Partial<ServerSettings>) {
    const res = await fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates),
    })
    if (res.ok) {
      await fetchSettings()
    }
    return res.ok
  }

  // Fetch available shells
  async function fetchShells() {
    const res = await fetch('/api/settings/shells')
    if (res.ok) {
      shells.value = await res.json()
    }
  }

  // Update UI setting and persist to localStorage
  function updateUI(partial: Partial<UISettings>) {
    ui.value = { ...ui.value, ...partial }
    localStorage.setItem(UI_SETTINGS_KEY, JSON.stringify(ui.value))
  }

  return {
    server,
    ui,
    shells,
    currentTheme,
    fetchSettings,
    saveSettings,
    fetchShells,
    updateUI,
  }
})
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/stores/settings.ts
git commit -m "feat(settings): Pinia store — server + UI + themes"
```

---

### Task 5: Frontend — SettingsView.vue

**Files:**
- Create: `frontend/src/views/SettingsView.vue`

- [ ] **Step 1: Create SettingsView**

Create `frontend/src/views/SettingsView.vue` — full Settings page with General, Terminal, and Theme sections. Uses the mockup layout from `/tmp/settings-mockup.html` adapted to Vue component style.

The component should:
- Call `settingsStore.fetchSettings()` and `settingsStore.fetchShells()` on mount
- Use local reactive copies of settings for the form (so Save/Reset works)
- Track dirty state by comparing local values to store values
- Show/hide the bottom save bar based on dirty state
- On Save: call `settingsStore.saveSettings()` for server fields, `settingsStore.updateUI()` for UI fields
- On Reset: revert local copies to store values
- Theme cards: grid of 7 themes with colored preview, highlight active
- Live preview terminal: a `<div>` that updates colors based on selected theme

Key structure:
```
<template>
  <div class="settings-view">
    <div class="page-header">...</div>

    <!-- General Section -->
    <section class="settings-section">
      <h2>General</h2>
      <!-- projects_dir input, default_project select, port input -->
    </section>

    <!-- Terminal Section -->
    <section class="settings-section">
      <h2>Terminal</h2>
      <!-- shell select, fontSize number, fontFamily select, scrollback number, cursorBlink toggle, maxSessions number -->
    </section>

    <!-- Theme Section -->
    <section class="settings-section">
      <h2>Terminal Theme</h2>
      <!-- theme cards grid + live preview -->
    </section>

    <!-- Save Bar -->
    <div v-if="isDirty" class="save-bar">...</div>
  </div>
</template>
```

CSS should match the mockup: dark theme, setting-row layout (label left, control right), theme-card grid, toggle switch for booleans, sticky save bar at bottom.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/SettingsView.vue
git commit -m "feat(settings): SettingsView — General, Terminal, Theme UI"
```

---

### Task 6: Frontend — Route + Sidebar + WebTerminal integration

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/AppSidebar.vue`
- Modify: `frontend/src/components/WebTerminal.vue`

- [ ] **Step 1: Add settings route**

In `frontend/src/router/index.ts`, add to routes array:

```typescript
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue'),
    },
```

- [ ] **Step 2: Add Settings nav item to sidebar**

In `frontend/src/components/AppSidebar.vue`, add before `</nav>`:

```html
      <router-link to="/settings" class="nav-item" active-class="active">
        <span class="nav-icon">&#9881;</span>
        Settings
      </router-link>
```

- [ ] **Step 3: Wire WebTerminal to settings store**

In `frontend/src/components/WebTerminal.vue`:

1. Import settings store:
```typescript
import { useSettingsStore } from '../stores/settings'
const settingsStore = useSettingsStore()
```

2. Replace hardcoded Terminal options with store values:
```typescript
  term = new Terminal({
    allowProposedApi: true,
    customGlyphs: true,
    cursorBlink: settingsStore.ui.cursorBlink,
    fontFamily: settingsStore.ui.fontFamily,
    fontSize: settingsStore.ui.fontSize,
    lineHeight: 1.0,
    letterSpacing: 0,
    scrollback: settingsStore.ui.scrollback,
    theme: settingsStore.currentTheme,
  })
```

3. Add a watcher to apply live theme/font changes to existing terminals:
```typescript
import { watch } from 'vue'

watch(() => settingsStore.currentTheme, (theme) => {
  if (term) term.options.theme = theme
}, { deep: true })

watch(() => settingsStore.ui.fontSize, (size) => {
  if (term) {
    term.options.fontSize = size
    fitAddon?.fit()
  }
})

watch(() => settingsStore.ui.fontFamily, (font) => {
  if (term) {
    term.options.fontFamily = font
    fitAddon?.fit()
  }
})

watch(() => settingsStore.ui.cursorBlink, (blink) => {
  if (term) term.options.cursorBlink = blink
})
```

- [ ] **Step 4: Build and verify**

Run: `go build ./... && cd frontend && npx vite build`
Expected: Both succeed.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/components/AppSidebar.vue frontend/src/components/WebTerminal.vue
git commit -m "feat(settings): роут, sidebar, WebTerminal читает настройки из store"
```

---

### Task 7: Integration test

- [ ] **Step 1: Manual verification**

Start servers and verify:

1. Open http://localhost:5173/settings — page renders with all sections
2. Change font size → open Console → terminal uses new size
3. Change theme → Console terminal recolors instantly
4. Change shell → Settings Save → new terminal tab uses selected shell
5. Change projects_dir → verify `~/.devhub.yaml` updated
6. Reload page → all settings persisted

- [ ] **Step 2: Final commit**

```bash
git add -A
git commit -m "feat(settings): Settings page — General, Terminal, Theme с live preview"
```
