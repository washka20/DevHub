import { defineStore } from 'pinia'
import { ref, computed, watch, onUnmounted } from 'vue'
import type {
  TerminalSession,
  TerminalTab,
  TerminalPane,
  PanelState,
  PersistedLayout,
} from '../types'

const STORAGE_KEY = 'devhub-terminal-layout'

let counter = 0
function nextId(prefix: string): string {
  return `${prefix}-${++counter}`
}

const defaultPanel: PanelState = {
  mode: 'pinned',
  visible: true,
  height: 30,
  floatingPos: { x: 100, y: 400, w: 500, h: 300 },
}

// ---------------------------------------------------------------------------
// Persistence helpers
// ---------------------------------------------------------------------------

function loadLayout(): PersistedLayout | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as PersistedLayout
    // Basic validation: must have tabs array
    if (!Array.isArray(parsed.tabs)) return null
    return parsed
  } catch {
    return null
  }
}

function saveLayout(tabs: TerminalTab[], activeTabId: string | null, panel: PanelState, sessions: Map<string, TerminalSession>): void {
  const layout: PersistedLayout = {
    tabs: tabs.map((t) => ({
      id: t.id,
      label: t.label,
      panes: t.panes.map((p) => ({
        id: p.id,
        cwd: p.cwd,
        sessionId: p.sessionId,
        label: p.sessionId ? (sessions.get(p.sessionId)?.label || 'shell') : undefined,
      })),
      direction: t.splitDirection,
    })),
    activeTabId,
    panel: { ...panel },
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(layout))
}

