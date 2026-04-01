<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { useTerminalStore } from '../stores/terminal'
import { useSettingsStore } from '../stores/settings'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  paneId: string
}>()

const terminalStore = useTerminalStore()
const settingsStore = useSettingsStore()

const terminalEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false
let intentionalClose = false
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 5
const watchStopHandles: (() => void)[] = []

// Find the pane reactively
const pane = computed(() => {
  for (const tab of terminalStore.tabs) {
    const found = tab.panes.find((p) => p.id === props.paneId)
    if (found) return found
  }
  return null
})

function isActiveTab(): boolean {
  for (const tab of terminalStore.tabs) {
    if (tab.panes.some((p) => p.id === props.paneId)) {
      return tab.id === terminalStore.activeTabId
    }
  }
  return false
}

const isDisconnected = computed(() => !pane.value || pane.value.status === 'disconnected')
const isConnecting = computed(() => pane.value?.status === 'connecting')
const isReconnecting = computed(() => pane.value?.status === 'reconnecting')

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

function connectWs(sessionId: string) {
  intentionalClose = false
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const url = `${proto}//${host}/api/terminal/ws/${sessionId}`

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    reconnectAttempts = 0
    if (term && fitAddon) {
      fitAddon.fit()
      sendResize(term.cols, term.rows)
    }
  }

  ws.onmessage = (event: MessageEvent) => {
    if (!term) return
    if (event.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(event.data))

      // Mark pane as having activity if its tab is not active
      if (pane.value && !isActiveTab()) {
        pane.value.hasActivity = true
      }
    } else if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'exit') {
          intentionalClose = true
          terminalStore.handleSessionExit(pane.value?.sessionId || '')
        }
      } catch {
        // ignore
      }
    }
  }

  ws.onclose = () => {
    ws = null
    if (!disposed && !intentionalClose) {
      scheduleReconnect(sessionId)
    }
    intentionalClose = false
  }
}

function scheduleReconnect(sessionId: string) {
  if (disposed) return
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    if (pane.value && pane.value.status !== 'disconnected') {
      pane.value.status = 'disconnected'
      // Don't clear sessionId — session may still be alive, user can retry
    }
    return
  }
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000)
  reconnectAttempts++
  reconnectTimer = setTimeout(() => {
    if (!disposed) connectWs(sessionId)
  }, delay)
}

function sendResize(cols: number, rows: number) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  }
}

// ---------------------------------------------------------------------------
// Terminal init
// ---------------------------------------------------------------------------

function initTerminal() {
  if (!terminalEl.value) return

  term = new Terminal({
    allowProposedApi: true,
    customGlyphs: true,
    cursorBlink: settingsStore.ui.cursorBlink,
    fontFamily: settingsStore.ui.fontFamily,
    fontSize: settingsStore.ui.fontSize,
    lineHeight: 1.0,
    letterSpacing: 0,
    scrollback: settingsStore.ui.scrollback,
    theme: settingsStore.currentTheme,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())

  const unicode11 = new Unicode11Addon()
  term.loadAddon(unicode11)
  term.unicode.activeVersion = '11'

  term.open(terminalEl.value)
  fitAddon.fit()

  const encoder = new TextEncoder()
  term.onData((data: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(encoder.encode(data))
    }
  })

  term.onResize(({ cols, rows }) => {
    sendResize(cols, rows)
  })

  resizeObserver = new ResizeObserver((entries) => {
    const { width, height } = entries[0].contentRect
    if (width === 0 || height === 0) return

    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      if (fitAddon && term) {
        fitAddon.fit()
        sendResize(term.cols, term.rows)
      }
    }, 50)
  })
  resizeObserver.observe(terminalEl.value)

  // Settings watchers
  watchStopHandles.push(watch(() => settingsStore.currentTheme, (theme) => {
    if (term) term.options.theme = theme
  }, { deep: true }))

  watchStopHandles.push(watch(() => settingsStore.ui.fontSize, (size) => {
    if (term) {
      term.options.fontSize = size
      fitAddon?.fit()
    }
  }))

  watchStopHandles.push(watch(() => settingsStore.ui.fontFamily, (font) => {
    if (term) {
      term.options.fontFamily = font
      fitAddon?.fit()
    }
  }))

  watchStopHandles.push(watch(() => settingsStore.ui.cursorBlink, (blink) => {
    if (term) term.options.cursorBlink = blink
  }))
}

