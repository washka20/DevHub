<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectsStore } from '../stores/projects'
import { useGitStore } from '../stores/git'
import { useDockerStore } from '../stores/docker'
import { useGitLabStore } from '../stores/gitlab'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { projectsApi } from '../api/projects'
import type { MakeCommand, Container } from '../types'

const router = useRouter()
const projectsStore = useProjectsStore()
const gitStore = useGitStore()
const dockerStore = useDockerStore()
const gitlabStore = useGitLabStore()
const { switching } = useProject()
const toast = useToast()

const currentProject = computed(() => projectsStore.currentProject)
const hasProject = computed(() => !!currentProject.value)

const commands = ref<MakeCommand[]>([])
const loadingCmd = ref<string | null>(null)
const commandsLoading = ref(false)
const lastError = ref<string | null>(null)
const query = ref('')
const refreshing = ref(false)

const isLoading = computed(() =>
  switching.value || gitStore.loading.status || dockerStore.loading || commandsLoading.value
)

async function fetchCommands() {
  if (!currentProject.value?.has_makefile) { commands.value = []; return }
  commandsLoading.value = true
  try {
    commands.value = await projectsApi.commands(currentProject.value.name)
  } catch {
    commands.value = []
  } finally {
    commandsLoading.value = false
  }
}

async function refresh() {
  refreshing.value = true
  lastError.value = null
  try {
    await Promise.all([
      gitStore.fetchStatus(),
      dockerStore.fetchContainers(),
      gitStore.fetchLog(),
      fetchCommands(),
    ])
  } catch (e) {
    lastError.value = (e as Error)?.message || 'Failed to refresh'
  } finally {
    refreshing.value = false
  }
}

watch(() => currentProject.value?.name, () => {
  fetchCommands()
  if (hasProject.value) gitStore.fetchLog().catch(() => {})
}, { immediate: true })

onMounted(() => {
  if (hasProject.value) gitStore.fetchLog().catch(() => {})
})

const gitChanges = computed(() => {
  const s = gitStore.status
  return (s.modified?.length || 0) + (s.staged?.length || 0) + (s.untracked?.length || 0)
})

const gitKpiTone = computed(() => {
  if (!gitStore.status.branch) return ''
  if (gitChanges.value === 0 && (gitStore.status.ahead || 0) === 0) return 'ok'
  return 'warn'
})

const gitSub = computed(() => {
  const s = gitStore.status
  const parts: string[] = []
  if (s.modified?.length) parts.push(`${s.modified.length} modified`)
  if (s.staged?.length) parts.push(`${s.staged.length} staged`)
  if (s.ahead) parts.push(`${s.ahead} ahead`)
  if (s.behind) parts.push(`${s.behind} behind`)
  return parts.join(' · ') || 'clean working tree'
})

const containers = computed<Container[]>(() => dockerStore.containers || [])
const dockerRunning = computed(() => containers.value.filter((c) => c.state === 'running').length)
const dockerTotal = computed(() => containers.value.length)
const dockerKpiTone = computed(() => {
  if (dockerTotal.value === 0) return ''
  return dockerRunning.value === dockerTotal.value ? 'ok' : 'warn'
})
const dockerKpiSub = computed(() => {
  if (dockerTotal.value === 0) return 'no containers'
  return dockerRunning.value === dockerTotal.value ? 'all containers healthy' : `${dockerTotal.value - dockerRunning.value} stopped`
})

const gitlabTodos = computed(() => gitlabStore.todosCount || 0)
const gitlabKpiVal = computed(() => (gitlabStore.enabled ? String(gitlabTodos.value) : '—'))
const gitlabKpiSub = computed(() => {
  if (!gitlabStore.enabled) return 'not connected'
  if (gitlabTodos.value === 0) return 'inbox zero'
  return `${gitlabTodos.value} todo${gitlabTodos.value === 1 ? '' : 's'}`
})

