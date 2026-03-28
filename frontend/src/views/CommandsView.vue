<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import CommandButton from '../components/CommandButton.vue'
import TerminalOutput from '../components/TerminalOutput.vue'
import { useProject } from '../composables/useProject'
import { useWebSocket } from '../composables/useWebSocket'
import type { MakeCommand } from '../types'

const { currentProject, projectApiUrl } = useProject()

const terminalLines = ref<string[]>([])
const running = ref(false)
const loadingCmd = ref<string | null>(null)
const commands = ref<MakeCommand[]>([])
const commandsLoading = ref(false)

// Command history (last 5 executions)
interface HistoryEntry {
  cmd: string
  exitCode: number
  timestamp: string
}
const history = ref<HistoryEntry[]>([])

// WebSocket setup
const ws = useWebSocket()

ws.onExecOutput((event) => {
  // Only handle events for the currently displayed command
  if (loadingCmd.value === event.cmd) {
    terminalLines.value.push(event.data)
  }
})

ws.onExecDone((event) => {
  if (loadingCmd.value === event.cmd) {
    const exitCode = event.data.exit_code
    const status = exitCode === 0 ? 'OK' : `FAILED (exit code: ${exitCode})`
    terminalLines.value.push(`--- ${status} ---`)
    running.value = false

    // Add to history
    history.value.unshift({
      cmd: event.cmd,
      exitCode,
      timestamp: new Date().toLocaleTimeString(),
    })
    if (history.value.length > 5) {
      history.value.pop()
    }

    loadingCmd.value = null
  }
})

// Connect WebSocket on mount
onMounted(() => {
  ws.connect()
})

// Subscribe to current project when it changes
let subscribedProject: string | null = null

watch(
  () => currentProject.value,
  (project) => {
    if (subscribedProject) {
      ws.unsubscribe(subscribedProject)
    }
    if (project) {
      ws.subscribe(project.name)
      subscribedProject = project.name
      fetchCommands()
    }
  },
  { immediate: true },
)

// Fetch Makefile commands from API
async function fetchCommands() {
  if (!currentProject.value) return

  commandsLoading.value = true
  try {
    const res = await fetch(`${projectApiUrl.value}/commands`)
    if (res.ok) {
      commands.value = await res.json()
    } else {
      commands.value = []
    }
  } catch {
    commands.value = []
  } finally {
    commandsLoading.value = false
  }
}

// Group commands by category
function groupedCommands(): Record<string, MakeCommand[]> {
  const groups: Record<string, MakeCommand[]> = {}
  for (const cmd of commands.value) {
    const cat = cmd.category || 'Other'
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(cmd)
  }
  return groups
}

// Execute a command
async function execute(cmdName: string) {
  if (!currentProject.value || running.value) return

  loadingCmd.value = cmdName
  running.value = true
  terminalLines.value = [`$ make ${cmdName}`]

  try {
    const res = await fetch(`${projectApiUrl.value}/exec`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cmd: cmdName }),
    })

    if (!res.ok) {
      const data = await res.json()
      terminalLines.value.push(`Error: ${data.error || 'Failed to start command'}`)
      running.value = false
      loadingCmd.value = null
    }
    // On success (202), output will stream via WebSocket
  } catch (err) {
    terminalLines.value.push(`Error: ${err}`)
    running.value = false
    loadingCmd.value = null
  }
}

// Clear terminal output
function clearTerminal() {
  terminalLines.value = []
}

onUnmounted(() => {
  if (subscribedProject) {
    ws.unsubscribe(subscribedProject)
  }
})
</script>

<template>
  <div class="commands-view">
    <header class="page-header">
      <h1>Commands</h1>
      <span v-if="!currentProject" class="no-project">Select a project</span>
    </header>

    <template v-if="currentProject">
      <!-- Command groups from Makefile -->
      <div v-if="commandsLoading" class="loading-state">Loading commands...</div>
      <div v-else-if="commands.length === 0" class="empty-state">
        No Makefile targets found for this project
      </div>
      <template v-else>
        <div v-for="(cmds, category) in groupedCommands()" :key="category" class="cmd-group">
          <h2>{{ category }}</h2>
          <div class="cmd-grid">
            <CommandButton
              v-for="cmd in cmds"
              :key="cmd.name"
              :name="cmd.name"
              :description="cmd.description"
              :category="category"
              :loading="loadingCmd === cmd.name"
              :disabled="running && loadingCmd !== cmd.name"
              @execute="execute(cmd.name)"
            />
          </div>
        </div>
      </template>

      <!-- Terminal output -->
      <section class="section">
        <div class="section-header">
          <h2>Output</h2>
          <button v-if="terminalLines.length > 0" class="btn-clear" @click="clearTerminal">
            Clear
          </button>
        </div>
        <TerminalOutput :lines="terminalLines" :running="running" />
      </section>

      <!-- Command history -->
      <section v-if="history.length > 0" class="section">
        <h2>History</h2>
        <div class="history-list">
          <div
            v-for="(entry, i) in history"
            :key="i"
            class="history-item"
            :class="{ success: entry.exitCode === 0, failure: entry.exitCode !== 0 }"
          >
            <span class="history-status">{{ entry.exitCode === 0 ? 'OK' : 'FAIL' }}</span>
            <span class="history-cmd">make {{ entry.cmd }}</span>
            <span class="history-time">{{ entry.timestamp }}</span>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.commands-view {
}

.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
}

.no-project {
  font-size: 14px;
  color: var(--text-secondary);
}

.loading-state,
.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
}

.cmd-group {
  margin-bottom: 24px;
}

.cmd-group h2 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 10px;
}

.cmd-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
}

.section {
  margin-top: 32px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.section h2 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 10px;
}

.section-header h2 {
  margin-bottom: 0;
}

.btn-clear {
  padding: 4px 12px;
  font-size: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.15s;
}

.btn-clear:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

/* History */
.history-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  background: var(--bg-secondary);
  border-radius: 6px;
  font-size: 13px;
}

.history-status {
  font-weight: 700;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  min-width: 36px;
  text-align: center;
}

.history-item.success .history-status {
  color: var(--accent-green);
  background: rgba(63, 185, 80, 0.1);
}

.history-item.failure .history-status {
  color: var(--accent-red, #f85149);
  background: rgba(248, 81, 73, 0.1);
}

.history-cmd {
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: var(--text-primary);
  flex: 1;
}

.history-time {
  color: var(--text-secondary);
  font-size: 12px;
}
</style>
