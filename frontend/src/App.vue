<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import AppSidebar from './components/AppSidebar.vue'
import BottomTerminal from './components/BottomTerminal.vue'
import { useProject } from './composables/useProject'
import { useWebSocket } from './composables/useWebSocket'
import { useKeyboardShortcuts } from './composables/useKeyboardShortcuts'
import { useDockerStore } from './stores/docker'
import { useGitStore } from './stores/git'
import { useSettingsStore } from './stores/settings'
import { useTerminalStore } from './stores/terminal'

const { initProject } = useProject()
useSettingsStore()
const { connect, onMessage } = useWebSocket()
useKeyboardShortcuts()
const dockerStore = useDockerStore()
const gitStore = useGitStore()
const terminalStore = useTerminalStore()
const route = useRoute()

const showBottomPanel = computed(() =>
  terminalStore.panel.visible && route.path !== '/console'
)

function handlePanelResize(panes: Array<{ size: number }>) {
  if (panes.length === 2) {
    terminalStore.updatePanel({ height: panes[1].size })
  }
}

onMounted(async () => {
  await initProject()

  connect()

  onMessage((data) => {
    const event = data as { type?: string }
    if (event.type === 'docker:update') {
      dockerStore.fetchContainers()
    }
    if (event.type === 'git:update') {
      gitStore.fetchStatus()
      gitStore.fetchGraph()
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
              <keep-alive include="ConsoleView">
                <component :is="Component" :key="r.name" class="route-view" />
              </keep-alive>
            </router-view>
          </main>
        </Pane>
        <Pane v-if="showBottomPanel" :size="terminalStore.panel.height" :min-size="10" :max-size="80">
          <BottomTerminal />
        </Pane>
      </Splitpanes>
    </div>
  </div>
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
</style>
