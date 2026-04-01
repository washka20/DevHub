# Console Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 6 features to the terminal Console: activity indicator, bell notification, tab context menu, CWD tracking, bottom terminal panel, and Ctrl+` shortcut.

**Architecture:** Incremental additions to existing terminal store + components. Features 1-3 are frontend-only. Feature 4 (CWD) adds one backend endpoint. Features 5-6 (bottom panel + shortcut) modify App.vue layout with a new wrapper component sharing the same Pinia store.

**Tech Stack:** Vue 3 + Pinia, xterm.js, Go (gorilla/mux), Splitpanes library (already installed).

**Spec:** `docs/superpowers/specs/2026-04-01-console-improvements-design.md`

---

## File Summary

| File | Changes |
|------|---------|
| `frontend/src/types/index.ts` | Add `hasActivity`, `hasBell` to TerminalPane |
| `frontend/src/stores/terminal.ts` | Add `renameTab`, `closeOtherTabs`, `closeAllTabs`, `clearPaneAlerts` |
| `frontend/src/components/WebTerminal.vue` | Activity tracking in ws.onmessage, bell handler, OSC 7 parser, CWD badge, CWD poll fallback |
| `frontend/src/components/TerminalTabBar.vue` | Pulsing dot styles, context menu, rename inline edit |
| `frontend/src/components/TabContextMenu.vue` | New — context menu dropdown |
| `frontend/src/components/BottomTerminal.vue` | New — bottom panel wrapper (pinned/floating) |
| `frontend/src/views/ConsoleView.vue` | Request notification permission |
| `frontend/src/App.vue` | Bottom panel integration, Ctrl+` listener |
| `internal/api/terminal_handlers.go` | GetSessionCWD handler |
| `internal/server/server.go` | Register CWD route |

---

## Task 1: Types + Store — Activity, Bell, Tab Management

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/terminal.ts`

- [ ] **Step 1: Add hasActivity and hasBell to TerminalPane type**

In `frontend/src/types/index.ts`, update `TerminalPane`:

```typescript
export interface TerminalPane {
  id: string
  sessionId: string | null
  cwd: string
  status: 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
  hasActivity?: boolean
  hasBell?: boolean
}
```

- [ ] **Step 2: Add tab management methods to store**

In `frontend/src/stores/terminal.ts`, add before the `return` block:

```typescript
  function renameTab(tabId: string, label: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (tab) tab.label = label
  }

  async function closeOtherTabs(keepTabId: string) {
    const toClose = tabs.value.filter((t) => t.id !== keepTabId)
    for (const tab of toClose) {
      await closeTab(tab.id)
    }
  }

  async function closeAllTabs() {
    const allTabs = [...tabs.value]
    for (const tab of allTabs) {
      await closeTab(tab.id)
    }
  }

  function clearPaneAlerts(tabId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return
    for (const pane of tab.panes) {
      pane.hasActivity = false
      pane.hasBell = false
    }
  }
```

Update `setActiveTab` to clear alerts:

```typescript
  function setActiveTab(tabId: string) {
    activeTabId.value = tabId
    clearPaneAlerts(tabId)
  }
```

Add to the `return` block:

```typescript
    renameTab,
    closeOtherTabs,
    closeAllTabs,
    clearPaneAlerts,
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: no errors

- [ ] **Step 4: Commit**

```
feat(terminal): add activity/bell fields, tab management store methods
```

---

## Task 2: Activity Indicator — WebTerminal + TabBar

**Files:**
- Modify: `frontend/src/components/WebTerminal.vue`
- Modify: `frontend/src/components/TerminalTabBar.vue`

- [ ] **Step 1: Track activity in WebTerminal**

In `WebTerminal.vue`, inside `ws.onmessage`, after `term.write(new Uint8Array(event.data))`, add activity tracking:

```typescript
  ws.onmessage = (event: MessageEvent) => {
    if (!term) return
    if (event.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(event.data))
      // Mark pane as having activity if its tab is not active
      if (pane.value && !isActiveTab()) {
        pane.value.hasActivity = true
      }
    } else if (typeof event.data === 'string') {
      // ... existing exit handler
    }
  }
