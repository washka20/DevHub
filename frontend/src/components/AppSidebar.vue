<script setup lang="ts">
import { computed } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { useDockerStore } from '../stores/docker'
import { useGitStore } from '../stores/git'
import ProjectSelector from './ProjectSelector.vue'
import IconDashboard from './icons/IconDashboard.vue'
import IconGit from './icons/IconGit.vue'
import IconCommands from './icons/IconCommands.vue'
import IconDocker from './icons/IconDocker.vue'
import IconTerminal from './icons/IconTerminal.vue'
import IconSettings from './icons/IconSettings.vue'
import IconReadme from './icons/IconReadme.vue'
import IconNotes from './icons/IconNotes.vue'

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
      <div class="nav-group">
        <div class="nav-group-label">Development</div>
        <router-link to="/" class="nav-item" exact-active-class="active">
          <IconDashboard class="nav-icon" />
          Dashboard
        </router-link>
        <router-link to="/git" class="nav-item" active-class="active">
          <IconGit class="nav-icon" />
          Git
          <span v-if="gitChanges > 0" class="badge badge-orange">{{ gitChanges }}</span>
        </router-link>
        <router-link to="/commands" class="nav-item" active-class="active">
          <IconCommands class="nav-icon" />
          Commands
        </router-link>
        <router-link to="/docker" class="nav-item" active-class="active">
          <IconDocker class="nav-icon" />
          Docker
          <span v-if="dockerRunning > 0" class="badge badge-green">{{ dockerRunning }}</span>
        </router-link>
        <router-link to="/console" class="nav-item" active-class="active">
          <IconTerminal class="nav-icon" />
          Console
        </router-link>
      </div>

      <div class="nav-group">
        <div class="nav-group-label">Files</div>
        <router-link to="/readme" class="nav-item" active-class="active">
          <IconReadme class="nav-icon" />
          README
        </router-link>
        <router-link to="/notes" class="nav-item" active-class="active">
          <IconNotes class="nav-icon" />
          Notes
        </router-link>
      </div>
    </nav>

    <div class="sidebar-bottom">
      <router-link to="/settings" class="nav-item" active-class="active">
        <IconSettings class="nav-icon" />
        Settings
      </router-link>
    </div>

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
  background: linear-gradient(180deg, var(--bg-secondary) 0%, var(--bg-primary) 100%);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  z-index: 50;
}

.sidebar-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
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
  overflow-y: auto;
}

.nav-group {
  padding: 4px 0;
}

.nav-group + .nav-group {
  border-top: 1px solid var(--border);
}

.nav-group-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  padding: 10px 16px 4px;
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
  width: 18px;
  height: 18px;
  flex-shrink: 0;
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

.sidebar-bottom {
  border-top: 1px solid var(--border);
  padding: 4px 0;
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
