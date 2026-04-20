<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useTerminalStore, type LiveSession } from '../stores/terminal'
import { formatRelativeTime } from '../utils/date'
import { shortCwd } from '../utils/path'

const terminalStore = useTerminalStore()

onMounted(() => {
  terminalStore.fetchLiveSessions()
})

const relativeTime = formatRelativeTime

const enrichedSessions = computed(() => {
  return terminalStore.liveSessions.map((s: LiveSession) => ({
    ...s,
    isAttached: terminalStore.attachedSessionIds.has(s.id),
    shortCwd: shortCwd(s.cwd),
    age: relativeTime(s.created_at),
  }))
})

function handleResume(s: LiveSession) {
  terminalStore.attachSession(s.id, s.cwd)
}

async function handleKill(s: LiveSession) {
  try {
    await terminalStore.destroySession(s.id)
  } catch { /* best-effort */ }
  // Proactively mark any pane using this session as disconnected
  // (don't wait for WS exit message — it may arrive late or not at all)
  terminalStore.handleSessionExit(s.id)
  terminalStore.fetchLiveSessions()
}

function handleRefresh() {
  terminalStore.fetchLiveSessions()
}
</script>

<template>
  <div class="sessions-panel">
    <div class="panel-header">
      <span class="panel-title">Sessions</span>
      <div class="panel-actions">
        <button class="panel-btn" @click="handleRefresh" title="Refresh">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 2.5a5.487 5.487 0 0 0-4.131 1.869l1.204 1.204A.25.25 0 0 1 4.896 6H1.25A.25.25 0 0 1 1 5.75V2.104a.25.25 0 0 1 .427-.177l1.38 1.38A7.002 7.002 0 0 1 14.95 7.16a.75.75 0 0 1-1.49.178A5.5 5.5 0 0 0 8 2.5ZM1.705 8.005a.75.75 0 0 1 .834.656 5.5 5.5 0 0 0 9.592 2.97l-1.204-1.204a.25.25 0 0 1 .177-.427h3.646a.25.25 0 0 1 .25.25v3.646a.25.25 0 0 1-.427.177l-1.38-1.38A7.002 7.002 0 0 1 1.05 8.84a.75.75 0 0 1 .656-.834Z"/>
          </svg>
        </button>
        <button class="panel-btn" @click="terminalStore.toggleSessionsPanel()" title="Close panel">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z"/>
          </svg>
        </button>
      </div>
    </div>

    <div class="panel-body">
      <div v-if="enrichedSessions.length === 0" class="empty">
        No active sessions
      </div>

      <div
        v-for="s in enrichedSessions"
        :key="s.id"
        class="session-row"
        :class="{ attached: s.isAttached }"
      >
        <div class="session-info">
          <div class="session-top">
            <span class="session-dot" :class="{ active: s.isAttached }"></span>
            <span class="session-label">{{ s.isAttached ? 'connected' : 'detached' }}</span>
            <span class="session-age">{{ s.age }}</span>
          </div>
          <div class="session-cwd" :title="s.cwd">{{ s.shortCwd }}</div>
          <div class="session-id">{{ s.id.slice(0, 12) }}</div>
        </div>
        <div class="session-actions">
          <button
            v-if="!s.isAttached"
            class="action-btn resume"
            @click="handleResume(s)"
            title="Open in new tab"
          >
            Resume
          </button>
          <button
            v-else
            class="action-btn goto"
            @click="terminalStore.attachSession(s.id, s.cwd)"
            title="Switch to tab"
          >
            Go to
          </button>
          <button
            class="action-btn kill"
            @click="handleKill(s)"
            title="Kill session"
          >
            Kill
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sessions-panel {
  width: 280px;
  min-width: 280px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary);
  border-left: 1px solid var(--border);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.panel-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.panel-actions {
  display: flex;
  gap: 4px;
}

.panel-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  padding: 0;
}

.panel-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 80px;
  color: var(--text-secondary);
  font-size: 12px;
}

.session-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  margin-bottom: 4px;
  background: var(--bg-primary);
  border: 1px solid transparent;
  transition: border-color 0.15s;
}

.session-row:hover {
  border-color: var(--border);
}

.session-row.attached {
  border-color: var(--ok-2);
}

.session-info {
  min-width: 0;
  flex: 1;
}

.session-top {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 2px;
}

.session-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-secondary);
  flex-shrink: 0;
  opacity: 0.5;
}

.session-dot.active {
  background: var(--accent-green);
  opacity: 1;
}

.session-label {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.session-age {
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.6;
  margin-left: auto;
}

.session-cwd {
  font-size: 12px;
  color: var(--text-primary);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 1px;
}

.session-id {
  font-size: 10px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  opacity: 0.4;
}

.session-actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
}

.action-btn {
  padding: 3px 10px;
  font-size: 11px;
  border: 1px solid var(--border);
  border-radius: 4px;
  cursor: pointer;
  background: none;
  white-space: nowrap;
  font-family: var(--font-ui);
  transition: all 0.15s;
}

.action-btn.resume {
  color: var(--accent-green);
  border-color: color-mix(in oklab, var(--ok) 40%, transparent);
}

.action-btn.resume:hover {
  background: var(--ok-2);
  border-color: var(--accent-green);
}

.action-btn.goto {
  color: var(--accent-blue);
  border-color: color-mix(in oklab, var(--accent) 40%, transparent);
}

.action-btn.goto:hover {
  background: var(--accent-2);
  border-color: var(--accent-blue);
}

.action-btn.kill {
  color: var(--text-secondary);
  opacity: 0.6;
}

.action-btn.kill:hover {
  color: var(--accent-red);
  border-color: var(--accent-red);
  opacity: 1;
  background: var(--bad-2);
}
</style>
