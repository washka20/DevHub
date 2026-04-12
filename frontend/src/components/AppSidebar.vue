<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { onToggleShortcuts } from '../composables/useKeyboardShortcuts'
import { useProjectsStore } from '../stores/projects'
import { useDockerStore } from '../stores/docker'
import { useGitStore } from '../stores/git'
import { useGitLabStore } from '../stores/gitlab'
import ProjectSelector from './ProjectSelector.vue'
import ShortcutsModal from './ShortcutsModal.vue'
import IconDashboard from './icons/IconDashboard.vue'
import IconGit from './icons/IconGit.vue'
import IconCommands from './icons/IconCommands.vue'
import IconDocker from './icons/IconDocker.vue'
import IconTerminal from './icons/IconTerminal.vue'
import IconSettings from './icons/IconSettings.vue'
import IconReadme from './icons/IconReadme.vue'
import IconNotes from './icons/IconNotes.vue'
import IconEditor from './icons/IconEditor.vue'
import IconGitLab from './icons/IconGitLab.vue'

const projectsStore = useProjectsStore()
const dockerStore = useDockerStore()
const gitStore = useGitStore()
const gitlabStore = useGitLabStore()
const route = useRoute()

const showShortcuts = ref(false)

let unsubShortcuts: (() => void) | undefined
onMounted(() => {
  gitlabStore.checkEnabled()
  unsubShortcuts = onToggleShortcuts(() => {
    showShortcuts.value = !showShortcuts.value
  })
})
onBeforeUnmount(() => {
  unsubShortcuts?.()
})

const gitChanges = computed(() => {
  const s = gitStore.status
  return (s.modified?.length || 0) + (s.staged?.length || 0) + (s.untracked?.length || 0)
})

const dockerRunning = computed(() =>
  (dockerStore.containers || []).filter((c) => c.state === 'running').length
)

const gitlabEnabled = computed(() => gitlabStore.enabled === true)
const gitlabTodos = computed(() => gitlabStore.todosCount)

const devNavItems = ['/', '/git', '/commands', '/docker', '/gitlab', '/console']
const filesNavItems = ['/readme', '/notes', '/editor']

const devIndicatorTop = computed(() => {
  const idx = devNavItems.indexOf(route.path)
  return idx >= 0 ? idx * 36 : -1
})

const filesIndicatorTop = computed(() => {
  const idx = filesNavItems.indexOf(route.path)
  return idx >= 0 ? idx * 36 : -1
})

const settingsActive = computed(() => route.path === '/settings')
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
        <div class="nav-wrap">
          <div
            v-if="devIndicatorTop >= 0"
            class="nav-indicator"
            :style="{ top: devIndicatorTop + 'px' }"
          ></div>
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
          <router-link to="/gitlab" class="nav-item" active-class="active">
            <IconGitLab class="nav-icon" />
            GitLab
            <span v-if="gitlabTodos > 0" class="badge badge-orange">{{ gitlabTodos }}</span>
          </router-link>
          <router-link to="/console" class="nav-item" active-class="active">
            <IconTerminal class="nav-icon" />
            Console
          </router-link>
        </div>
      </div>

      <div class="nav-group">
        <div class="nav-group-label">Files</div>
        <div class="nav-wrap">
          <div
            v-if="filesIndicatorTop >= 0"
            class="nav-indicator"
            :style="{ top: filesIndicatorTop + 'px' }"
          ></div>
          <router-link to="/readme" class="nav-item" active-class="active">
            <IconReadme class="nav-icon" />
            README
          </router-link>
          <router-link to="/notes" class="nav-item" active-class="active">
            <IconNotes class="nav-icon" />
            Notes
          </router-link>
          <router-link to="/editor" class="nav-item" active-class="active">
            <IconEditor class="nav-icon" />
            Editor
          </router-link>
        </div>
      </div>
    </nav>

    <div class="sidebar-bottom">
      <div class="nav-wrap">
        <div
          v-if="settingsActive"
          class="nav-indicator"
          style="top: 0"
        ></div>
        <router-link to="/settings" class="nav-item" active-class="active">
          <IconSettings class="nav-icon" />
          Settings
        </router-link>
      </div>
    </div>

    <div class="sidebar-footer">
      <div class="footer-hint">
        <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor" opacity="0.5">
          <path d="M8 0a8 8 0 1 1 0 16A8 8 0 0 1 8 0zM1.5 8a6.5 6.5 0 1 0 13 0 6.5 6.5 0 0 0-13 0zm4.879-2.773l4.264 2.559a.25.25 0 0 1 0 .428l-4.264 2.559A.25.25 0 0 1 6 10.559V5.442a.25.25 0 0 1 .379-.215z"/>
        </svg>
        <span>{{ projectsStore.projects.length }} projects</span>
      </div>
      <button class="shortcuts-btn" @click="showShortcuts = true">
        <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
          <path d="M0 6a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V6zm13 .25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5a.25.25 0 0 0-.25.25zM2.25 8a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 3 8.75v-.5A.25.25 0 0 0 2.75 8h-.5zM4 8.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 5 8.75v-.5A.25.25 0 0 0 4.75 8h-.5a.25.25 0 0 0-.25.25zM6.25 8a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h3.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-3.5zM8 6.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 9 6.75v-.5A.25.25 0 0 0 8.75 6h-.5a.25.25 0 0 0-.25.25zM11.25 6a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5zM10 6.25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5a.25.25 0 0 0-.25.25zM7.25 6a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 8 6.75v-.5A.25.25 0 0 0 7.75 6h-.5zM6 6.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 7 6.75v-.5A.25.25 0 0 0 6.75 6h-.5a.25.25 0 0 0-.25.25zM5.25 6a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 6 6.75v-.5A.25.25 0 0 0 5.75 6h-.5zM4 6.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 5 6.75v-.5A.25.25 0 0 0 4.75 6h-.5a.25.25 0 0 0-.25.25zM11.25 10.5a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5zM2.25 10a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5zM4 10.25v.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-7.5a.25.25 0 0 0-.25.25zM13.25 10a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5a.25.25 0 0 0 .25-.25v-.5a.25.25 0 0 0-.25-.25h-.5zM2.25 6a.25.25 0 0 0-.25.25v.5c0 .138.112.25.25.25h.5A.25.25 0 0 0 3 6.75v-.5A.25.25 0 0 0 2.75 6h-.5z"/>
        </svg>
        Shortcuts
      </button>
    </div>

    <ShortcutsModal :visible="showShortcuts" @close="showShortcuts = false" />
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

.nav-wrap {
  position: relative;
}

.nav-indicator {
  position: absolute;
  left: 0;
  right: 0;
  height: 36px;
  background: rgba(88, 166, 255, 0.08);
  border-left: 2px solid var(--accent-blue);
  border-radius: 0 6px 6px 0;
  transition: top var(--transition-smooth);
  pointer-events: none;
  z-index: 0;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  color: var(--text-primary);
  background: transparent;
  position: relative;
  z-index: 1;
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

.sidebar-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.footer-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-secondary);
}

.shortcuts-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--accent-blue);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 8px;
  transition: background var(--transition-fast), border-color var(--transition-fast);
}

.shortcuts-btn:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-blue);
}
</style>
