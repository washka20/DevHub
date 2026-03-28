<script setup lang="ts">
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

const emit = defineEmits<{
  split: [direction: 'horizontal' | 'vertical']
}>()

async function handleNewTab() {
  const cwd = projectsStore.currentProject?.path || ''
  try {
    await terminalStore.addTab(cwd)
  } catch {
    // max sessions reached or backend unavailable
  }
}
</script>

<template>
  <div class="tab-bar">
    <div class="tabs">
      <div
        v-for="tab in terminalStore.tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: terminalStore.activeTabId === tab.id }"
        @click="terminalStore.setActiveTab(tab.id)"
      >
        <span class="tab-dot" :class="{ active: terminalStore.activeTabId === tab.id }"></span>
        <span class="tab-label">{{ tab.label }}</span>
        <span class="tab-close" @click.stop="terminalStore.closeTab(tab.id)">&#10005;</span>
      </div>
      <button class="tab-add" @click="handleNewTab" title="New terminal">+</button>
    </div>

    <div class="toolbar">
      <button
        class="toolbar-btn"
        :class="{ active: terminalStore.activeTab?.splitDirection === 'horizontal' }"
        @click="emit('split', 'horizontal')"
        :disabled="!terminalStore.activeTab || (terminalStore.activeTab?.panes.length ?? 0) >= 2"
        title="Split horizontal"
      >
        &#9776; Split H
      </button>
      <button
        class="toolbar-btn"
        :class="{ active: terminalStore.activeTab?.splitDirection === 'vertical' }"
        @click="emit('split', 'vertical')"
        :disabled="!terminalStore.activeTab || (terminalStore.activeTab?.panes.length ?? 0) >= 2"
        title="Split vertical"
      >
        &#9783; Split V
      </button>
    </div>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  background: #161b22;
  border-bottom: 1px solid var(--border);
  padding: 0 8px;
  height: 36px;
  flex-shrink: 0;
}

.tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
}

.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 6px 6px 0 0;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
}

.tab.active {
  background: #0d1117;
  border: 1px solid var(--border);
  border-bottom: 1px solid #0d1117;
  margin-bottom: -1px;
  color: var(--text-primary);
}

.tab-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-secondary);
}

.tab-dot.active {
  background: var(--accent-green);
}

.tab-close {
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.5;
  cursor: pointer;
  padding: 0 2px;
}

.tab-close:hover {
  opacity: 1;
  color: var(--accent-red, #f85149);
}

.tab-add {
  padding: 4px 8px;
  font-size: 14px;
  color: var(--text-secondary);
  cursor: pointer;
  border: none;
  background: none;
  border-radius: 4px;
}

.tab-add:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.toolbar {
  display: flex;
  gap: 4px;
  align-items: center;
}

.toolbar-btn {
  padding: 3px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  cursor: pointer;
  background: none;
  white-space: nowrap;
}

.toolbar-btn:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--text-secondary);
}

.toolbar-btn.active {
  color: var(--accent-blue);
  border-color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.1);
}

.toolbar-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
