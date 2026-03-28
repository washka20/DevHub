<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import { useDockerStore } from '../stores/docker'
import { useProjectsStore } from '../stores/projects'
import WebTerminal from '../components/WebTerminal.vue'
import { useTerminalStore } from '../stores/terminal'

const dockerStore = useDockerStore()
const projectsStore = useProjectsStore()
const terminalStore = useTerminalStore()

// Docker exec terminal
const execContainer = ref<string | null>(null)
const execSessionId = ref<string | null>(null)

async function openTerminal(containerName: string) {
  const projectName = projectsStore.currentProject?.name
  if (!projectName) return

  try {
    const res = await fetch(`/api/projects/${projectName}/docker/${containerName}/exec`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cols: 80, rows: 24 }),
    })
    if (!res.ok) throw new Error(await res.text())
    const data = await res.json()
    execContainer.value = containerName
    execSessionId.value = data.session_id
  } catch (e) {
    console.error('Failed to exec into container:', e)
  }
}

function closeTerminal() {
  if (execSessionId.value) {
    fetch(`/api/terminal/sessions/${execSessionId.value}`, { method: 'DELETE' }).catch(() => {})
  }
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

onMounted(() => {
  dockerStore.fetchContainers()
})

onUnmounted(() => {
  disconnectLogs()
})
</script>

<template>
  <div class="docker-view">
    <!-- Header -->
    <header class="page-header">
      <div class="header-row">
        <div class="header-title">
          <h1>Containers</h1>
          <span class="header-count" v-if="dockerStore.totalCount > 0">
            {{ dockerStore.runningCount }} running / {{ dockerStore.totalCount }} total
          </span>
        </div>
        <div class="header-actions">
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

    <!-- Container table -->
    <section class="section">
      <div v-if="dockerStore.loading && !dockerStore.containers?.length" class="empty">
        Loading containers...
      </div>
      <div v-else-if="!dockerStore.containers?.length" class="empty">
        No containers found
      </div>
      <table v-else class="containers-table">
        <thead>
          <tr>
            <th class="col-status">Status</th>
            <th class="col-name">Name</th>
            <th class="col-image">Image</th>
            <th class="col-ports">Ports</th>
            <th class="col-state">State</th>
            <th class="col-actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="c in dockerStore.containers"
            :key="c.name"
            :class="{ 'row-active': dockerStore.selectedContainer === c.name }"
            @click="selectRow(c.name)"
          >
            <td class="cell-status">
              <span class="status-dot" :class="stateClass(c.state)"></span>
            </td>
            <td class="cell-name">{{ c.name }}</td>
            <td class="cell-image">{{ c.image }}</td>
            <td class="cell-ports">{{ c.ports || '-' }}</td>
            <td class="cell-state">
              <span class="state-badge" :class="'state-' + c.state">{{ c.state }}</span>
            </td>
            <td class="cell-actions" @click.stop>
              <button
                v-if="c.state !== 'running'"
                class="action-btn action-start"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'start')"
              >Start</button>
              <button
                v-if="c.state === 'running'"
                class="action-btn action-stop"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'stop')"
              >Stop</button>
              <button
                class="action-btn action-restart"
                :disabled="dockerStore.actionLoading === c.name"
                @click="dockerStore.containerAction(c.name, 'restart')"
              >Restart</button>
              <button
                v-if="c.state === 'running'"
                class="action-btn action-terminal"
                @click="openTerminal(c.name)"
              >Terminal</button>
            </td>
          </tr>
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
            <WebTerminal :session-id="execSessionId" />
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.docker-view {
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
  border-bottom: 1px solid #30363d;
}

.containers-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #30363d;
  vertical-align: middle;
}

.containers-table tbody tr {
  cursor: pointer;
  transition: background 0.1s;
}

.containers-table tbody tr:hover td {
  background: #161b22;
}

.containers-table tbody tr.row-active td {
  background: rgba(88, 166, 255, 0.08);
  border-color: rgba(88, 166, 255, 0.2);
}

.col-status {
  width: 50px;
}

.col-actions {
  width: 180px;
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
  background: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
  animation: pulse-green 2s ease-in-out infinite;
}

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 6px rgba(63, 185, 80, 0.5); }
  50% { box-shadow: 0 0 12px rgba(63, 185, 80, 0.8); }
}

.dot-exited {
  background: #f85149;
}

.dot-restarting {
  background: #d29922;
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
  color: #3fb950;
}

.state-exited {
  background: rgba(248, 81, 73, 0.15);
  color: #f85149;
}

.state-restarting {
  background: rgba(210, 153, 34, 0.15);
  color: #d29922;
}

.cell-actions {
  display: flex;
  gap: 6px;
}

.action-btn {
  padding: 3px 10px;
  font-size: 12px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  transition: color 0.15s, border-color 0.15s;
}

.action-btn:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--text-secondary);
}

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-start:hover:not(:disabled) {
  color: #3fb950;
  border-color: #3fb950;
}

.action-stop:hover:not(:disabled) {
  color: #f85149;
  border-color: #f85149;
}

.action-restart:hover:not(:disabled) {
  color: #d29922;
  border-color: #d29922;
}

.action-terminal:hover:not(:disabled) {
  color: #58a6ff;
  border-color: #58a6ff;
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
  background: #0d1117;
  border: 1px solid #30363d;
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
  background: #161b22;
  border-bottom: 1px solid #30363d;
  flex-shrink: 0;
}

.docker-term-title {
  font-size: 13px;
  font-weight: 600;
  color: #58a6ff;
  font-family: var(--font-mono);
}

.docker-term-close {
  background: none;
  border: none;
  color: #8b949e;
  font-size: 20px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.docker-term-close:hover {
  color: #f85149;
}

.docker-term-body {
  flex: 1;
  min-height: 0;
}

/* Logs panel */
.logs-section {
  border: 1px solid #30363d;
  border-radius: 8px;
  overflow: hidden;
}

.logs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid #30363d;
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
  background: #0d1117;
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
</style>