function restoreTabs(layout: PersistedLayout, sessions: Map<string, TerminalSession>): TerminalTab[] {
  return layout.tabs.map((t) => ({
    id: t.id,
    label: t.label,
    panes: t.panes.map((p) => {
      // Re-populate sessions Map with saved label for reconnected panes
      if (p.sessionId && p.label) {
        sessions.set(p.sessionId, { id: p.sessionId, label: p.label, cwd: p.cwd })
      }
      return {
        id: p.id,
        sessionId: p.sessionId ?? null,
        cwd: p.cwd,
        status: p.sessionId ? 'reconnecting' as const : 'disconnected' as const,
      }
    }),
    splitDirection: t.direction,
  }))
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useTerminalStore = defineStore('terminal', () => {
  const sessions = ref<Map<string, TerminalSession>>(new Map())

  // Try to restore from localStorage
  const saved = loadLayout()

  const tabs = ref<TerminalTab[]>(saved ? restoreTabs(saved, sessions.value) : [])
  const activeTabId = ref<string | null>(saved?.activeTabId ?? null)
  const panel = ref<PanelState>(saved?.panel ? { ...defaultPanel, ...saved.panel } : { ...defaultPanel })

  // Ensure counter doesn't collide with restored IDs
  if (saved) {
    let maxNum = 0
    for (const t of saved.tabs) {
      for (const p of t.panes) {
        const m = p.id.match(/\d+$/)
        if (m) maxNum = Math.max(maxNum, parseInt(m[0], 10))
      }
      const tm = t.id.match(/\d+$/)
      if (tm) maxNum = Math.max(maxNum, parseInt(tm[0], 10))
    }
    counter = maxNum
  }

  const activeTab = computed(() =>
    tabs.value.find((t) => t.id === activeTabId.value) ?? null,
  )

  // -------------------------------------------------------------------------
  // Autosave: debounced watch on tabs + activeTabId + panel
  // -------------------------------------------------------------------------

  let saveTimer: ReturnType<typeof setTimeout> | null = null

  function scheduleSave() {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => {
      saveLayout(tabs.value, activeTabId.value, panel.value, sessions.value)
    }, 500)
  }

  watch(tabs, scheduleSave, { deep: true })
  watch(activeTabId, scheduleSave)
  watch(panel, scheduleSave, { deep: true })

  onUnmounted(() => {
    if (saveTimer) clearTimeout(saveTimer)
  })

  // -------------------------------------------------------------------------
  // Session management
  // -------------------------------------------------------------------------

  async function createSession(cwd: string, cols = 80, rows = 24): Promise<TerminalSession> {
    const res = await fetch('/api/terminal/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cols, rows, cwd }),
    })
    if (!res.ok) {
      throw new Error(`Failed to create session: ${res.statusText}`)
    }
    const data = await res.json()
    const session: TerminalSession = {
      id: data.session_id,
      label: data.shell.split('/').pop() || 'shell',
      cwd,
    }
    sessions.value.set(session.id, session)
    return session
  }

  async function destroySession(id: string) {
    const res = await fetch(`/api/terminal/sessions/${id}`, { method: 'DELETE' })
    if (!res.ok && res.status !== 404) {
      // best-effort: log but don't throw — callers already wrap in try/catch
      console.warn(`destroySession: server returned ${res.status} for session ${id}`)
    }
    sessions.value.delete(id)
  }

  // -------------------------------------------------------------------------
  // Lazy connect: called when a disconnected pane is activated
  // -------------------------------------------------------------------------

  async function connectPane(paneId: string): Promise<string | null> {
    let targetPane: TerminalPane | undefined
    for (const tab of tabs.value) {
      targetPane = tab.panes.find((p) => p.id === paneId)
      if (targetPane) break
    }
    if (!targetPane) return null
    if (targetPane.status === 'connected' && targetPane.sessionId) return targetPane.sessionId
    if (targetPane.status === 'connecting') return null
    if (targetPane.status === 'reconnecting') return null  // guard: already trying

    // Try reconnecting to saved session
    if (targetPane.sessionId) {
      targetPane.status = 'reconnecting'
      try {
        const res = await fetch(`/api/terminal/sessions/${targetPane.sessionId}`)
        if (res.ok) {
          targetPane.status = 'connected'
          return targetPane.sessionId  // session alive — just open WS
        }
      } catch { /* fall through */ }
      targetPane.sessionId = null  // stale, clear it
    }

    // Create new session
    targetPane.status = 'connecting'
    try {
      const session = await createSession(targetPane.cwd)
      targetPane.sessionId = session.id
      targetPane.status = 'connected'
      return session.id
    } catch {
      targetPane.status = 'disconnected'
      return null
    }
  }

  // -------------------------------------------------------------------------
  // Tab / pane management
  // -------------------------------------------------------------------------

  async function addTab(cwd: string): Promise<TerminalTab> {
    const session = await createSession(cwd)
    const pane: TerminalPane = {
      id: nextId('pane'),
      sessionId: session.id,
      cwd,
      status: 'connected',
    }
    const tab: TerminalTab = {
      id: `tab-${session.id}`,
      label: session.label,
      panes: [pane],
      splitDirection: null,
    }
    tabs.value.push(tab)
    activeTabId.value = tab.id
    return tab
  }

  async function closeTab(tabId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return
    for (const pane of tab.panes) {
      if (pane.sessionId) {
        try { await destroySession(pane.sessionId) } catch { /* best-effort */ }
      }
    }
    tabs.value = tabs.value.filter((t) => t.id !== tabId)
    if (activeTabId.value === tabId) {
      activeTabId.value = tabs.value.length > 0 ? tabs.value[tabs.value.length - 1].id : null
    }
  }

  function setActiveTab(tabId: string) {
    activeTabId.value = tabId
  }

  async function splitPane(tabId: string, direction: 'horizontal' | 'vertical', cwd: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return
    if (tab.panes.length >= 2) return

    const session = await createSession(cwd)
    const pane: TerminalPane = {
      id: nextId('pane'),
      sessionId: session.id,
      cwd,
      status: 'connected',
    }
    tab.panes.push(pane)
    tab.splitDirection = direction
  }

  async function closePane(tabId: string, paneId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return

    const pane = tab.panes.find((p) => p.id === paneId)
    if (!pane) return

    if (pane.sessionId) {
      try { await destroySession(pane.sessionId) } catch { /* best-effort */ }
    }
    tab.panes = tab.panes.filter((p) => p.id !== paneId)

    if (tab.panes.length <= 1) {
      tab.splitDirection = null
    }

    if (tab.panes.length === 0) {
      await closeTab(tabId)
    }
  }

  // -------------------------------------------------------------------------
  // Panel state
  // -------------------------------------------------------------------------

  function updatePanel(partial: Partial<PanelState>) {
    panel.value = { ...panel.value, ...partial }
  }

  // -------------------------------------------------------------------------
  // Cleanup
  // -------------------------------------------------------------------------

  async function cleanOrphans() {
    try {
      const res = await fetch('/api/terminal/sessions')
      if (!res.ok) return
      const liveSessions: Array<{ id: string }> = await res.json()

      const referencedIds = new Set<string>()
      for (const tab of tabs.value) {
        for (const pane of tab.panes) {
          if (pane.sessionId) referencedIds.add(pane.sessionId)
        }
      }

      for (const sess of liveSessions) {
        if (!referencedIds.has(sess.id)) {
          await fetch(`/api/terminal/sessions/${sess.id}`, { method: 'DELETE' }).catch(() => {})
        }
      }
    } catch { /* best-effort */ }
  }

  function handleSessionExit(sessionId: string) {
    sessions.value.delete(sessionId)
    for (const tab of [...tabs.value]) {
      const pane = tab.panes.find((p) => p.sessionId === sessionId)
      if (pane) {
        // Mark as disconnected instead of removing — user can reconnect
        pane.sessionId = null
        pane.status = 'disconnected'
      }
    }
  }

  function clearLayout() {
    localStorage.removeItem(STORAGE_KEY)
  }

  return {
    sessions,
    tabs,
    activeTabId,
    activeTab,
    panel,
    addTab,
    closeTab,
    setActiveTab,
    splitPane,
    closePane,
    connectPane,
    updatePanel,
    cleanOrphans,
    handleSessionExit,
    clearLayout,
  }
})
