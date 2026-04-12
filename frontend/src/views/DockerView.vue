<script setup lang="ts">
import { onUnmounted, ref, computed, watch, nextTick } from 'vue'
import { useDockerStore } from '../stores/docker'
import { useProjectsStore } from '../stores/projects'
import { api, postJson, projectUrl } from '../api/client'
import { terminalApi } from '../api/terminal'
import ShimmerBlock from '../components/ShimmerBlock.vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const dockerStore = useDockerStore()
const projectsStore = useProjectsStore()

// Docker exec terminal
const execContainer = ref<string | null>(null)
const execSessionId = ref<string | null>(null)
const execTermEl = ref<HTMLDivElement | null>(null)
let execTerm: Terminal | null = null
let execFitAddon: FitAddon | null = null
let execWs: WebSocket | null = null

async function openTerminal(containerName: string) {
  const projectName = projectsStore.currentProject?.name
  if (!projectName) return

  // Clean up previous terminal to prevent leak
  closeTerminal()

  try {
    const data = await api<{ session_id: string }>(
      `${projectUrl(projectName)}/docker/${containerName}/exec`,
      postJson({ cols: 80, rows: 24 }),
    )
    execContainer.value = containerName
    execSessionId.value = data.session_id
    // Mount terminal after DOM update
    await nextTick()
    mountExecTerminal(data.session_id)
  } catch (e) {
    console.error('Failed to exec into container:', e)
  }
}