```

Add helper function:

```typescript
function isActiveTab(): boolean {
  for (const tab of terminalStore.tabs) {
    if (tab.panes.some((p) => p.id === props.paneId)) {
      return tab.id === terminalStore.activeTabId
    }
  }
  return false
}
```

- [ ] **Step 2: Add pulsing dot styles in TerminalTabBar**

In `TerminalTabBar.vue`, update the tab-dot to check for activity/bell state. Replace the tab-dot span:

```html
        <span
          class="tab-dot"
          :class="{
            active: terminalStore.activeTabId === tab.id,
            activity: terminalStore.activeTabId !== tab.id && tabHasActivity(tab),
            bell: terminalStore.activeTabId !== tab.id && tabHasBell(tab),
          }"
        ></span>
```

Add helper methods:

```typescript
function tabHasActivity(tab: TerminalTab): boolean {
  return tab.panes.some((p) => p.hasActivity)
}

function tabHasBell(tab: TerminalTab): boolean {
  return tab.panes.some((p) => p.hasBell)
}
```

Add the import:

```typescript
import type { TerminalTab } from '../types'
```

Add CSS:

```css
.tab-dot.activity {
  background: var(--accent-blue);
  animation: pulse 1.5s ease-in-out infinite;
}

.tab-dot.bell {
  background: var(--accent-orange);
  animation: bell-flash 0.5s ease-in-out 3;
}

@keyframes pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(88, 166, 255, 0.4); }
  50% { opacity: 0.6; box-shadow: 0 0 0 4px rgba(88, 166, 255, 0); }
}

@keyframes bell-flash {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
```

- [ ] **Step 3: Verify visually**

Open Console, create 2 tabs. In tab 2 run `while true; do echo test; sleep 1; done`. Switch to tab 1. Tab 2's dot should pulse blue.

- [ ] **Step 4: Commit**

```
feat(terminal): activity indicator — blue pulsing dot on tabs with background output
```

---

## Task 3: Bell Notification

**Files:**
- Modify: `frontend/src/components/WebTerminal.vue`
- Modify: `frontend/src/views/ConsoleView.vue`

- [ ] **Step 1: Request notification permission in ConsoleView**

In `ConsoleView.vue`, inside `onActivated`, add after the existing code:

```typescript
  // Request notification permission for terminal bell
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
```

- [ ] **Step 2: Add bell handler in WebTerminal**

In `WebTerminal.vue`, inside `initTerminal()`, after the `term.onResize` handler, add:

```typescript
  term.onBell(() => {
    if (!isActiveTab() && pane.value) {
      pane.value.hasBell = true
      // Clear bell indicator after 3 seconds
      setTimeout(() => {
        if (pane.value) pane.value.hasBell = false
      }, 3000)
      // Browser notification
      if ('Notification' in window && Notification.permission === 'granted') {
        const tabLabel = getTabLabel()
        new Notification('Terminal bell', { body: `Tab: ${tabLabel}`, icon: '/favicon.ico' })
      }
    }
  })
```

Add helper:

```typescript
function getTabLabel(): string {
  for (const tab of terminalStore.tabs) {
    if (tab.panes.some((p) => p.id === props.paneId)) {
      return tab.label
    }
  }
  return 'terminal'
}
```

- [ ] **Step 3: Test bell**

Open Console with 2 tabs. In tab 2 run: `echo -e '\a'`. Switch to tab 1, run in tab 2 terminal: tab 2's dot should flash orange + browser notification appears.

- [ ] **Step 4: Commit**

```
feat(terminal): bell notification — orange flash + browser Notification API
```

---

## Task 4: Tab Context Menu

**Files:**
- Create: `frontend/src/components/TabContextMenu.vue`
- Modify: `frontend/src/components/TerminalTabBar.vue`

- [ ] **Step 1: Create TabContextMenu component**

Create `frontend/src/components/TabContextMenu.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  x: number
  y: number
  tabId: string
  canSplit: boolean
}>()

const emit = defineEmits<{
  close: []
  rename: [tabId: string]
  splitH: [tabId: string]
  splitV: [tabId: string]
  closeTab: [tabId: string]
  closeOthers: [tabId: string]
  closeAll: []
}>()

const menuEl = ref<HTMLDivElement>()

