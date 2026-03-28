import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { TerminalSession, TerminalTab, TerminalPane } from '../types'

let counter = 0
function nextId(): string {
  return `pane-${++counter}`
}

export const useTerminalStore = defineStore('terminal', () => {
  const sessions = ref<Map<string, TerminalSession>>(new Map())
  const tabs = ref<TerminalTab[]>([])
  const activeTabId = ref<string | null>(null)

  const activeTab = computed(() =>
    tabs.value.find((t) => t.id === activeTabId.value) ?? null,
  )

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
    await fetch(`/api/terminal/sessions/${id}`, { method: 'DELETE' })
    sessions.value.delete(id)
  }

  async function addTab(cwd: string): Promise<TerminalTab> {
    const session = await createSession(cwd)
    const pane: TerminalPane = { id: nextId(), sessionId: session.id }
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
      try { await destroySession(pane.sessionId) } catch { /* best-effort, WS cleanup is safety net */ }
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
    const pane: TerminalPane = { id: nextId(), sessionId: session.id }
    tab.panes.push(pane)
    tab.splitDirection = direction
  }

  async function closePane(tabId: string, paneId: string) {
    const tab = tabs.value.find((t) => t.id === tabId)
    if (!tab) return

    const pane = tab.panes.find((p) => p.id === paneId)
    if (!pane) return

    try { await destroySession(pane.sessionId) } catch { /* best-effort */ }
    tab.panes = tab.panes.filter((p) => p.id !== paneId)

    if (tab.panes.length <= 1) {
      tab.splitDirection = null
    }

    if (tab.panes.length === 0) {
      await closeTab(tabId)
    }
  }

  // Clean up orphan backend sessions (from page refresh / stale state)
  async function cleanOrphans() {
    try {
      await fetch('/api/terminal/sessions', { method: 'DELETE' })
    } catch { /* best-effort */ }
  }

  // Called when shell exits (PTY sends exit event)
  function handleSessionExit(sessionId: string) {
    sessions.value.delete(sessionId)
    // Find and close any tab/pane using this session
    for (const tab of [...tabs.value]) {
      const pane = tab.panes.find((p) => p.sessionId === sessionId)
      if (pane) {
        tab.panes = tab.panes.filter((p) => p.id !== pane.id)
        if (tab.panes.length <= 1) tab.splitDirection = null
        if (tab.panes.length === 0) {
          tabs.value = tabs.value.filter((t) => t.id !== tab.id)
          if (activeTabId.value === tab.id) {
            activeTabId.value = tabs.value.length > 0 ? tabs.value[tabs.value.length - 1].id : null
          }
        }
      }
    }
  }

  return {
    sessions,
    tabs,
    activeTabId,
    activeTab,
    addTab,
    closeTab,
    setActiveTab,
    splitPane,
    closePane,
    cleanOrphans,
    handleSessionExit,
  }
})
