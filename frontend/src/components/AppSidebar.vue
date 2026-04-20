<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { onToggleShortcuts } from '../composables/useKeyboardShortcuts'
import { useDockerStore } from '../stores/docker'
import { useGitStore } from '../stores/git'
import { useGitLabStore } from '../stores/gitlab'
import ProjectSelector from './ProjectSelector.vue'
import ShortcutsModal from './ShortcutsModal.vue'
import ThemeToggle from './ThemeToggle.vue'
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
import { useProjectsStore } from '../stores/projects'

const dockerStore = useDockerStore()
const gitStore = useGitStore()
const gitlabStore = useGitLabStore()
const projectsStore = useProjectsStore()

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
  (dockerStore.containers || []).filter((c) => c.state === 'running').length,
)

const gitlabTodos = computed(() => gitlabStore.todosCount)

const userInitial = computed(() => {
  const name = projectsStore.currentProject?.name || 'U'
  return name.slice(0, 1).toUpperCase()
})

const isMac = typeof navigator !== 'undefined' && /Mac/i.test(navigator.platform || '')
const metaKey = isMac ? '⌘K' : 'Ctrl+K'
</script>

<template>
  <aside class="sb app-sb">
    <div class="sb-head">
      <router-link to="/" class="logo">
        <span class="logo-mark"></span>
        <span>DevHub</span>
      </router-link>
      <button class="sb-collapse" title="Shortcuts" @click="showShortcuts = true" aria-label="Shortcuts">
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M4 6l4 4 4-4"/>
        </svg>
      </button>
    </div>

    <ProjectSelector />

    <nav class="sb-nav">
      <div class="nav-group">
        <div class="nav-label">Workspace</div>
        <router-link to="/" class="ds-nav-item" exact-active-class="active">
          <IconDashboard />
          <span>Dashboard</span>
        </router-link>
        <router-link to="/git" class="ds-nav-item" active-class="active">
          <IconGit />
          <span>Git</span>
          <span v-if="gitChanges > 0" class="badge orange">{{ gitChanges }}</span>
        </router-link>
        <router-link to="/commands" class="ds-nav-item" active-class="active">
          <IconCommands />
          <span>Commands</span>
        </router-link>
        <router-link to="/docker" class="ds-nav-item" active-class="active">
          <IconDocker />
          <span>Docker</span>
          <span v-if="dockerRunning > 0" class="badge green">{{ dockerRunning }}</span>
        </router-link>
        <router-link to="/gitlab" class="ds-nav-item" active-class="active">
          <IconGitLab />
          <span>GitLab</span>
          <span v-if="gitlabTodos > 0" class="badge info">{{ gitlabTodos }}</span>
        </router-link>
        <router-link to="/console" class="ds-nav-item" active-class="active">
          <IconTerminal />
          <span>Console</span>
        </router-link>
      </div>

      <div class="nav-group">
        <div class="nav-label">Files</div>
        <router-link to="/editor" class="ds-nav-item" active-class="active">
          <IconEditor />
          <span>Editor</span>
        </router-link>
        <router-link to="/notes" class="ds-nav-item" active-class="active">
          <IconNotes />
          <span>Notes</span>
        </router-link>
        <router-link to="/readme" class="ds-nav-item" active-class="active">
          <IconReadme />
          <span>README</span>
        </router-link>
      </div>

      <div class="nav-group">
        <div class="nav-label">System</div>
        <router-link to="/settings" class="ds-nav-item" active-class="active">
          <IconSettings />
          <span>Settings</span>
        </router-link>
      </div>
    </nav>

    <div class="sb-bottom">
      <div class="user">
        <span class="avatar">{{ userInitial }}</span>
        <span class="kbd" :title="metaKey">{{ metaKey }}</span>
      </div>
      <ThemeToggle />
    </div>

    <ShortcutsModal :visible="showShortcuts" @close="showShortcuts = false" />
  </aside>
</template>

<style scoped>
.app-sb {
  width: var(--sidebar-width);
  height: 100vh;
  position: fixed;
  top: 0;
  left: 0;
  z-index: 50;
}

/* router-link active gets `.active` (via active-class prop) so the design-system
 * `.ds-nav-item.active` rules in components.css handle the amber stripe + glow. */
.ds-nav-item,
.ds-nav-item:visited { color: var(--fg-2); }
.ds-nav-item > :deep(svg) { width: 16px; height: 16px; flex-shrink: 0; color: currentColor; opacity: .85; }
</style>