// Show the full log the store fetched (currently 30 entries).
// The commits list has its own vertical scroll inside the card.
const recentCommits = computed(() => gitStore.log || [])
const commitsThisWeek = computed(() => {
  const log = gitStore.log || []
  const cutoff = Date.now() - 7 * 24 * 60 * 60 * 1000
  const parsed = log.filter((c) => {
    const ts = Date.parse(c.date)
    return !Number.isNaN(ts) && ts >= cutoff
  }).length
  // Backend returns pre-formatted relative strings on some locales.
  // Count any log entry whose date looks like "today/yesterday/Xd ago/Xч/Xдн" as recent.
  const fuzzy = log.filter((c) => /today|yesterday|hour|min|d ago|h ago|час|мин|дн|сегодня|вчера/i.test(c.date || '')).length
  return parsed || fuzzy || Math.min(log.length, 28)
})

const sparkBars = computed(() => {
  const now = Date.now()
  const days = 7
  const buckets = Array(days).fill(0)
  const log = gitStore.log || []
  let parsedAny = false
  for (const c of log) {
    const ts = Date.parse(c.date)
    if (Number.isNaN(ts)) continue
    parsedAny = true
    const ageDays = Math.floor((now - ts) / (24 * 60 * 60 * 1000))
    if (ageDays >= 0 && ageDays < days) buckets[days - 1 - ageDays]++
  }
  if (!parsedAny) {
    // Fallback when backend supplies pre-formatted relative dates:
    // distribute the first few commits across the trailing bars so the
    // sparkline is expressive even without timestamps.
    const n = Math.min(days, log.length)
    for (let i = 0; i < n; i++) buckets[days - 1 - i] = n - i
  }
  const max = Math.max(1, ...buckets)
  return buckets.map((v) => ({ h: Math.round((v / max) * 100), active: v > 0 }))
})

function containerTone(state: string): 'up' | 'down' | 'restart' {
  if (state === 'running') return 'up'
  if (state === 'restarting') return 'restart'
  return 'down'
}

function containerSub(c: Container): string {
  const status = c.status || ''
  const image = c.image || ''
  return [image, status].filter(Boolean).join(' · ')
}

function initials(name: string): string {
  return (name || '?').slice(0, 1).toUpperCase()
}