function mountExecTerminal(sessionId: string) {
  if (!execTermEl.value) return

  execTerm = new Terminal({
    cursorBlink: true,
    fontFamily: 'monospace',
    fontSize: 13,
    lineHeight: 1.0,
    scrollback: 2000,
  })
  execFitAddon = new FitAddon()
  execTerm.loadAddon(execFitAddon)
  execTerm.open(execTermEl.value)
  execFitAddon.fit()

  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  execWs = new WebSocket(`${proto}//${host}/api/terminal/ws/${sessionId}`)
  execWs.binaryType = 'arraybuffer'

  execWs.onopen = () => {
    if (execTerm && execFitAddon) {
      execFitAddon.fit()
      execWs?.send(JSON.stringify({ type: 'resize', cols: execTerm.cols, rows: execTerm.rows }))
    }
  }

  execWs.onmessage = (event: MessageEvent) => {
    if (!execTerm) return
    if (event.data instanceof ArrayBuffer) {
      execTerm.write(new Uint8Array(event.data))
    }
  }

  const encoder = new TextEncoder()
  execTerm.onData((data: string) => {
    if (execWs?.readyState === WebSocket.OPEN) {
      execWs.send(encoder.encode(data))
    }
  })

  execTerm.onResize(({ cols, rows }) => {
    if (execWs?.readyState === WebSocket.OPEN) {
      execWs.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
  })
}

function closeTerminal() {
  if (execSessionId.value) {
    terminalApi.destroySession(execSessionId.value).catch(() => {})
  }
  if (execWs) {
    execWs.onclose = null
    execWs.close()
    execWs = null
  }
  execTerm?.dispose()
  execTerm = null
  execFitAddon = null
  execContainer.value = null
  execSessionId.value = null
}

// Logs state
const logLines = ref<string[]>([])
const logsRef = ref<HTMLElement | null>(null)
let eventSource: EventSource | null = null

// Auto-scroll logs to bottom
function scrollToBottom() {
  nextTick(() => {
    if (logsRef.value) {
      logsRef.value.scrollTop = logsRef.value.scrollHeight
    }
  })
}

// Connect SSE for selected container
function connectLogs(name: string) {
  disconnectLogs()
  logLines.value = []

  const url = dockerStore.logsUrl(name)
  eventSource = new EventSource(url)

  eventSource.onmessage = (event) => {
    logLines.value.push(event.data)
    // Keep buffer at 5000 lines max
    if (logLines.value.length > 5000) {
      logLines.value = logLines.value.slice(-4000)
    }
    scrollToBottom()
  }

  eventSource.onerror = () => {
    // EventSource will auto-reconnect; if we want to stop, disconnect
    if (eventSource && eventSource.readyState === EventSource.CLOSED) {
      disconnectLogs()
    }
  }
}

function disconnectLogs() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

function closeLogs() {
  disconnectLogs()
  dockerStore.selectContainer(null)
  logLines.value = []
}

function clearLogs() {
  logLines.value = []
}

function selectRow(name: string) {
  if (dockerStore.selectedContainer === name) {
    closeLogs()
  } else {
    dockerStore.selectContainer(name)
  }
}

// Watch selectedContainer to connect/disconnect SSE
watch(
  () => dockerStore.selectedContainer,
  (name) => {
    if (name) {
      connectLogs(name)
    } else {
      disconnectLogs()
    }
  }
)

async function startAll() {
  await dockerStore.containerAction('_', 'start-all')
}

async function stopAll() {
  await dockerStore.containerAction('_', 'stop-all')
}

function maskEnvValue(env: string): string {
  const eqIdx = env.indexOf('=')
  if (eqIdx === -1) return env
  const key = env.substring(0, eqIdx)
  const val = env.substring(eqIdx + 1)
  const secretPatterns = ['PASSWORD', 'SECRET', 'TOKEN', 'KEY', 'PRIVATE', 'CREDENTIAL']
  const upper = key.toUpperCase()
  if (secretPatterns.some((p) => upper.includes(p))) {
    return `${key}=${'***'}`
  }
  return `${key}=${val}`
}

function formatDate(dateStr: string): string {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '-'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

function stateClass(state: string): string {
  switch (state) {
    case 'running':
      return 'dot-running'
    case 'restarting':
      return 'dot-restarting'
    default:
      return 'dot-exited'
  }
}

const hasDocker = computed(() => projectsStore.currentProject?.has_docker ?? false)

function cpuClass(cpuPerc: string): string {
  const val = parseFloat(cpuPerc)
  if (isNaN(val)) return ''
  if (val >= 80) return 'cpu-high'
  if (val >= 50) return 'cpu-medium'
  return 'cpu-low'
}

// Refetch when project changes (component survives route transitions)
watch(
  () => projectsStore.currentProject?.name,
  () => {
    closeLogs()
    closeTerminal()
    dockerStore.stopStatsPolling()
    if (hasDocker.value) {
      dockerStore.fetchContainers()
      dockerStore.startStatsPolling()
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  disconnectLogs()
  closeTerminal()
  dockerStore.stopStatsPolling()
})
</script>

<template>
  <div class="docker-view">
    <!-- Header — always visible -->
    <header class="page-header">
      <div class="header-row">
        <div class="header-title">
          <h1>Docker</h1>
          <span class="header-count" v-if="dockerStore.totalCount > 0">
            {{ dockerStore.runningCount }} running / {{ dockerStore.totalCount }} total
          </span>
        </div>
        <div v-if="hasDocker" class="header-actions">
          <button class="btn btn-green" @click="startAll" :disabled="dockerStore.loading">
            Start All
          </button>
          <button class="btn btn-red" @click="stopAll" :disabled="dockerStore.loading">
            Stop All
          </button>
          <button class="btn" @click="dockerStore.fetchContainers()" :disabled="dockerStore.loading">
            Refresh
          </button>
        </div>
      </div>
    </header>

    <!-- No Docker -->
    <div v-if="!hasDocker" class="no-docker">
      <div class="no-docker-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
          <line x1="12" y1="22.08" x2="12" y2="12"/>
        </svg>
      </div>
      <h2>Docker not available</h2>
      <p>This project does not have a <code>docker-compose.yml</code> file.</p>
      <p class="no-docker-hint">Add a docker-compose.yml to the project root to manage containers here.</p>
    </div>

    <template v-else>
    <!-- Docker Compose section -->
    <section class="compose-section">
      <div class="compose-header">
        <span class="compose-title">Docker Compose</span>
        <span class="compose-file">docker-compose.yml</span>
      </div>
      <div class="compose-buttons">
        <button
          class="compose-btn compose-btn-green"
          :disabled="dockerStore.composeLoading !== null"
          @click="dockerStore.composeUp()"
        >
          <svg v-if="dockerStore.composeLoading === 'up'" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="currentColor">
            <polygon points="5,3 19,12 5,21"/>
          </svg>
          Up
        </button>
        <button
          class="compose-btn compose-btn-blue"
          :disabled="dockerStore.composeLoading !== null"
          @click="dockerStore.composeUpBuild()"
        >
          <svg v-if="dockerStore.composeLoading === 'rebuild'" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10"/>
            <path d="M20.49 15a9 9 0 1 1-.07-5.05"/>
          </svg>
          Rebuild
        </button>
        <button
          class="compose-btn compose-btn-red"
          :disabled="dockerStore.composeLoading !== null"
          @click="dockerStore.composeDown()"
        >
          <svg v-if="dockerStore.composeLoading === 'down'" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="currentColor">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
          </svg>
          Down
        </button>
      </div>
    </section>

    <!-- Container table -->
    <section class="section">
      <div v-if="dockerStore.loading && !dockerStore.containers?.length" class="shimmer-rows">
        <ShimmerBlock variant="row" :lines="4" />
      </div>
      <div v-else-if="!dockerStore.containers?.length" class="empty">
        No containers found
      </div>
      <table v-else class="containers-table">
        <thead>
          <tr>
            <th class="col-expand"></th>
            <th class="col-status">Status</th>
            <th class="col-name">Name</th>
            <th class="col-image">Image</th>
            <th class="col-cpu">CPU%</th>
            <th class="col-mem">MEM</th>
            <th class="col-ports">Ports</th>
            <th class="col-state">State</th>
            <th class="col-actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="c in dockerStore.containers" :key="c.name">
          <tr
            :class="{ 'row-active': dockerStore.selectedContainer === c.name }"
            @click="selectRow(c.name)"
          >
            <td class="cell-expand" @click.stop>
              <button
                class="expand-btn"
                :class="{ 'expand-btn-open': dockerStore.expandedContainer === c.name }"
                @click="dockerStore.toggleInspect(c.name)"
              >
                <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                  <path d="M8 5l7 7-7 7z"/>
                </svg>
              </button>
            </td>
            <td class="cell-status">
              <span class="status-dot" :class="stateClass(c.state)"></span>
            </td>
            <td class="cell-name">{{ c.name }}</td>
            <td class="cell-image">{{ c.image }}</td>
            <td class="cell-cpu">
              <span
                v-if="c.state === 'running' && dockerStore.statsForContainer(c.name)"
                class="cpu-value"
                :class="cpuClass(dockerStore.statsForContainer(c.name)!.cpu_perc)"
              >{{ dockerStore.statsForContainer(c.name)!.cpu_perc }}</span>
              <span v-else class="stat-na">-</span>
            </td>
            <td class="cell-mem">
              <span v-if="c.state === 'running' && dockerStore.statsForContainer(c.name)" class="mem-value">
                {{ dockerStore.statsForContainer(c.name)!.mem_usage }}
              </span>
              <span v-else class="stat-na">-</span>
            </td>
            <td class="cell-ports">{{ c.ports || '-' }}</td>
            <td class="cell-state">
              <span class="state-badge" :class="'state-' + c.state">{{ c.state }}</span>
            </td>
            <td class="cell-actions" @click.stop>
              <button
                v-if="c.state !== 'running'"
                class="action-btn action-btn-start"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'start')"
              >
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <polygon points="5,3 19,12 5,21"/>
                </svg>
                Start
              </button>
              <button
                v-if="c.state === 'running'"
                class="action-btn action-btn-stop"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'stop')"
              >
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <rect x="3" y="3" width="18" height="18" rx="2"/>
                </svg>
                Stop
              </button>
              <button
                v-if="c.state === 'running'"
                class="action-btn action-btn-restart"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'restart')"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="23 4 23 10 17 10"/>
                  <path d="M20.49 15a9 9 0 1 1-.07-5.05"/>
                </svg>
                Restart
              </button>
              <button
                v-if="c.state === 'running'"
                class="action-btn action-btn-terminal"
                @click="openTerminal(c.name)"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="4 17 10 11 4 5"/>
                  <line x1="12" y1="19" x2="20" y2="19"/>
                </svg>
                Terminal
              </button>
              <button
                class="action-btn action-btn-logs"
                @click="selectRow(c.name)"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                  <line x1="16" y1="13" x2="8" y2="13"/>
                  <line x1="16" y1="17" x2="8" y2="17"/>
                  <polyline points="10 9 9 9 8 9"/>
                </svg>
                Logs
              </button>
            </td>
          </tr>
          <!-- Expandable inspect row -->
          <tr v-if="dockerStore.expandedContainer === c.name" class="inspect-row">
            <td :colspan="9">
              <div class="inspect-panel">
                <div v-if="dockerStore.inspectLoading" class="inspect-loading">Loading...</div>
                <template v-else-if="dockerStore.inspectData">
                  <div class="inspect-grid">
                    <!-- Health & Status -->
                    <div class="inspect-section">
                      <h4 class="inspect-section-title">Health & Status</h4>
                      <div class="inspect-kv">
                        <span class="inspect-key">Health</span>
                        <span class="inspect-value">
                          <span class="health-badge" :class="'health-' + dockerStore.inspectData.health">
                            {{ dockerStore.inspectData.health }}
                          </span>
                        </span>
                      </div>
                      <div class="inspect-kv">
                        <span class="inspect-key">Restart Count</span>
                        <span class="inspect-value">{{ dockerStore.inspectData.restart_count }}</span>
                      </div>
                      <div class="inspect-kv">
                        <span class="inspect-key">Created</span>
                        <span class="inspect-value">{{ formatDate(dockerStore.inspectData.created) }}</span>
                      </div>
                      <div class="inspect-kv">
                        <span class="inspect-key">Started</span>
                        <span class="inspect-value">{{ formatDate(dockerStore.inspectData.started_at) }}</span>
                      </div>
                    </div>

                    <!-- Networks -->
                    <div class="inspect-section">
                      <h4 class="inspect-section-title">Networks</h4>
                      <div v-if="dockerStore.inspectData.networks.length === 0" class="inspect-empty">No networks</div>
                      <div v-else>
                        <div v-for="net in dockerStore.inspectData.networks" :key="net" class="inspect-item">
                          {{ net }}
                        </div>
                        <div v-if="dockerStore.inspectData.ip_address" class="inspect-kv">
                          <span class="inspect-key">IP</span>
                          <span class="inspect-value mono">{{ dockerStore.inspectData.ip_address }}</span>
                        </div>
                      </div>
                    </div>

                    <!-- Command -->
                    <div class="inspect-section">
                      <h4 class="inspect-section-title">Command</h4>
                      <div v-if="dockerStore.inspectData.cmd.length === 0" class="inspect-empty">No command</div>
                      <code v-else class="inspect-cmd">{{ dockerStore.inspectData.cmd.join(' ') }}</code>
                    </div>

                    <!-- Ports -->
                    <div class="inspect-section">
                      <h4 class="inspect-section-title">Ports</h4>
                      <div v-if="dockerStore.inspectData.ports.length === 0" class="inspect-empty">No ports</div>
                      <div v-else class="inspect-list">
                        <div v-for="(p, i) in dockerStore.inspectData.ports" :key="i" class="inspect-item">
                          <span class="mono">{{ p.host_port || '*' }} &rarr; {{ p.container_port }}/{{ p.protocol }}</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Mounts -->
                  <div v-if="dockerStore.inspectData.mounts.length > 0" class="inspect-section inspect-section-full">
                    <h4 class="inspect-section-title">Mounts</h4>
                    <div class="inspect-list">
                      <div v-for="(m, i) in dockerStore.inspectData.mounts" :key="i" class="inspect-mount">
                        <span class="mono">{{ m.source }}</span>
                        <span class="inspect-arrow">&rarr;</span>
                        <span class="mono">{{ m.destination }}</span>
                        <span class="inspect-tag">{{ m.type }}</span>
                        <span v-if="m.mode" class="inspect-tag">{{ m.mode }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Environment -->
                  <div class="inspect-section inspect-section-full">
                    <h4 class="inspect-section-title">Environment</h4>
                    <div v-if="dockerStore.inspectData.env.length === 0" class="inspect-empty">No environment variables</div>
                    <div v-else class="inspect-env-list">
                      <div v-for="(e, i) in dockerStore.inspectData.env" :key="i" class="inspect-env-item mono">
                        {{ maskEnvValue(e) }}
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </td>
          </tr>
          </template>
        </tbody>
      </table>
    </section>

    <!-- Logs panel -->
    <section v-if="dockerStore.selectedContainer" class="logs-section">
      <div class="logs-header">
        <h2>
          <span class="logs-icon">></span>
          {{ dockerStore.selectedContainer }}
        </h2>
        <div class="logs-actions">
          <button class="btn btn-sm" @click="clearLogs">Clear</button>
          <button class="btn btn-sm" @click="closeLogs">Close</button>
        </div>
      </div>
      <div class="logs-container" ref="logsRef">
        <div v-if="logLines.length === 0" class="logs-empty">
          Waiting for log output...
        </div>
        <pre v-else class="logs-content"><span
          v-for="(line, i) in logLines"
          :key="i"
          class="log-line"
        >{{ line }}
</span></pre>
      </div>
    </section>

    <!-- Docker exec terminal modal -->
    <Teleport to="body">
      <div v-if="execSessionId" class="docker-term-overlay" @click.self="closeTerminal">
        <div class="docker-term-modal">
          <div class="docker-term-header">
            <span class="docker-term-title">Terminal: {{ execContainer }}</span>
            <button class="docker-term-close" @click="closeTerminal">&times;</button>
          </div>
          <div class="docker-term-body">
            <div ref="execTermEl" class="exec-terminal"></div>
          </div>
        </div>
      </div>
    </Teleport>
    </template>
  </div>
</template>

<style scoped>
.docker-view {
}

/* No Docker state */
.no-docker {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 24px;
  text-align: center;
  color: var(--text-secondary);
}

.no-docker-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 20px;
  opacity: 0.3;
}

