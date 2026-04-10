<script setup lang="ts">
defineOptions({ name: 'ConsoleView' })

import { onActivated, onDeactivated, nextTick } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import TerminalTabBar from '../components/TerminalTabBar.vue'
import WebTerminal from '../components/WebTerminal.vue'
import SessionsPanel from '../components/SessionsPanel.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

let initialized = false
let panelWasVisible = false

onActivated(async () => {
  // Hide bottom panel on /console — it's redundant
  panelWasVisible = terminalStore.panel.visible
  if (terminalStore.panel.visible) {
    terminalStore.updatePanel({ visible: false })
  }

  await nextTick()
  terminalStore.triggerReconnect()

  // Request notification permission for terminal bell
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }

  if (!initialized) {
    initialized = true
    // Clean orphan backend sessions from previous server lifecycle.
    // This does NOT clear the persisted layout — just kills stale PTYs.
    await terminalStore.cleanOrphans()
  }

  // If there are restored tabs from localStorage, do nothing — they show
  // the "Press Enter to connect" placeholder and connect lazily.
  // Only create a fresh tab if the store is truly empty (first ever visit).
  if (terminalStore.tabs.length === 0) {
    const cwd = projectsStore.currentProject?.path || ''
    try {
      await terminalStore.addTab(cwd)
    } catch {
      // session creation may fail on first load before backend is ready
    }
  }
})

onDeactivated(() => {
  if (panelWasVisible) {
    terminalStore.updatePanel({ visible: true })
  }
})

function handleSplit(direction: 'horizontal' | 'vertical') {
  if (!terminalStore.activeTab) return
  const cwd = projectsStore.currentProject?.path || ''
  terminalStore.splitPane(terminalStore.activeTab.id, direction, cwd)
}

function handlePaneClose(paneId: string) {
  if (!terminalStore.activeTab) return
  terminalStore.closePane(terminalStore.activeTab.id, paneId)
}

function handlePaneDetach(paneId: string) {
  if (!terminalStore.activeTab) return
  terminalStore.detachToTab(terminalStore.activeTab.id, paneId)
}
</script>

<template>
  <div class="console-view">
    <TerminalTabBar @split="handleSplit" />

    <div class="console-body">
      <div class="terminals">
        <!-- Render ALL tabs, show only active via v-show (keeps xterm alive) -->
        <div
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          v-show="tab.id === terminalStore.activeTabId"
          class="terminal-area"
        >
          <Splitpanes
            :horizontal="tab.splitDirection === 'vertical'"
            class="default-theme"
          >
            <Pane v-for="pane in tab.panes" :key="pane.id">
              <div class="pane-container">
                <div v-show="tab.panes.length > 1" class="pane-header">
                  <span class="pane-title">
                    {{ pane.sessionId ? (terminalStore.sessions.get(pane.sessionId)?.label || 'shell') : 'disconnected' }}
                  </span>
                  <button class="pane-detach" @click="handlePaneDetach(pane.id)" title="Detach to tab">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                      <path d="M3.5 3.5v9h9v-4.5h1.5v4.5a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 2 12.5v-9A1.5 1.5 0 0 1 3.5 2H8v1.5H3.5ZM10 2h4v4h-1.5V3.5H10V2Z"/>
                    </svg>
                  </button>
                  <button class="pane-close" @click="handlePaneClose(pane.id)" title="Close pane">
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                      <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/>
                    </svg>
                  </button>
                </div>
                <div class="pane-body">
                  <WebTerminal :pane-id="pane.id" />
                </div>
              </div>
            </Pane>
          </Splitpanes>
        </div>

        <div v-if="terminalStore.tabs.length === 0" class="empty-state">
          <p>No terminal sessions. Click + to open one.</p>
        </div>
      </div>

      <SessionsPanel v-if="terminalStore.sessionsPanelOpen" />
    </div>
  </div>
</template>

<style scoped>
.console-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px -32px;
  background: var(--bg-primary);
}

.console-body {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.terminals {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.terminal-area {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.pane-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.pane-title {
  font-family: var(--font-mono);
}

.pane-detach {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  cursor: pointer;
  color: var(--text-secondary);
  opacity: 0.5;
  padding: 0;
  border-radius: 4px;
}

.pane-detach:hover {
  opacity: 1;
  color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.15);
}

.pane-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  cursor: pointer;
  color: var(--text-secondary);
  opacity: 0.5;
  padding: 0;
  border-radius: 4px;
}

.pane-close:hover {
  opacity: 1;
  color: var(--accent-red);
  background: rgba(248, 81, 73, 0.15);
}

.pane-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 14px;
}

:deep(.splitpanes.default-theme .splitpanes__splitter) {
  background: var(--border);
  min-width: 4px;
  min-height: 4px;
}

:deep(.splitpanes.default-theme .splitpanes__splitter:hover) {
  background: var(--accent-blue);
}

:deep(.splitpanes.default-theme .splitpanes__splitter::before),
:deep(.splitpanes.default-theme .splitpanes__splitter::after) {
  display: none;
}

:deep(.splitpanes.default-theme .splitpanes__pane:only-child + .splitpanes__splitter) {
  display: none;
}
</style>