function handleClickOutside(e: MouseEvent) {
  if (menuEl.value && !menuEl.value.contains(e.target as Node)) {
    emit('close')
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => {
  document.addEventListener('mousedown', handleClickOutside)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div ref="menuEl" class="context-menu" :style="{ left: x + 'px', top: y + 'px' }">
    <div class="menu-item" @click="emit('rename', tabId); emit('close')">
      <span>Rename</span>
      <span class="hint">F2</span>
    </div>
    <div class="menu-sep"></div>
    <div class="menu-item" :class="{ disabled: !canSplit }" @click="canSplit && (emit('splitH', tabId), emit('close'))">
      Split Horizontal
    </div>
    <div class="menu-item" :class="{ disabled: !canSplit }" @click="canSplit && (emit('splitV', tabId), emit('close'))">
      Split Vertical
    </div>
    <div class="menu-sep"></div>
    <div class="menu-item" @click="emit('closeTab', tabId); emit('close')">Close</div>
    <div class="menu-item" @click="emit('closeOthers', tabId); emit('close')">Close Others</div>
    <div class="menu-item danger" @click="emit('closeAll'); emit('close')">Close All</div>
  </div>
</template>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 1000;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 4px 0;
  min-width: 180px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  font-family: var(--font-ui);
  font-size: 13px;
}

.menu-item {
  padding: 6px 12px;
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.menu-item:hover {
  background: var(--accent-blue);
  color: #fff;
}

.menu-item.danger {
  color: var(--accent-red);
}

.menu-item.danger:hover {
  background: var(--accent-red);
  color: #fff;
}

.menu-item.disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.menu-item.disabled:hover {
  background: none;
  color: var(--text-primary);
}

.hint {
  font-size: 11px;
  color: var(--text-secondary);
}

.menu-item:hover .hint {
  color: rgba(255, 255, 255, 0.7);
}

.menu-sep {
  height: 1px;
  background: var(--border);
  margin: 4px 0;
}
</style>
```

- [ ] **Step 2: Integrate context menu into TerminalTabBar**

In `TerminalTabBar.vue`:

Add imports and state:

```typescript
import { ref } from 'vue'
import TabContextMenu from './TabContextMenu.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'
import type { TerminalTab } from '../types'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

const contextMenu = ref<{ x: number; y: number; tabId: string } | null>(null)
const renamingTabId = ref<string | null>(null)
const renameValue = ref('')
```

Add context menu handlers:

```typescript
function handleContextMenu(e: MouseEvent, tabId: string) {
  e.preventDefault()
  contextMenu.value = { x: e.clientX, y: e.clientY, tabId }
}

function startRename(tabId: string) {
  const tab = terminalStore.tabs.find((t) => t.id === tabId)
  if (!tab) return
  renamingTabId.value = tabId
  renameValue.value = tab.label
}

function finishRename() {
  if (renamingTabId.value && renameValue.value.trim()) {
    terminalStore.renameTab(renamingTabId.value, renameValue.value.trim())
  }
  renamingTabId.value = null
}

function cancelRename() {
  renamingTabId.value = null
}

function handleSplitFromMenu(tabId: string, direction: 'horizontal' | 'vertical') {
  terminalStore.setActiveTab(tabId)
  emit('split', direction)
}
```

Update the tab template to add `@contextmenu` and inline rename:

```html
      <div
        v-for="tab in terminalStore.tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: terminalStore.activeTabId === tab.id }"
        @click="terminalStore.setActiveTab(tab.id)"
        @contextmenu="handleContextMenu($event, tab.id)"
      >
        <span
          class="tab-dot"
          :class="{
            active: terminalStore.activeTabId === tab.id,
            activity: terminalStore.activeTabId !== tab.id && tabHasActivity(tab),
            bell: terminalStore.activeTabId !== tab.id && tabHasBell(tab),
          }"
        ></span>
        <!-- Inline rename input -->
        <input
          v-if="renamingTabId === tab.id"
          v-model="renameValue"
          class="tab-rename-input"
          @keydown.enter="finishRename"
          @keydown.escape="cancelRename"
          @blur="finishRename"
          @click.stop
          ref="renameInput"
          autofocus
        />
        <span v-else class="tab-label">{{ tab.label }}</span>
        <button
          class="tab-close"
          @click.stop="terminalStore.closeTab(tab.id)"
          title="Close tab"
          aria-label="Close tab"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/>
          </svg>
        </button>
      </div>
```

Add the context menu component at the end of template (inside `.tab-bar`):

```html
    <Teleport to="body">
      <TabContextMenu
        v-if="contextMenu"
        :x="contextMenu.x"
        :y="contextMenu.y"
        :tab-id="contextMenu.tabId"
        :can-split="(terminalStore.tabs.find(t => t.id === contextMenu?.tabId)?.panes.length ?? 0) < 2"
        @close="contextMenu = null"
        @rename="startRename"
        @split-h="(id) => handleSplitFromMenu(id, 'horizontal')"
        @split-v="(id) => handleSplitFromMenu(id, 'vertical')"
        @close-tab="terminalStore.closeTab"
        @close-others="terminalStore.closeOtherTabs"
        @close-all="terminalStore.closeAllTabs"
      />
    </Teleport>
```

Add CSS for rename input:

```css
.tab-rename-input {
  background: var(--bg-primary);
  border: 1px solid var(--accent-blue);
  border-radius: 3px;
  color: var(--text-primary);
  font-size: 12px;
  font-family: var(--font-mono);
  padding: 1px 4px;
  width: 80px;
  outline: none;
}
```

- [ ] **Step 3: Verify context menu works**

Right-click a tab → menu appears. Rename → inline input. Close Others → only clicked tab remains.

- [ ] **Step 4: Commit**

```
feat(terminal): tab context menu — rename, split, close others/all
```

---

## Task 5: CWD Tracking — Backend Endpoint

**Files:**
- Modify: `internal/api/terminal_handlers.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add GetSessionCWD handler**

In `internal/api/terminal_handlers.go`, add:

```go
// GetSessionCWD handles GET /api/terminal/sessions/{id}/cwd.
func (th *TerminalHandlers) GetSessionCWD(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if len(id) > 64 {
		jsonError(w, "invalid session id", http.StatusBadRequest)
		return
	}
	sess, ok := th.Manager.Get(id)
	if !ok {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}
	cwd := sess.CWD // fallback
	if sess.Cmd.Process != nil {
		if link, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", sess.Cmd.Process.Pid)); err == nil {
			cwd = link
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"cwd": cwd})
}
```

Add `"fmt"` to imports if not already present.

- [ ] **Step 2: Register route**

In `internal/server/server.go`, add after the existing terminal routes (after the `th.DestroySession` line):

```go
	apiRouter.HandleFunc("/terminal/sessions/{id}/cwd", th.GetSessionCWD).Methods("GET")
```

- [ ] **Step 3: Build and test**

Run: `go build ./...`

Test: `curl http://localhost:9000/api/terminal/sessions/<ID>/cwd`
Expected: `{"cwd":"/home/washka/project/devhub"}`

- [ ] **Step 4: Commit**

```
feat(terminal): GET /api/terminal/sessions/{id}/cwd — reads /proc/{pid}/cwd
```

---

## Task 6: CWD Tracking — Frontend (OSC 7 + Fallback + Badge)

**Files:**
- Modify: `frontend/src/components/WebTerminal.vue`

- [ ] **Step 1: Add OSC 7 parser and CWD badge**

In `WebTerminal.vue`, inside `initTerminal()`, after loading addons and before `term.open(terminalEl.value)`, register the OSC handler:

```typescript
  // OSC 7: shell reports current working directory
  // Format: \e]7;file://hostname/path\a
  term.parser.registerOscHandler(7, (data) => {
    try {
      const url = new URL(data)
      if (url.protocol === 'file:' && url.pathname) {
        const newCwd = decodeURIComponent(url.pathname)
        if (pane.value && newCwd !== pane.value.cwd) {
          pane.value.cwd = newCwd
          oscCwdReceived = true
        }
      }
    } catch { /* ignore malformed OSC 7 */ }
    return false // don't prevent default handling
  })
```

Add state variables at the top of `<script setup>`:

```typescript
let oscCwdReceived = false
let cwdPollTimer: ReturnType<typeof setInterval> | null = null
```

Add CWD poll fallback — start it in `connectWs` `ws.onopen`:

```typescript
  ws.onopen = () => {
    reconnectAttempts = 0
    if (term && fitAddon) {
      fitAddon.fit()
      sendResize(term.cols, term.rows)
    }
    // Start CWD polling fallback after 10s if no OSC 7 received
    if (cwdPollTimer) clearInterval(cwdPollTimer)
    setTimeout(() => {
      if (!oscCwdReceived && !disposed && pane.value?.sessionId) {
        cwdPollTimer = setInterval(() => pollCwd(), 5000)
      }
    }, 10000)
  }
```

Add poll function:

```typescript
async function pollCwd() {
  if (!pane.value?.sessionId || disposed) {
    if (cwdPollTimer) clearInterval(cwdPollTimer)
    return
  }
  try {
    const res = await fetch(`/api/terminal/sessions/${pane.value.sessionId}/cwd`)
    if (res.ok) {
      const data = await res.json()
      if (pane.value && data.cwd && data.cwd !== pane.value.cwd) {
        pane.value.cwd = data.cwd
      }
    }
  } catch { /* ignore */ }
}
```

Clean up poll timer in `onBeforeUnmount`:

```typescript
  if (cwdPollTimer) clearInterval(cwdPollTimer)
```

- [ ] **Step 2: Add CWD badge to template**

In the `v-else` terminal div, wrap it with a relative container and add the badge:

```html
  <!-- Connected terminal -->
  <div v-else class="terminal-wrapper">
    <div ref="terminalEl" class="web-terminal"></div>
    <div v-if="pane?.cwd" class="cwd-badge" :title="pane.cwd">
      {{ shortCwd(pane.cwd) }}
    </div>
  </div>
```

Add helper:

```typescript
function shortCwd(cwd: string): string {
  const home = '/home/'
  const idx = cwd.indexOf('/', home.length)
  if (idx > 0) return '~' + cwd.slice(idx)
  return cwd
}
```

Add CSS:

```css
.terminal-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
}

.cwd-badge {
  position: absolute;
  top: 4px;
  right: 12px;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  opacity: 0.5;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid var(--border);
  pointer-events: none;
  z-index: 1;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

- [ ] **Step 3: Rebuild backend and test**

Run: `go build ./... && cd frontend && npx vue-tsc --noEmit`

Test: Open console, `cd /tmp` → badge should update to `/tmp` (via OSC 7) or after 10s fallback poll.

- [ ] **Step 4: Commit**

```
feat(terminal): CWD tracking — OSC 7 parser + /proc fallback + floating badge
```

---

## Task 7: Bottom Terminal Panel Component

**Files:**
- Create: `frontend/src/components/BottomTerminal.vue`
- Modify: `frontend/src/stores/terminal.ts`

- [ ] **Step 1: Add panel toggle method to store**

In `frontend/src/stores/terminal.ts`, add:

```typescript
  function togglePanel() {
    panel.value.visible = !panel.value.visible
  }

  function setPanelMode(mode: 'pinned' | 'floating') {
    panel.value.mode = mode
  }
```

Add to return:

```typescript
    togglePanel,
    setPanelMode,
```

- [ ] **Step 2: Create BottomTerminal.vue**

Create `frontend/src/components/BottomTerminal.vue`:

```vue
<script setup lang="ts">
import { computed } from 'vue'
import WebTerminal from './WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

const activeTab = computed(() => terminalStore.activeTab)

async function handleNewTab() {
  const cwd = projectsStore.currentProject?.path || ''
  try {
    await terminalStore.addTab(cwd)
  } catch { /* max sessions */ }
}

function handleClose() {
  terminalStore.updatePanel({ visible: false })
}

function handleMaximize() {
  // Navigate to /console
  window.location.hash = '' // will be handled by router
  import('../router').then(m => m.default.push('/console'))
}

function toggleMode() {
  terminalStore.setPanelMode(
    terminalStore.panel.mode === 'pinned' ? 'floating' : 'pinned'
  )
}
</script>

<template>
  <div class="bottom-terminal" :class="terminalStore.panel.mode">
    <!-- Header -->
    <div class="bt-header">
      <div class="bt-tabs">
        <div
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          class="bt-tab"
          :class="{ active: terminalStore.activeTabId === tab.id }"
          @click="terminalStore.setActiveTab(tab.id)"
        >
          <span class="bt-dot" :class="{ active: terminalStore.activeTabId === tab.id }"></span>
          <span class="bt-tab-label">{{ tab.label }}</span>
        </div>
        <button class="bt-tab-add" @click="handleNewTab" title="New terminal">+</button>
      </div>
      <div class="bt-actions">
        <button class="bt-btn" @click="handleMaximize" title="Maximize">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M1.75 10a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-3.25a.75.75 0 0 1-.75-.75v-3.25a.75.75 0 0 1 .75-.75Zm12.5 0a.75.75 0 0 1 .75.75v3.25a.75.75 0 0 1-.75.75H11a.75.75 0 0 1 0-1.5h2.5v-2.5a.75.75 0 0 1 .75-.75ZM2.5 2.5v2.5H5a.75.75 0 0 1 0 1.5H1.75A.75.75 0 0 1 1 5.75V2.5a.75.75 0 0 1 1.5 0Zm10 0a.75.75 0 0 1 1.5 0v3.25a.75.75 0 0 1-.75.75H11a.75.75 0 0 1 0-1.5h2.5V2.5h-1Z"/>
          </svg>
        </button>
        <button class="bt-btn" @click="toggleMode" :title="terminalStore.panel.mode === 'pinned' ? 'Float' : 'Pin'">
          <svg v-if="terminalStore.panel.mode === 'pinned'" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M2.75 2h10.5a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H2.75a.75.75 0 0 1-.75-.75V2.75A.75.75 0 0 1 2.75 2Zm.75 1.5v9h9v-9h-9Z"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M4.75 0a.75.75 0 0 1 .75.75V2h5V.75a.75.75 0 0 1 1.5 0V2h2.25a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H.75a.75.75 0 0 1-.75-.75V2.75A.75.75 0 0 1 .75 2H4V.75A.75.75 0 0 1 4.75 0ZM1.5 3.5v9h13v-9h-13Z"/>
          </svg>
        </button>
        <button class="bt-btn" @click="handleClose" title="Hide (Ctrl+`)">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Terminal area — render all tabs, show only active -->
    <div class="bt-body">
      <div
        v-for="tab in terminalStore.tabs"
        :key="tab.id"
        v-show="tab.id === terminalStore.activeTabId"
        class="bt-terminal-area"
      >
        <WebTerminal
          v-for="pane in tab.panes"
          :key="pane.id"
          :pane-id="pane.id"
        />
      </div>
      <div v-if="terminalStore.tabs.length === 0" class="bt-empty" @click="handleNewTab">
        Click to open terminal
      </div>
    </div>
  </div>
</template>

<style scoped>
.bottom-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary);
  border-top: 1px solid var(--border);
}