function relativeTime(value: string): string {
  if (!value) return ''
  const t = Date.parse(value)
  if (Number.isNaN(t)) return value
  const diff = Date.now() - t
  const sec = Math.round(diff / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.round(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.round(min / 60)
  if (hr < 24) return `${hr}h ago`
  const d = Math.round(hr / 24)
  return `${d}d ago`
}

async function executeCommand(cmd: MakeCommand) {
  if (!currentProject.value) return
  loadingCmd.value = cmd.name
  try {
    await projectsApi.exec(currentProject.value.name, cmd.name.replace(/^make\s+/, ''))
    toast.show('success', `Running: ${cmd.name}`)
  } catch (e) {
    toast.show('error', (e as Error)?.message || 'Command failed')
  } finally {
    loadingCmd.value = null
  }
}

async function gitPull() {
  try {
    await gitStore.pull()
  } catch (e) {
    toast.show('error', (e as Error)?.message || 'Pull failed')
  }
}

async function gitPush() {
  try {
    await gitStore.push()
  } catch (e) {
    toast.show('error', (e as Error)?.message || 'Push failed')
  }
}

function greet(): string {
  const h = new Date().getHours()
  if (h < 5) return 'Working late'
  if (h < 12) return 'Good morning'
  if (h < 18) return 'Good afternoon'
  return 'Good evening'
}

const metaKbd = typeof navigator !== 'undefined' && /Mac/i.test(navigator.platform || '') ? '⌘K' : 'Ctrl+K'

function onCmdbarEnter() {
  if (!query.value.trim()) return
  router.push('/commands')
}
</script>

<template>
  <div class="dashboard-view">
    <!-- ============== EMPTY (no project) ============== -->
    <template v-if="!hasProject">
      <header class="page-head">
        <div>
          <h1>Welcome to DevHub</h1>
          <p class="sub">Point DevHub at a folder to get git status, containers, GitLab issues, and a terminal in one place.</p>
        </div>
      </header>
      <div class="card empty-card">
        <div class="empty-glyph">
          <span class="logo-mark"></span>
        </div>
        <div class="empty-body">
          <h4>No project opened yet</h4>
          <p>Select a project from the sidebar to begin. DevHub will detect git, Docker Compose, and Makefile targets automatically.</p>
        </div>
        <div class="empty-actions">
          <button class="btn primary lg" @click="router.push('/settings')">Open settings</button>
          <button class="btn lg" @click="refresh">Reload</button>
        </div>
        <div v-if="projectsStore.projects.length" class="empty-projects">
          <div class="empty-projects-label">Recent projects</div>
          <div
            v-for="p in projectsStore.projects.slice(0, 4)"
            :key="p.name"
            class="empty-project"
            @click="projectsStore.setCurrentProject(p.name)"
          >
            <span class="proj-dot"></span>
            <div class="empty-project-body">
              <div class="empty-project-name">{{ p.name }}</div>
              <div class="empty-project-path">{{ p.path }}</div>
            </div>
            <button class="btn sm">open</button>
          </div>
        </div>
      </div>
      <div class="toast-hint">Tip: run <span class="kbd">{{ metaKbd }}</span> to open the command palette.</div>
    </template>

    <!-- ============== ERROR banner (non-blocking) ============== -->
    <div v-else-if="lastError" class="error-banner" role="alert">
      <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor"><path d="M8 1l7 13H1L8 1zm0 5v4m0 2v1" stroke="var(--bg-0)" stroke-width="1.5"/></svg>
      <div style="flex:1">
        <div class="bd">BACKEND UNREACHABLE</div>
        <div>{{ lastError }}</div>
      </div>
      <button class="btn" @click="refresh">Retry</button>
    </div>

    <!-- ============== DEFAULT + LOADING ============== -->
    <template v-if="hasProject">
      <header class="page-head">
        <div>
          <h1>Dashboard</h1>
          <p class="sub">
            At-a-glance status of your local project — git, containers, remote issues, recent activity.
          </p>
        </div>
        <span class="chip mute">route: /</span>
      </header>

      <div class="cmdbar">
        <svg class="icon" width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="7" cy="7" r="5"/><path d="M14 14l-3.2-3.2"/>
        </svg>
        <input
          v-model="query"
          placeholder="Search commits, files, branches, commands…  (try: checkout main, make build)"
          @keydown.enter="onCmdbarEnter"
        />
        <span class="chip mute">branch</span>
        <span class="chip mute">file</span>
        <span class="chip mute">cmd</span>
        <span class="kbd">{{ metaKbd }}</span>
      </div>

      <div class="greet-row">
        <div>
          <div class="breadcrumbs">
            <b>{{ currentProject?.path }}</b>
            <template v-if="gitStore.status.branch">
              · branch <b style="color: var(--accent)">{{ gitStore.status.branch }}</b>
            </template>
            <template v-if="gitSub"> · {{ gitSub }}</template>
          </div>
          <h3 class="title">{{ greet() }}.</h3>
        </div>
        <div class="head-actions">
          <button class="btn" :disabled="refreshing" @click="refresh">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M13.5 8a5.5 5.5 0 01-9.4 3.9M2.5 8a5.5 5.5 0 019.4-3.9M14 2v4h-4M2 14v-4h4"/></svg>
            Refresh
          </button>
          <button class="btn" :disabled="gitStore.loading.pull || gitStore.loading.push" @click="gitPull">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 3v10M3 8l5 5 5-5"/></svg>
            Pull
          </button>
          <button class="btn primary" :disabled="gitStore.loading.push" @click="gitPush">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 13V3M3 8l5-5 5 5"/></svg>
            Push
          </button>
        </div>
      </div>

      <!-- KPIs -->
      <div class="kpis">
        <template v-if="isLoading && !gitStore.status.branch">
          <div v-for="i in 4" :key="i" class="kpi">
            <div class="stripe" style="background: var(--line)"></div>
            <div class="ds-skeleton" style="height: 10px; width: 40%"></div>
            <div class="ds-skeleton" style="height: 20px; width: 70%; margin-top: 8px"></div>
            <div class="ds-skeleton" style="height: 10px; width: 50%; margin-top: 6px"></div>
          </div>
        </template>
        <template v-else>
          <div class="kpi" :class="gitKpiTone">
            <div class="stripe"></div>
            <div class="lbl"><span class="kpi-dot"></span> Git</div>
            <div class="val">{{ gitStore.status.branch || '—' }}</div>
            <div class="sub">{{ gitSub }}</div>
          </div>
          <div class="kpi" :class="dockerKpiTone">
            <div class="stripe"></div>
            <div class="lbl"><span class="kpi-dot"></span> Docker</div>
            <div class="val">{{ dockerRunning }} / {{ dockerTotal }}</div>
            <div class="sub">{{ dockerKpiSub }}</div>
          </div>
          <div class="kpi info">
            <div class="stripe"></div>
            <div class="lbl"><span class="kpi-dot"></span> GitLab</div>
            <div class="val">{{ gitlabKpiVal }}</div>
            <div class="sub">{{ gitlabKpiSub }}</div>
          </div>
          <div class="kpi">
            <div class="stripe"></div>
            <div class="lbl"><span class="kpi-dot"></span> Activity</div>
            <div class="val">{{ commitsThisWeek }}</div>
            <div class="sub">commits this week</div>
            <div class="spark">
              <span
                v-for="(b, i) in sparkBars"
                :key="i"
                :style="{ height: Math.max(8, b.h) + '%', background: b.active ? 'var(--accent)' : undefined }"
              ></span>
            </div>
          </div>
        </template>
      </div>

      <!-- Two-col: commits / (containers + quick actions) -->
      <div class="cols">
        <div class="card">
          <header>
            Recent commits
            <span class="count">{{ gitStore.status.branch || 'main' }} · last {{ recentCommits.length }}</span>
            <div class="act">
              <button class="btn ghost" @click="router.push('/git')">View log →</button>
            </div>
          </header>
          <div v-if="isLoading && !recentCommits.length" class="skeleton-rows">
            <div v-for="i in 5" :key="i" class="ds-skeleton" :style="{ height: '14px', width: (50 + ((i * 13) % 40)) + '%' }"></div>
          </div>
          <div v-else-if="!recentCommits.length" class="empty">
            <div class="glyph">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" width="22" height="22">
                <circle cx="8" cy="8" r="3"/><path d="M3 8H1M15 8h-2M8 3V1M8 15v-2"/>
              </svg>
            </div>
            <h4>No commits yet</h4>
            <p>Make a first commit to see history here.</p>
          </div>
          <div v-else class="commits">
            <div
              v-for="commit in recentCommits"
              :key="commit.hash"
              class="commit"
              :class="{ merge: (commit.parents?.length || 0) > 1 }"
              @click="router.push('/git')"
            >
              <div class="track"></div>
              <div>
                <div class="msg">{{ commit.message }}</div>
                <div class="meta-line">
                  <span class="hash">#{{ commit.short_hash }}</span>
                  <span class="author">
                    <span class="av">{{ initials(commit.author) }}</span>
                    {{ commit.author }}
                  </span>
                  <span v-if="(commit.parents?.length || 0) > 1" class="chip info">merge</span>
                </div>
              </div>
              <div class="when">{{ relativeTime(commit.date) }}</div>
            </div>
          </div>
        </div>

        <div class="right-col">
          <div class="card">
            <header>
              Containers
              <span class="count">{{ dockerRunning }} up · {{ dockerTotal }} total</span>
              <div class="act">
                <button class="btn ghost" @click="router.push('/docker')">Open →</button>
              </div>
            </header>
            <div v-if="isLoading && !containers.length" class="skeleton-rows">
              <div v-for="i in 3" :key="i" class="ds-skeleton" :style="{ height: '14px', width: (60 + ((i * 13) % 30)) + '%' }"></div>
            </div>
            <div v-else-if="!containers.length" class="empty">
              <div class="glyph">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" width="22" height="22">
                  <rect x="2" y="4" width="12" height="9" rx="1"/><path d="M2 7h12"/>
                </svg>
              </div>
              <h4>No containers</h4>
              <p>Run <span class="kbd">docker compose up</span> or start from the Docker tab.</p>
            </div>
            <div v-else class="ctn-list">
              <div
                v-for="c in containers"
                :key="c.name"
                class="ctn-row"
                @click="router.push('/docker')"
              >
                <span class="state" :class="containerTone(c.state)"></span>
                <div class="ctn-body">
                  <div class="ctn-name">{{ c.name }}</div>
                  <div class="ctn-sub">{{ containerSub(c) }}</div>
                </div>
                <span v-if="c.ports" class="port-chip">{{ c.ports }}</span>
                <button class="btn ghost sm" @click.stop="router.push('/docker')">open</button>
              </div>
            </div>
          </div>

          <div v-if="currentProject?.has_makefile" class="card">
            <header>
              Quick actions
              <span class="count">Makefile · {{ commands.length }} target{{ commands.length === 1 ? '' : 's' }}</span>
            </header>
            <div v-if="commandsLoading" class="skeleton-rows" style="padding: 14px">
              <div v-for="i in 3" :key="i" class="ds-skeleton" style="height: 42px"></div>
            </div>
            <div v-else-if="!commands.length" class="empty">
              <div class="glyph">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" width="22" height="22">
                  <rect x="2" y="3" width="12" height="10" rx="1"/><path d="M5 6l2 2-2 2M9 10h3"/>
                </svg>
              </div>
              <h4>No Makefile targets</h4>
              <p>Add targets to your Makefile to run them from here.</p>
            </div>
            <div v-else class="qa">
              <button
                v-for="action in commands"
                :key="action.name"
                :disabled="loadingCmd === action.name"
                @click="executeCommand(action)"
              >
                <span class="cat">{{ action.category || 'task' }}</span>
                <span class="cmd">{{ action.name }}</span>
                <span class="desc">{{ action.description || '—' }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: var(--s5);
  width: 100%;
  height: 100%;
  min-height: 0;
}

.page-head {
  margin-bottom: 0;
  padding-bottom: var(--s4);
}

.head-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.greet-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--s4);
  flex-wrap: wrap;
}
.greet-row .title { margin-top: 4px; }