.no-docker-icon svg {
  width: 100%;
  height: 100%;
}

.no-docker h2 {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.no-docker p {
  font-size: 14px;
  margin-bottom: 4px;
}

.no-docker code {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--bg-tertiary);
  padding: 2px 6px;
  border-radius: 4px;
}

.no-docker-hint {
  margin-top: 12px;
  font-size: 13px;
  opacity: 0.7;
}

/* Header */
.page-header h1 {
  font-size: 28px;
  font-weight: 700;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.header-title {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.header-count {
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 400;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.btn {
  padding: 6px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  transition: background 0.15s;
}

.btn:hover:not(:disabled) {
  background: var(--border);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-sm {
  padding: 3px 12px;
  font-size: 12px;
}

.btn-green {
  background: rgba(63, 185, 80, 0.15);
  border-color: var(--accent-green);
  color: var(--accent-green);
}

.btn-green:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.25);
}

.btn-red {
  background: rgba(248, 81, 73, 0.15);
  border-color: var(--accent-red);
  color: var(--accent-red);
}

.btn-red:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.25);
}

/* Compose section */
.compose-section {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 16px;
  margin-bottom: 24px;
  background: var(--bg-secondary);
}

.compose-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.compose-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.compose-file {
  font-size: 12px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 1px 8px;
  border-radius: 4px;
  border: 1px solid var(--border);
}