.bt-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  padding: 0 8px;
  height: 28px;
  flex-shrink: 0;
}

.bt-tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
}

.bt-tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
  border-radius: 3px;
}

.bt-tab.active {
  color: var(--text-primary);
  border-bottom: 2px solid var(--accent-blue);
}

.bt-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--text-secondary);
  flex-shrink: 0;
}

.bt-dot.active {
  background: var(--accent-green);
}

.bt-tab-add {
  font-size: 12px;
  color: var(--text-secondary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0 4px;
}

.bt-tab-add:hover {
  color: var(--text-primary);
}

.bt-tab-label {
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bt-actions {
  display: flex;
  gap: 2px;
  align-items: center;
}

.bt-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  padding: 0;
}

.bt-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.bt-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.bt-terminal-area {
  height: 100%;
}

.bt-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.bt-empty:hover {
  color: var(--text-primary);
}

/* Floating mode */
.bottom-terminal.floating {
  position: fixed;
  z-index: 1000;
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  resize: both;
  overflow: auto;
  min-width: 400px;
  min-height: 200px;
}

.bottom-terminal.floating .bt-header {
  cursor: move;
  border-radius: 8px 8px 0 0;
}
</style>
```

- [ ] **Step 3: TypeScript check**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: no errors

- [ ] **Step 4: Commit**

```
feat(terminal): BottomTerminal component — compact panel with pinned/floating modes
```

---

## Task 8: App.vue Integration + Ctrl+` Shortcut

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/views/ConsoleView.vue`

- [ ] **Step 1: Read current App.vue**

Read `frontend/src/App.vue` for exact current structure.

- [ ] **Step 2: Add bottom panel and keyboard shortcut to App.vue**

In `App.vue`, add imports:

```typescript
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import BottomTerminal from './components/BottomTerminal.vue'
import { useTerminalStore } from './stores/terminal'
import { useRoute } from 'vue-router'
```

Add setup logic:

```typescript
const terminalStore = useTerminalStore()
const route = useRoute()

const showBottomPanel = computed(() =>
  terminalStore.panel.visible && route.path !== '/console'
)

// Ctrl+` keyboard shortcut
onMounted(() => {
  document.addEventListener('keydown', handleGlobalKeydown, true) // capture phase
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleGlobalKeydown, true)
})

