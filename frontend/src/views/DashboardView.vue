<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectsStore } from '../stores/projects'
import { useGitStore } from '../stores/git'
import { useDockerStore } from '../stores/docker'
import StatusCard from '../components/StatusCard.vue'
import CommandButton from '../components/CommandButton.vue'
import type { MakeCommand } from '../types'

const router = useRouter()

const projectsStore = useProjectsStore()
const gitStore = useGitStore()
const dockerStore = useDockerStore()

const loadingCmd = ref<string | null>(null)
const commands = ref<MakeCommand[]>([])

const currentProject = computed(() => projectsStore.currentProject)

async function fetchCommands() {
  if (!currentProject.value?.has_makefile) {
    commands.value = []
    return
  }
  try {
    const res = await fetch(`/api/projects/${currentProject.value.name}/commands`)
    if (res.ok) commands.value = await res.json()
  } catch { commands.value = [] }
}

const quickActions = computed(() => commands.value)

watch(() => currentProject.value?.name, () => fetchCommands(), { immediate: true })

const gitChanges = computed(() => {
  const s = gitStore.status
  return (s.modified?.length || 0) + (s.staged?.length || 0) + (s.untracked?.length || 0)
})

const gitCardColor = computed(() =>
  gitChanges.value > 0 ? 'var(--accent-orange)' : 'var(--accent-green)'
)

const dockerRunning = computed(() =>
  (dockerStore.containers || []).filter((c) => c.state === 'running').length
)

const dockerTotal = computed(() => (dockerStore.containers || []).length)

const dockerCardColor = computed(() => {
  if (dockerTotal.value === 0) return 'var(--text-secondary)'
  return dockerRunning.value === dockerTotal.value
    ? 'var(--accent-green)'
    : 'var(--accent-red)'
})

const dockerUp = computed(() => `${dockerRunning.value}/${dockerTotal.value}`)

const lastCommitMessage = computed(() => {
  if (!gitStore.log?.length) return '---'
  const msg = gitStore.log[0].message
  return msg.length > 60 ? msg.slice(0, 57) + '...' : msg
})

const lastCommitTime = computed(() => {
  if (!gitStore.log?.length) return ''
  return gitStore.log[0].date
})


function containerPillClass(state: string): string {
  if (state === 'running') return 'pill-green'
  if (state === 'restarting') return 'pill-yellow'
  return 'pill-red'
}

async function executeCommand(name: string) {
  if (!currentProject.value) return
  loadingCmd.value = name
  // Strip "make " prefix — backend runs make internally
  const cmd = name.replace(/^make\s+/, '')
  try {
    await fetch(`/api/projects/${currentProject.value.name}/exec`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cmd }),
    })
  } finally {
    loadingCmd.value = null
  }
}
</script>

<template>
  <div class="dashboard">
    <header class="page-header">
      <h1>{{ currentProject?.name ?? 'DevHub' }}</h1>
      <div class="header-meta">
        <span v-if="currentProject" class="meta-path">{{ currentProject.path }}</span>
        <span v-if="gitStore.status.branch" class="meta-branch">
          &#9741; {{ gitStore.status.branch }}
        </span>
      </div>
    </header>

    <section class="cards-row">
      <StatusCard
        label="Git"
        :value="gitStore.status.branch || '---'"
        :subtext="`${gitChanges} changes`"
        :color="gitCardColor"
        to="/git"
      />
      <StatusCard
        label="Docker"
        :value="dockerUp"
        subtext="containers UP"
        :color="dockerCardColor"
        to="/docker"
      />
      <StatusCard
        label="Last Commit"
        :value="lastCommitMessage"
        :subtext="lastCommitTime"
        color="var(--text-primary)"
        to="/git"
      />
    </section>

    <section v-if="currentProject?.has_makefile" class="section">
      <h2>Quick Actions</h2>
      <div class="actions-grid">
        <CommandButton
          v-for="action in quickActions"
          :key="action.name"
          :name="action.name"
          :description="action.description"
          :category="action.category"
          :loading="loadingCmd === action.name"
          @execute="executeCommand(action.name)"
        />
      </div>
    </section>

    <section v-if="currentProject?.has_docker && dockerStore.containers?.length" class="section">
      <h2>Containers</h2>
      <div class="containers-pills">
        <span
          v-for="container in (dockerStore.containers || [])"
          :key="container.name"
          class="pill clickable"
          :class="containerPillClass(container.state)"
          @click="router.push('/docker')"
        >
          {{ container.name }}
        </span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard {
}

.page-header {
  margin-bottom: 16px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 4px;
}

.header-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.meta-branch {
  color: var(--accent-blue);
  font-family: var(--font-mono);
}

.cards-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.section {
  margin-bottom: 32px;
}

.section h2 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
  color: var(--text-primary);
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}

.containers-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pill {
  padding: 6px 14px;
  border-radius: 14px;
  font-size: 13px;
  font-weight: 500;
  font-family: var(--font-mono);
  transition: box-shadow var(--transition-fast), transform var(--transition-fast);
  cursor: default;
}

.pill.clickable {
  cursor: pointer;
}

.pill:hover {
  transform: translateY(-1px);
}

.pill-green {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
  border: 1px solid rgba(63, 185, 80, 0.3);
}

.pill-green:hover {
  box-shadow: var(--glow-green);
}

.pill-red {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
  border: 1px solid rgba(248, 81, 73, 0.3);
}

.pill-red:hover {
  box-shadow: var(--glow-red);
}

.pill-yellow {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
  border: 1px solid rgba(210, 153, 34, 0.3);
}

.pill-yellow:hover {
  box-shadow: var(--glow-orange);
}
</style>
