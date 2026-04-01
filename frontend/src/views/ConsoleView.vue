<script setup lang="ts">
defineOptions({ name: 'ConsoleView' })

import { onActivated } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import TerminalTabBar from '../components/TerminalTabBar.vue'
import WebTerminal from '../components/WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

let initialized = false

onActivated(async () => {
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

function handleSplit(direction: 'horizontal' | 'vertical') {
  if (!terminalStore.activeTab) return
  const cwd = projectsStore.currentProject?.path || ''
  terminalStore.splitPane(terminalStore.activeTab.id, direction, cwd)
}

function handlePaneClose(paneId: string) {
  if (!terminalStore.activeTab) return
  terminalStore.closePane(terminalStore.activeTab.id, paneId)
}
</script>

<template>
  <div class="console-view">
    <TerminalTabBar @split="handleSplit" />

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
              <span class="pane-close" @click="handlePaneClose(pane.id)">&#10005;</span>
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
</template>

<style scoped>
.console-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px -32px;
  background: var(--bg-primary);
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

.pane-close {
  cursor: pointer;
  color: var(--text-secondary);
  opacity: 0.5;
  padding: 0 4px;
}

.pane-close:hover {
  opacity: 1;
  color: var(--accent-red, #f85149);
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
