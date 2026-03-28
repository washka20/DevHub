<script setup lang="ts">
defineOptions({ name: 'ConsoleView' })

import { onMounted } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import TerminalTabBar from '../components/TerminalTabBar.vue'
import WebTerminal from '../components/WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

onMounted(async () => {
  if (terminalStore.tabs.length === 0) {
    const cwd = projectsStore.currentProject?.path || ''
    await terminalStore.addTab(cwd)
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

    <div class="terminal-area" v-if="terminalStore.activeTab">
      <template v-if="terminalStore.activeTab.panes.length === 1">
        <WebTerminal :session-id="terminalStore.activeTab.panes[0].sessionId" />
      </template>

      <template v-else-if="terminalStore.activeTab.panes.length > 1">
        <Splitpanes
          :horizontal="terminalStore.activeTab.splitDirection === 'vertical'"
          class="default-theme"
        >
          <Pane
            v-for="pane in terminalStore.activeTab.panes"
            :key="pane.id"
          >
            <div class="pane-container">
              <div class="pane-header">
                <span class="pane-title">
                  {{ terminalStore.sessions.get(pane.sessionId)?.label || 'shell' }}
                </span>
                <span class="pane-close" @click="handlePaneClose(pane.id)">&#10005;</span>
              </div>
              <div class="pane-body">
                <WebTerminal :session-id="pane.sessionId" />
              </div>
            </div>
          </Pane>
        </Splitpanes>
      </template>
    </div>

    <div v-else class="empty-state">
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
  background: #0d1117;
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
  background: #161b22;
  border-bottom: 1px solid #21262d;
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
  background: #30363d;
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
</style>