.compose-buttons {
  display: flex;
  gap: 8px;
}

.compose-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border-radius: 6px;
  border: 1px solid;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.compose-btn svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.compose-btn-green {
  background: rgba(63, 185, 80, 0.1);
  color: var(--accent-green);
  border-color: rgba(63, 185, 80, 0.3);
}

.compose-btn-green:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.2);
  border-color: var(--accent-green);
}

.compose-btn-blue {
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  border-color: rgba(88, 166, 255, 0.3);
}

.compose-btn-blue:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.2);
  border-color: var(--accent-blue);
}

.compose-btn-red {
  background: rgba(248, 81, 73, 0.1);
  color: var(--accent-red);
  border-color: rgba(248, 81, 73, 0.3);
}

.compose-btn-red:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.2);
  border-color: var(--accent-red);
}

.compose-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.spin-icon {
  animation: spin 0.8s linear infinite;
}

/* Table */
.section {
  margin-bottom: 24px;
}

.empty {
  color: var(--text-secondary);
  font-size: 14px;
  padding: 24px;
  background: var(--bg-secondary);
  border-radius: 8px;
  text-align: center;
}

.containers-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.containers-table th {
  text-align: left;
  padding: 10px 12px;
  color: var(--text-secondary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
}

.containers-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

.containers-table tbody tr {
  cursor: pointer;
  transition: background 0.1s;
}

.containers-table tbody tr:hover td {
  background: var(--bg-secondary);
}

.containers-table tbody tr.row-active td {
  background: rgba(88, 166, 255, 0.08);
  border-color: rgba(88, 166, 255, 0.2);
}

.col-expand {
  width: 36px;
}

.col-status {
  width: 50px;
}

.col-cpu {
  width: 80px;
}

.col-mem {
  width: 160px;
}

.col-actions {
  width: 280px;
}

.cell-status {
  text-align: center;
}

.status-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot-running {
  background: var(--accent-green);
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
  animation: pulse-green 2s ease-in-out infinite;
}

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 6px rgba(63, 185, 80, 0.5); }
  50% { box-shadow: 0 0 12px rgba(63, 185, 80, 0.8); }
}