function handleGlobalKeydown(e: KeyboardEvent) {
  if (e.ctrlKey && e.key === '`') {
    e.preventDefault()
    e.stopPropagation()
    if (route.path !== '/console') {
      terminalStore.togglePanel()
    }
  }
}
```

Wrap the main content area with Splitpanes. Replace the existing `<main>` with:

```html
    <Splitpanes horizontal class="app-splitpanes">
      <Pane :size="showBottomPanel ? 100 - terminalStore.panel.height : 100">
        <main class="main-content">
          <router-view v-slot="{ Component }">
            <keep-alive include="ConsoleView">
              <component :is="Component" />
            </keep-alive>
          </router-view>
        </main>
      </Pane>
      <Pane v-if="showBottomPanel" :size="terminalStore.panel.height" :min-size="10" :max-size="80">
        <BottomTerminal />
      </Pane>
    </Splitpanes>
```

Add Splitpanes styles to remove default theme artifacts:

```css
.app-splitpanes {
  flex: 1;
  min-height: 0;
}

:deep(.app-splitpanes > .splitpanes__splitter) {
  background: var(--border);
  min-height: 4px;
}

:deep(.app-splitpanes > .splitpanes__splitter:hover) {
  background: var(--accent-blue);
}

:deep(.app-splitpanes > .splitpanes__splitter::before),
:deep(.app-splitpanes > .splitpanes__splitter::after) {
  display: none;
}
```

- [ ] **Step 3: Auto-hide panel on /console**

In `ConsoleView.vue`, add:

```typescript
import { onActivated, onDeactivated } from 'vue'

