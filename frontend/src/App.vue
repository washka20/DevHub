<script setup lang="ts">
import { onMounted } from 'vue'
import AppSidebar from './components/AppSidebar.vue'
import { useProject } from './composables/useProject'
import { useWebSocket } from './composables/useWebSocket'
import { useDockerStore } from './stores/docker'
import { useGitStore } from './stores/git'

const { initProject } = useProject()
const { connect, onMessage } = useWebSocket()
const dockerStore = useDockerStore()
const gitStore = useGitStore()

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
    <main class="main-content">
      <router-view v-slot="{ Component }">
        <component :is="Component" class="route-view" />
      </router-view>
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
}

.main-content {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: var(--sidebar-width);
  padding: 16px 32px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.route-view {
  flex: 1;
  min-height: 0;
}
</style>