.dot-exited {
  background: var(--accent-red);
}

.dot-restarting {
  background: var(--accent-orange);
  box-shadow: 0 0 6px rgba(210, 153, 34, 0.5);
  animation: pulse-orange 1.5s ease-in-out infinite;
}

@keyframes pulse-orange {
  0%, 100% { box-shadow: 0 0 6px rgba(210, 153, 34, 0.5); }
  50% { box-shadow: 0 0 12px rgba(210, 153, 34, 0.8); }
}

.cell-name {
  font-weight: 600;
  font-family: var(--font-mono);
  font-size: 13px;
}

.cell-image {
  color: var(--text-secondary);
  font-size: 13px;
  max-width: 250px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cell-cpu,
.cell-mem {
  font-family: var(--font-mono);
  font-size: 12px;
  white-space: nowrap;
}

.cpu-value {
  font-weight: 600;
}

.cpu-low {
  color: var(--accent-green);
}

.cpu-medium {
  color: var(--accent-orange);
}

.cpu-high {
  color: var(--accent-red);
}

.mem-value {
  color: var(--text-secondary);
}

.stat-na {
  color: var(--text-secondary);
  opacity: 0.5;
}

.cell-ports {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-blue);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.state-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.state-running {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.state-exited {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
}

.state-restarting {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.cell-actions {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

/* Action buttons */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 6px;
  border: 1px solid;
  background: none;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.15s;
}

.action-btn svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-btn-stop {
  color: var(--accent-red);
  border-color: rgba(248, 81, 73, 0.3);
}

.action-btn-stop:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.12);
}

.action-btn-start {
  color: var(--accent-green);
  border-color: rgba(63, 185, 80, 0.3);
}

.action-btn-start:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.12);
}

