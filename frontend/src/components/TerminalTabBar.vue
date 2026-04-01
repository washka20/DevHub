<script setup lang="ts">
import { ref } from 'vue'
import { useTerminalStore } from '../stores/terminal'
import { useProjectsStore } from '../stores/projects'
import TabContextMenu from './TabContextMenu.vue'
import type { TerminalTab } from '../types'

const terminalStore = useTerminalStore()
const projectsStore = useProjectsStore()

const emit = defineEmits<{
  split: [direction: 'horizontal' | 'vertical']
}>()

const contextMenu = ref<{ x: number; y: number; tabId: string } | null>(null)
const renamingTabId = ref<string | null>(null)
const renameValue = ref('')

function tabHasActivity(tab: TerminalTab): boolean {
  return tab.panes.some((p) => p.hasActivity)
}

function tabHasBell(tab: TerminalTab): boolean {
  return tab.panes.some((p) => p.hasBell)
}

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
        <input
          v-if="renamingTabId === tab.id"
          v-model="renameValue"
          class="tab-rename-input"
          @keydown.enter="finishRename"
          @keydown.escape="cancelRename"
          @blur="finishRename"
          @click.stop
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

    <Teleport to="body">
      <TabContextMenu
        v-if="contextMenu"
        :x="contextMenu.x"
        :y="contextMenu.y"
        :tab-id="contextMenu.tabId"
        :can-split="(terminalStore.tabs.find(t => t.id === contextMenu?.tabId)?.panes.length ?? 0) < 2"
        @close="contextMenu = null"
        @rename="startRename"
        @split-h="(id: string) => handleSplitFromMenu(id, 'horizontal')"
        @split-v="(id: string) => handleSplitFromMenu(id, 'vertical')"
        @close-tab="terminalStore.closeTab"
        @close-others="terminalStore.closeOtherTabs"
        @close-all="terminalStore.closeAllTabs"
      />
    </Teleport>
  </div>
</template>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  background: var(--bg-secondary);
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
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-bottom: 1px solid var(--bg-primary);
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
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.5;
  cursor: pointer;
  padding: 0 2px;
  background: none;
  border: none;
  line-height: 1;
}

.tab-close:hover {
  opacity: 1;
  color: var(--accent-red, #f85149);
}

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
</style>
