<script setup lang="ts">
import { useRouter } from 'vue-router'
import WebTerminal from './WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()
const router = useRouter()

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
  router.push('/console')
}

function toggleMode() {
  terminalStore.setPanelMode(
    terminalStore.panel.mode === 'pinned' ? 'floating' : 'pinned'
  )
}
</script>

<template>
  <div class="bottom-terminal">
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
            <path d="M1.75 10a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-3.25a.75.75 0 0 1-.75-.75v-3.25a.75.75 0 0 1 .75-.75Zm12.5 0a.75.75 0 0 1 .75.75v3.25a.75.75 0 0 1-.75.75H11a.75.75 0 0 1 0-1.5h2.5v-2.5a.75.75 0 0 1 .75-.75ZM2.5 2.5v2.5H5a.75.75 0 0 1 0 1.5H1.75A.75.75 0 0 1 1 5.75v-3.5a.75.75 0 0 1 1.5 0Zm11.5 0a.75.75 0 0 0-1.5 0v2.5H10a.75.75 0 0 0 0 1.5h3.25a.75.75 0 0 0 .75-.75v-3.25Z"/>
          </svg>
        </button>
        <button class="bt-btn" @click="toggleMode" :title="terminalStore.panel.mode === 'pinned' ? 'Float' : 'Pin'">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M2.75 2h10.5a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H2.75a.75.75 0 0 1-.75-.75V2.75A.75.75 0 0 1 2.75 2Zm.75 1.5v9h9v-9h-9Z"/>
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
</style>