.action-btn-restart {
  color: var(--accent-blue);
  border-color: rgba(88, 166, 255, 0.3);
}

.action-btn-restart:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.12);
}

.action-btn-terminal {
  color: var(--accent-orange);
  border-color: rgba(210, 153, 34, 0.3);
}

.action-btn-terminal:hover:not(:disabled) {
  background: rgba(210, 153, 34, 0.12);
}

.action-btn-logs {
  color: var(--accent-purple);
  border-color: rgba(188, 140, 255, 0.3);
}

.action-btn-logs:hover:not(:disabled) {
  background: rgba(188, 140, 255, 0.12);
}

/* Docker exec terminal modal */
.docker-term-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.docker-term-modal {
  width: 80vw;
  height: 70vh;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 16px 48px rgba(0,0,0,0.5);
}

.docker-term-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.docker-term-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--accent-blue);
  font-family: var(--font-mono);
}

.docker-term-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 20px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.docker-term-close:hover {
  color: var(--accent-red);
}

.docker-term-body {
  flex: 1;
  min-height: 0;
  padding: 8px;
}

.exec-terminal {
  width: 100%;
  height: 100%;
}

.exec-terminal :deep(.xterm) {
  height: 100%;
  padding: 4px 4px 4px 8px;
}

.exec-terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
}