// ---------------------------------------------------------------------------
// Lazy connect: user clicks the placeholder
// ---------------------------------------------------------------------------

async function handleConnect() {
  if (!pane.value || pane.value.status !== 'disconnected') return

  const sessionId = await terminalStore.connectPane(props.paneId)
  if (!sessionId) return

  // Init terminal if not yet created, then connect WS
  if (!term) {
    await document.fonts.ready
    if (disposed) return
    initTerminal()
  }
  connectWs(sessionId)
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

onMounted(async () => {
  // If the pane is already connected (e.g. freshly created via addTab), init immediately
  if (pane.value?.status === 'connected' && pane.value.sessionId) {
    await document.fonts.ready
    if (!disposed) {
      initTerminal()
      connectWs(pane.value.sessionId)
    }
    return
  }

  // If the pane is reconnecting (restored from saved layout), auto-reconnect
  if (pane.value?.status === 'reconnecting') {
    const sessionId = await terminalStore.connectPane(props.paneId)
    if (!sessionId || disposed) return
    await nextTick()  // wait for Vue to render the terminal div (v-else branch)
    await document.fonts.ready
    if (disposed) return
    initTerminal()
    connectWs(sessionId)
    return
  }

  // Otherwise, the placeholder is shown and user clicks to connect
})

onBeforeUnmount(() => {
  disposed = true
  watchStopHandles.forEach(stop => stop())
  watchStopHandles.length = 0
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeObserver?.disconnect()
  ws?.close()
  term?.dispose()
  term = null
  ws = null
  fitAddon = null
  resizeObserver = null
  resizeTimer = null
  reconnectTimer = null
})

// Watch for sessionId changes (e.g., reconnect after exit)
watch(
  () => pane.value?.sessionId,
  (newId, oldId) => {
    if (newId && newId !== oldId) {
      if (ws) {
        ws.onclose = null  // prevent stale reconnect
        ws.close()
        ws = null
      }
      if (term) {
        connectWs(newId)
      }
    }
  },
)
</script>

<template>
  <!-- Disconnected placeholder -->
  <div v-if="isDisconnected" class="placeholder-overlay" role="button" aria-label="Connect terminal" @click="handleConnect" @keydown.enter="handleConnect" tabindex="0">
    <div class="placeholder-content">
      <div class="placeholder-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="4 17 10 11 4 5"></polyline>
          <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
      </div>
      <div class="placeholder-text">Press Enter to connect</div>
      <div class="placeholder-cwd">{{ pane?.cwd || 'default directory' }}</div>
    </div>
  </div>

  <!-- Connecting spinner -->
  <div v-else-if="isConnecting" class="placeholder-overlay">
    <div class="placeholder-content">
      <div class="placeholder-spinner"></div>
      <div class="placeholder-text">Connecting...</div>
    </div>
  </div>

  <!-- Reconnecting overlay -->
  <div v-else-if="isReconnecting" class="placeholder-overlay">
    <div class="placeholder-content">
      <div class="placeholder-spinner"></div>
      <div class="placeholder-text">Reconnecting...</div>
      <div class="placeholder-cwd">{{ pane?.cwd || '' }}</div>
    </div>
  </div>

  <!-- Connected terminal -->
  <div v-else ref="terminalEl" class="web-terminal"></div>
</template>

<style scoped>
.web-terminal {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.web-terminal :deep(.xterm) {
  height: 100%;
  padding: 4px 4px 4px 8px;
}

.web-terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
}

.placeholder-overlay {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  cursor: pointer;
  outline: none;
}

.placeholder-overlay:hover .placeholder-content,
.placeholder-overlay:focus .placeholder-content {
  border-color: var(--accent-blue);
}

.placeholder-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 32px 48px;
  border: 1px dashed var(--border);
  border-radius: 12px;
  transition: border-color 0.2s;
}

.placeholder-icon {
  color: var(--text-secondary);
  opacity: 0.5;
}

.placeholder-text {
  font-family: var(--font-mono);
  font-size: 14px;
  color: var(--text-secondary);
}

.placeholder-cwd {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.5;
}

.placeholder-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
