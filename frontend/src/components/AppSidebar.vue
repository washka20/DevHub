<script setup lang="ts">
import { computed } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { useDockerStore } from '../stores/docker'
import { useGitStore } from '../stores/git'
import ProjectSelector from './ProjectSelector.vue'

const projectsStore = useProjectsStore()
const dockerStore = useDockerStore()
const gitStore = useGitStore()

const gitChanges = computed(() => {
  const s = gitStore.status
  return (s.modified?.length || 0) + (s.staged?.length || 0) + (s.untracked?.length || 0)
})

const dockerRunning = computed(() =>
  (dockerStore.containers || []).filter((c) => c.state === 'running').length
)
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span class="logo">
        <span class="logo-dot"></span>
        DevHub
      </span>
    </div>

    <ProjectSelector />

    <nav class="sidebar-nav">
      <router-link to="/" class="nav-item" exact-active-class="active">
        <span class="nav-icon">&#9783;</span>
        Dashboard
      </router-link>
      <router-link to="/git" class="nav-item" active-class="active">
        <span class="nav-icon">&#9741;</span>
        Git
        <span v-if="gitChanges > 0" class="badge badge-orange">{{ gitChanges }}</span>
      </router-link>
      <router-link to="/commands" class="nav-item" active-class="active">
        <span class="nav-icon">&#9654;</span>
        Commands
      </router-link>
      <router-link to="/docker" class="nav-item" active-class="active">
        <span class="nav-icon">&#9964;</span>
        Docker
        <span v-if="dockerRunning > 0" class="badge badge-green">{{ dockerRunning }}</span>
      </router-link>
      <router-link to="/console" class="nav-item" active-class="active">
        <span class="nav-icon">&#9002;</span>
        Console
      </router-link>
      <router-link to="/settings" class="nav-item" active-class="active">
        <span class="nav-icon">&#9881;</span>
        Settings
      </router-link>
    </nav>

    <div class="sidebar-footer">
      <div class="footer-hint">
        <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor" opacity="0.5">
          <path d="M8 0a8 8 0 1 1 0 16A8 8 0 0 1 8 0zM1.5 8a6.5 6.5 0 1 0 13 0 6.5 6.5 0 0 0-13 0zm4.879-2.773l4.264 2.559a.25.25 0 0 1 0 .428l-4.264 2.559A.25.25 0 0 1 6 10.559V5.442a.25.25 0 0 1 .379-.215z"/>
        </svg>
        <span>{{ projectsStore.projects.length }} projects</span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  height: 100vh;
  position: fixed;
  top: 0;
  left: 0;
  background: linear-gradient(180deg, var(--bg-secondary) 0%, #12161c 100%);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  z-index: 50;
}

.sidebar-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.sidebar-nav {
  padding: 4px 0;
}

.logo {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-green);
  box-shadow: 0 0 6px var(--accent-green);
}

.sidebar-nav {
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  color: var(--text-secondary);
  font-size: 14px;
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  color: var(--text-primary);
  background: var(--bg-tertiary);
  border-left: 2px solid var(--accent-blue);
  box-shadow: inset 3px 0 8px rgba(88, 166, 255, 0.1);
}

.nav-icon {
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.badge {
  margin-left: auto;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}

.badge-orange {
  background: var(--accent-orange);
}

.badge-green {
  background: var(--accent-green);
}

.sidebar-footer {
  border-top: 1px solid var(--border);
  padding: 12px 16px;
}

.footer-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-secondary);
}
</style>
