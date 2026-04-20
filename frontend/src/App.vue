<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import AppSidebar from './components/AppSidebar.vue'
import BottomTerminal from './components/BottomTerminal.vue'
import { useProject } from './composables/useProject'
import { useWebSocket } from './composables/useWebSocket'
import { useKeyboardShortcuts, onToggleFileSearch } from './composables/useKeyboardShortcuts'
import { useDockerStore } from './stores/docker'
import { useGitStore } from './stores/git'
import { useFilesStore } from './stores/files'
import { useSettingsStore } from './stores/settings'
import { useTerminalStore } from './stores/terminal'
import SearchModal from './components/SearchModal.vue'
import ToastContainer from './components/ToastContainer.vue'
import CommandPalette from './components/CommandPalette.vue'

const { initProject } = useProject()
useSettingsStore()
const { connect, onMessage } = useWebSocket()
useKeyboardShortcuts()
const showSearch = ref(false)
onToggleFileSearch(() => { showSearch.value = !showSearch.value })
const dockerStore = useDockerStore()
const gitStore = useGitStore()
const terminalStore = useTerminalStore()
const route = useRoute()

const transitionName = ref('slide-forward')
let prevOrder = (route.meta.order as number) ?? 0

watch(() => route.meta.order, (newOrder) => {
  const n = (newOrder as number) ?? 0
  transitionName.value = n >= prevOrder ? 'slide-forward' : 'slide-backward'
  prevOrder = n
})

const showBottomPanel = computed(() =>
  terminalStore.panel.visible && route.path !== '/console'
)

function handlePanelResize(panes: Array<{ size: number }>) {
  if (panes.length === 2) {
    terminalStore.updatePanel({ height: panes[1].size })
  }
}

let floatingCleanup: (() => void) | null = null

function startFloatingDrag(e: MouseEvent) {
  e.preventDefault()
  const startX = e.clientX
  const startY = e.clientY
  const startPos = { ...terminalStore.panel.floatingPos }

  function onMove(ev: MouseEvent) {
    const maxX = window.innerWidth - 100
    const maxY = window.innerHeight - 50
    terminalStore.updatePanel({
      floatingPos: {
        ...terminalStore.panel.floatingPos,
        x: Math.max(0, Math.min(maxX, startPos.x + (ev.clientX - startX))),
        y: Math.max(0, Math.min(maxY, startPos.y + (ev.clientY - startY))),
      },
    })
  }
  function onUp() {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    floatingCleanup = null
  }
  floatingCleanup?.()
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
  floatingCleanup = onUp
}

function startFloatingResize(e: MouseEvent) {
  e.preventDefault()
  const startX = e.clientX
  const startY = e.clientY
  const startPos = { ...terminalStore.panel.floatingPos }

  function onMove(ev: MouseEvent) {
    terminalStore.updatePanel({
      floatingPos: {
        ...terminalStore.panel.floatingPos,
        w: Math.max(300, startPos.w + (ev.clientX - startX)),
        h: Math.max(200, startPos.h + (ev.clientY - startY)),
      },
    })
  }
  function onUp() {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    floatingCleanup = null
  }
  floatingCleanup?.()
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
  floatingCleanup = onUp
}

onMounted(async () => {
  await initProject()

  connect()

  onMessage((data) => {
    const event = data as { type?: string; data?: unknown }
    if (event.type === 'docker:update') {
      dockerStore.fetchContainers()
    }
    if (event.type === 'git:update') {
      gitStore.fetchStatus()
      gitStore.fetchGraph()
    }
    if (event.type === 'files_changed') {
      gitStore.fetchStatus()
      try {
        const filesStore = useFilesStore()
        filesStore.fetchTree()
        const paths = Array.isArray(event.data) ? (event.data as string[]) : []
        if (paths.length > 0) {
          filesStore.checkOpenFiles(paths)
        }
      } catch {
        // files store not initialized yet, ignore
      }
    }
  })

})
</script>

<template>
  <div class="app-layout">
    <AppSidebar />
    <div class="main-area">
      <Splitpanes horizontal class="app-splitpanes" @resized="handlePanelResize">
        <Pane :size="showBottomPanel ? 100 - terminalStore.panel.height : 100">
          <main class="main-content">
            <router-view v-slot="{ Component, route: r }">
              <Transition :name="transitionName" mode="out-in">
                <keep-alive include="ConsoleView">
                  <component :is="Component" :key="r.name" class="route-view" />
                </keep-alive>
              </Transition>
            </router-view>
          </main>
        </Pane>
        <Pane v-if="showBottomPanel && terminalStore.panel.mode === 'pinned'" :size="terminalStore.panel.height" :min-size="10" :max-size="80">
          <BottomTerminal />
        </Pane>
      </Splitpanes>
      <div
        v-if="showBottomPanel && terminalStore.panel.mode === 'floating'"
        class="floating-terminal"
        :style="{
          left: terminalStore.panel.floatingPos.x + 'px',
          top: terminalStore.panel.floatingPos.y + 'px',
          width: terminalStore.panel.floatingPos.w + 'px',
          height: terminalStore.panel.floatingPos.h + 'px',
        }"
      >
        <div class="floating-drag-handle" @mousedown="startFloatingDrag">
          <span class="floating-title">{{ terminalStore.activeTab?.label || 'Terminal' }}</span>
          <div class="floating-actions">
            <button class="floating-btn" @click="terminalStore.setPanelMode('pinned')" title="Pin">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                <path d="M2.75 2h10.5a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H2.75a.75.75 0 0 1-.75-.75V2.75A.75.75 0 0 1 2.75 2Zm.75 1.5v9h9v-9h-9Z"/>
              </svg>
            </button>
            <button class="floating-btn" @click="terminalStore.updatePanel({ visible: false })" title="Close">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/>
              </svg>
            </button>
          </div>
        </div>
        <div class="floating-body">
          <BottomTerminal />
        </div>
        <div class="floating-resize-handle" @mousedown="startFloatingResize"></div>
      </div>
    </div>
  </div>
  <SearchModal :visible="showSearch" @close="showSearch = false" />
  <CommandPalette />
  <ToastContainer />
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
}

.main-area {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: var(--sidebar-width);
  display: flex;
  flex-direction: column;
}

.app-splitpanes {
  flex: 1;
  min-height: 0;
}

.main-content {
  padding: 16px 32px;
  overflow-y: auto;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.route-view {
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

.floating-terminal {
  position: fixed;
  z-index: 200;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  overflow: hidden;
}

.floating-drag-handle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  background: var(--bg-secondary);
  cursor: grab;
  user-select: none;
  flex-shrink: 0;
}

.floating-drag-handle:active {
  cursor: grabbing;
}

.floating-title {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
}

.floating-actions {
  display: flex;
  gap: 2px;
}

.floating-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
}

.floating-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.floating-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.floating-resize-handle {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 12px;
  height: 12px;
  cursor: nwse-resize;
}

.floating-resize-handle::after {
  content: '';
  position: absolute;
  right: 3px;
  bottom: 3px;
  width: 6px;
  height: 6px;
  border-right: 2px solid var(--text-secondary);
  border-bottom: 2px solid var(--text-secondary);
  opacity: 0.4;
}
</style>