let panelWasVisible = false

onActivated(() => {
  // Hide bottom panel on /console — it's redundant
  panelWasVisible = terminalStore.panel.visible
  if (terminalStore.panel.visible) {
    terminalStore.updatePanel({ visible: false })
  }
  // ... existing init code
})

onDeactivated(() => {
  // Restore panel when leaving /console
  if (panelWasVisible) {
    terminalStore.updatePanel({ visible: true })
  }
})
```

- [ ] **Step 4: Save panel height on resize**

In `App.vue`, listen for Splitpanes resize:

```html
    <Splitpanes horizontal class="app-splitpanes" @resized="handlePanelResize">
```

```typescript
function handlePanelResize(panes: Array<{ size: number }>) {
  if (panes.length === 2) {
    terminalStore.updatePanel({ height: panes[1].size })
  }
}
```

- [ ] **Step 5: TypeScript check and verify**

Run: `cd frontend && npx vue-tsc --noEmit`

Test: On any page (Dashboard, Git), press Ctrl+` → bottom panel appears. Press again → hides. Navigate to /console → panel auto-hides. Leave /console → restores.

- [ ] **Step 6: Commit**

```
feat(terminal): bottom panel in App.vue + Ctrl+` toggle, auto-hide on /console
```

---

## Verification Checklist

1. **Activity indicator**: 2 tabs, background tab running output → blue pulsing dot
2. **Bell**: background tab `echo -e '\a'` → orange flash + browser notification
3. **Context menu**: right-click tab → Rename, Split, Close Others/All
4. **CWD badge**: `cd /tmp` → badge updates to `/tmp`
5. **Bottom panel**: Ctrl+` on Git page → panel appears, drag splitter resizes, Ctrl+` hides
6. **Auto-hide**: Navigate to /console → panel hides, leave → restores
7. **Floating mode**: Click float button → panel becomes draggable overlay
8. **TypeScript**: `cd frontend && npx vue-tsc --noEmit` — no errors
9. **Go build**: `go build ./...` — no errors