/* Logs panel */
.logs-section {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.logs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
}

.logs-header h2 {
  font-size: 14px;
  font-weight: 600;
  font-family: var(--font-mono);
  display: flex;
  align-items: center;
  gap: 8px;
}

.logs-icon {
  color: var(--accent-green);
  font-weight: 700;
}

.logs-actions {
  display: flex;
  gap: 6px;
}

.logs-container {
  background: var(--bg-primary);
  height: 400px;
  overflow-y: auto;
  overflow-x: hidden;
}

.logs-empty {
  padding: 24px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.logs-content {
  margin: 0;
  padding: 12px 16px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-primary);
}

.log-line {
  display: block;
}

.log-line:hover {
  background: rgba(88, 166, 255, 0.04);
}

/* Expand button */
.cell-expand {
  text-align: center;
  width: 36px;
  padding: 0 4px !important;
}

.expand-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
}

.expand-btn svg {
  width: 16px;
  height: 16px;
  transition: transform 0.2s ease;
}

.expand-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.expand-btn-open svg {
  transform: rotate(90deg);
}

/* Inspect panel */
.inspect-row td {
  padding: 0 !important;
  border-bottom: 1px solid var(--border);
}

.inspect-panel {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  padding: 16px 20px;
}

.inspect-loading {
  color: var(--text-secondary);
  font-size: 13px;
  padding: 12px 0;
}

.inspect-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
  margin-bottom: 16px;
}

.inspect-section {
  min-width: 0;
}

.inspect-section-full {
  margin-top: 4px;
}

.inspect-section-title {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  font-weight: 600;
}

.inspect-kv {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  margin-bottom: 4px;
}

.inspect-key {
  color: var(--text-secondary);
  white-space: nowrap;
}

.inspect-value {
  color: var(--text-primary);
}

.inspect-empty {
  color: var(--text-secondary);
  font-size: 12px;
  opacity: 0.6;
}

.inspect-item {
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 3px;
}

.inspect-cmd {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--bg-tertiary);
  padding: 6px 10px;
  border-radius: 4px;
  display: block;
  color: var(--text-primary);
  word-break: break-all;
}

.inspect-mount {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  margin-bottom: 4px;
  flex-wrap: wrap;
}

.inspect-arrow {
  color: var(--text-secondary);
}

.inspect-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 3px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.inspect-env-list {
  max-height: 200px;
  overflow-y: auto;
  background: var(--bg-tertiary);
  border-radius: 4px;
  padding: 8px 10px;
  border: 1px solid var(--border);
}

.inspect-env-item {
  font-size: 12px;
  color: var(--text-primary);
  line-height: 1.7;
  word-break: break-all;
}

.mono {
  font-family: var(--font-mono);
}

.health-badge {
  font-size: 12px;
  padding: 1px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.health-healthy {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.health-unhealthy {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
}

.health-starting {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.health-none {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}
</style>