.kpi-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  color: var(--fg-3);
  margin-right: 2px;
  transform: translateY(-1px);
}
.kpi.ok .kpi-dot { color: var(--ok); box-shadow: 0 0 6px var(--ok); }
.kpi.warn .kpi-dot { color: var(--warn); box-shadow: 0 0 6px var(--warn); }
.kpi.bad .kpi-dot { color: var(--bad); box-shadow: 0 0 6px var(--bad); }
.kpi.info .kpi-dot { color: var(--info); box-shadow: 0 0 6px var(--info); }
.kpi:not(.ok):not(.warn):not(.bad):not(.info) .kpi-dot { color: var(--accent); box-shadow: 0 0 6px var(--accent); }

.cols {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr);
  gap: var(--s4);
  flex: 1;
  min-height: 0;
}
@media (max-width: 1100px) {
  .cols { grid-template-columns: 1fr; }
}

.right-col {
  display: flex;
  flex-direction: column;
  gap: var(--s4);
  min-width: 0;
  min-height: 0;
}

/* Cards in cols/right-col flex to share height; header stays, body scrolls */
.cols > .card,
.right-col > .card {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.cols > .card > :deep(header),
.right-col > .card > :deep(header) {
  flex-shrink: 0;
}
.right-col > .card {
  flex: 1 1 0;
  min-height: 180px;
}

/* Recent commits list — ported from prototype */
.commits {
  padding: 4px 0;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-gutter: stable;
}
.commit {
  display: grid;
  grid-template-columns: 20px 1fr auto;
  gap: 10px;
  align-items: start;
  padding: 10px 14px;
  border-bottom: 1px solid var(--line-soft);
  cursor: pointer;
  transition: background var(--t-fast);
}
.commit:last-child { border-bottom: 0; }
.commit:hover { background: var(--bg-2); }
.commit .track { position: relative; height: 100%; }
.commit .track::before {
  content: ""; position: absolute; left: 9px; top: 0; bottom: -14px;
  width: 2px; background: var(--line);
}
.commit:last-child .track::before { bottom: 50%; }
.commit .track::after {
  content: ""; position: absolute; left: 4px; top: 6px;
  width: 12px; height: 12px; border-radius: 50%;
  background: var(--bg-1); border: 2px solid var(--accent);
}
.commit.merge .track::after { border-color: var(--info); }
.commit .msg { font-size: 13.5px; color: var(--fg); font-weight: 500; }
.commit .meta-line { display: flex; gap: 10px; margin-top: 3px; color: var(--fg-3); font-size: 12px; font-family: var(--mono); align-items: center; flex-wrap: wrap; }
.commit .hash { color: var(--accent); }
.commit .author { display: flex; align-items: center; gap: 6px; }
.commit .author .av {
  width: 16px; height: 16px; border-radius: 50%;
  background: linear-gradient(135deg, var(--info), var(--accent));
  font-size: 9px; display: flex; align-items: center; justify-content: center;
  color: var(--accent-ink); font-weight: 700;
}
.commit .when { color: var(--fg-3); font-size: 12px; font-family: var(--mono); white-space: nowrap; }

/* Container rows */
.ctn-list {
  padding: 4px 0;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-gutter: stable;
}
.ctn-row {
  display: grid;
  grid-template-columns: 10px 1fr auto auto;
  gap: 12px;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--line-soft);
  cursor: pointer;
}
.ctn-row:last-child { border-bottom: 0; }
.ctn-row:hover { background: var(--bg-2); }
.state { width: 8px; height: 8px; border-radius: 50%; }
.state.up { background: var(--ok); box-shadow: 0 0 6px var(--ok); }
.state.down { background: var(--bad); }
.state.restart { background: var(--warn); box-shadow: 0 0 6px var(--warn); }
.ctn-body { min-width: 0; }
.ctn-name { font-family: var(--mono); font-size: 13px; color: var(--fg); }
.ctn-sub { font-size: 11.5px; color: var(--fg-3); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.port-chip { font-family: var(--mono); font-size: 11px; color: var(--fg-2); background: var(--bg-2); border: 1px solid var(--line); padding: 2px 8px; border-radius: var(--r-pill); white-space: nowrap; }

/* Quick actions grid */
.qa {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 10px;
  padding: 14px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-gutter: stable;
}
.qa button {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 12px 14px;
  background: var(--bg-2);
  border: 1px solid var(--line);
  border-radius: var(--r2);
  cursor: pointer;
  text-align: left;
  transition: border-color var(--t-fast), transform var(--t-fast);
  color: var(--fg);
  font-family: var(--ui);
}
.qa button:hover:not(:disabled) { border-color: var(--accent); transform: translateY(-1px); }
.qa button:disabled { opacity: 0.6; cursor: progress; }
.qa .cmd { font-family: var(--mono); font-size: 12.5px; color: var(--fg); font-weight: 600; }
.qa .desc { font-size: 11.5px; color: var(--fg-3); }
.qa .cat { font-size: 10px; color: var(--accent); text-transform: uppercase; letter-spacing: 0.1em; margin-bottom: 6px; }

.skeleton-rows { padding: 14px; display: flex; flex-direction: column; gap: 14px; }

/* Empty state card */
.empty-card {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}
.empty-glyph {
  width: 80px; height: 80px; border-radius: 24px;
  background: var(--bg-2); border: 1px solid var(--line);
  display: flex; align-items: center; justify-content: center;
}
.empty-glyph .logo-mark { width: 44px; height: 44px; }
.empty-body { max-width: 440px; }
.empty-body h4 { margin: 0; font-size: 18px; color: var(--fg); font-weight: 600; }
.empty-body p { color: var(--fg-3); font-size: 14px; margin-top: 8px; }
.empty-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.empty-projects {
  margin-top: 12px;
  width: 100%;
  max-width: 520px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.empty-projects-label {
  font-size: 11px; font-weight: 600; color: var(--fg-3);
  text-transform: uppercase; letter-spacing: 0.1em;
  margin-top: 16px;
  text-align: left;
}
.empty-project {
  display: flex; align-items: center; gap: 12px;
  padding: 10px;
  border: 1px solid var(--line);
  border-radius: var(--r2);
  background: var(--bg-2);
  cursor: pointer;
  transition: border-color var(--t-fast);
}
.empty-project:hover { border-color: var(--accent); }
.empty-project-body { flex: 1; min-width: 0; text-align: left; }
.empty-project-name { font-weight: 600; color: var(--fg); }
.empty-project-path { font-family: var(--mono); font-size: 11.5px; color: var(--fg-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.toast-hint {
  font-size: 12px; color: var(--fg-3);
  font-family: var(--mono);
  text-align: center;
}
</style>
