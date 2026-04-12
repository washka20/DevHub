<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { SearchAddon } from '@xterm/addon-search'
import { useTerminalStore } from '../stores/terminal'
import { useSettingsStore } from '../stores/settings'
import { shortCwd } from '../utils/path'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  paneId: string
}>()

const terminalStore = useTerminalStore()
const settingsStore = useSettingsStore()

const terminalEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let searchAddon: SearchAddon | null = null
let serializeAddon: import('@xterm/addon-serialize').SerializeAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false
let intentionalClose = false
let oscCwdReceived = false
let cwdPollTimer: ReturnType<typeof setInterval> | null = null
let cwdStartTimer: ReturnType<typeof setTimeout> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
let connectedSessionId: string | null = null  // tracks which session WS is connected to
let hasConnectedOnce = false
const MAX_RECONNECT_ATTEMPTS = 5
const watchStopHandles: (() => void)[] = []

const searchVisible = ref(false)
const searchQuery = ref('')
const searchInputEl = ref<HTMLInputElement>()

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

function getTabLabel(): string {
  for (const tab of terminalStore.tabs) {
    if (tab.panes.some((p) => p.id === props.paneId)) {
      return tab.label
    }
  }
  return 'terminal'
}

const isDisconnected = computed(() => !pane.value || pane.value.status === 'disconnected')
const isConnecting = computed(() => pane.value?.status === 'connecting')
const isReconnecting = computed(() => pane.value?.status === 'reconnecting')

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

function connectWs(sessionId: string) {
  // Idempotent: skip if already connected/connecting to this session
  if (connectedSessionId === sessionId && ws && ws.readyState <= WebSocket.OPEN) {
    return
  }

  // Close previous connection if switching sessions
  if (ws) {
    intentionalClose = true
    ws.onclose = null
    ws.close()
    ws = null
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }

  if (hasConnectedOnce && term) {
    term.reset()
  }

  oscCwdReceived = false
  intentionalClose = false
  connectedSessionId = sessionId
  reconnectAttempts = 0
  hasConnectedOnce = true

  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const url = `${proto}//${host}/api/terminal/ws/${sessionId}`

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    reconnectAttempts = 0
    // Reset so the first resize after connect always goes through
    lastSentCols = 0
    lastSentRows = 0
    if (term && fitAddon) {
      fitAddon.fit()
      sendResize(term.cols, term.rows)
    }

    // Start CWD polling fallback after 10s if no OSC 7 received
    if (cwdPollTimer) { clearInterval(cwdPollTimer); cwdPollTimer = null }
    if (cwdStartTimer) { clearTimeout(cwdStartTimer); cwdStartTimer = null }
    cwdStartTimer = setTimeout(() => {
      cwdStartTimer = null
      if (!oscCwdReceived && !disposed && pane.value?.sessionId) {
        cwdPollTimer = setInterval(() => pollCwd(), 5000)
      }
    }, 10000)
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
          connectedSessionId = null
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
      connectedSessionId = null
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
  if (reconnectTimer) clearTimeout(reconnectTimer)
  reconnectTimer = setTimeout(() => {
    if (!disposed) connectWs(sessionId)
  }, delay)
}

let lastSentCols = 0
let lastSentRows = 0

function sendResize(cols: number, rows: number) {
  if (ws?.readyState !== WebSocket.OPEN) return
  if (cols === lastSentCols && rows === lastSentRows) return
  lastSentCols = cols
  lastSentRows = rows
  ws.send(JSON.stringify({ type: 'resize', cols, rows }))
}

async function pollCwd() {
  if (!pane.value?.sessionId || disposed) {
    if (cwdPollTimer) { clearInterval(cwdPollTimer); cwdPollTimer = null }
    return
  }
  try {
    const res = await fetch(`/api/terminal/sessions/${pane.value.sessionId}/cwd`)
    if (res.ok) {
      const data = await res.json()
      if (pane.value && data.cwd && data.cwd !== pane.value.cwd) {
        pane.value.cwd = data.cwd
      }
    }
  } catch { /* ignore */ }
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

  searchAddon = new SearchAddon()
  term.loadAddon(searchAddon)

  // OSC 7: shell reports current working directory
  // Format: \e]7;file://hostname/path\a
  term.parser.registerOscHandler(7, (data) => {
    try {
      const url = new URL(data)
      if (url.protocol === 'file:' && url.pathname) {
        const newCwd = decodeURIComponent(url.pathname)
        if (pane.value && newCwd !== pane.value.cwd) {
          pane.value.cwd = newCwd
          oscCwdReceived = true
        }
      }
    } catch { /* ignore malformed OSC 7 */ }
    return false // don't prevent default handling
  })

  term.onTitleChange((title) => {
    if (!title) return
    for (const tab of terminalStore.tabs) {
      if (tab.panes.some((p) => p.id === props.paneId)) {
        tab.label = title
        break
      }
    }
  })

  term.open(terminalEl.value)
  fitAddon.fit()

  try {
    const webgl = new WebglAddon()
    webgl.onContextLost(() => { webgl.dispose() })
    term.loadAddon(webgl)
  } catch { /* fallback to canvas */ }

  import('@xterm/addon-image').then(({ ImageAddon }) => {
    if (term && !disposed) term.loadAddon(new ImageAddon())
  }).catch(() => {})

  import('@xterm/addon-ligatures').then(({ LigaturesAddon }) => {
    if (term && !disposed) term.loadAddon(new LigaturesAddon())
  }).catch(() => {})

  import('@xterm/addon-serialize').then(({ SerializeAddon }) => {
    if (term && !disposed) {
      serializeAddon = new SerializeAddon()
      term.loadAddon(serializeAddon)
    }
  }).catch(() => {})

  const encoder = new TextEncoder()
  term.onData((data: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(encoder.encode(data))
    }
    if (terminalStore.broadcastMode) {
      window.dispatchEvent(new CustomEvent('terminal:broadcast', {
        detail: { data, sourcePaneId: props.paneId },
      }))
    }
  })

  term.onResize(({ cols, rows }) => {
    sendResize(cols, rows)
  })

  term.onBell(() => {
    if (!isActiveTab() && pane.value) {
      pane.value.hasBell = true
      setTimeout(() => {
        if (pane.value) pane.value.hasBell = false
      }, 3000)
      // Browser notification
      if ('Notification' in window && Notification.permission === 'granted') {
        const tabLabel = getTabLabel()
        new Notification('Terminal bell', { body: `Tab: ${tabLabel}` })
      }
    }
  })

  term.onSelectionChange(() => {
    const sel = term?.getSelection()
    if (sel) navigator.clipboard.writeText(sel).catch(() => {})
  })

  terminalEl.value.addEventListener('contextmenu', (e) => {
    e.preventDefault()
    navigator.clipboard.readText().then((text) => {
      if (text && term) term.paste(text)
    }).catch(() => {})
  })

  term.attachCustomKeyEventHandler((e: KeyboardEvent) => {
    if (e.ctrlKey && e.shiftKey && e.code === 'KeyF' && e.type === 'keydown') {
      toggleSearch()
      return false
    }
    return true
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
    }, 16)
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
// Search
// ---------------------------------------------------------------------------

function toggleSearch() {
  searchVisible.value = !searchVisible.value
  if (searchVisible.value) {
    nextTick(() => searchInputEl.value?.focus())
  } else {
    searchAddon?.clearDecorations()
    searchQuery.value = ''
  }
}

function searchNext() {
  if (searchQuery.value && searchAddon) {
    searchAddon.findNext(searchQuery.value, { caseSensitive: false })
  }
}

function searchPrev() {
  if (searchQuery.value && searchAddon) {
    searchAddon.findPrevious(searchQuery.value, { caseSensitive: false })
  }
}

function handleSearchInput() {
  if (searchQuery.value && searchAddon) {
    searchAddon.findNext(searchQuery.value, { caseSensitive: false })
  }
}

function handleSearchKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    toggleSearch()
  } else if (e.key === 'Enter') {
    e.shiftKey ? searchPrev() : searchNext()
  }
}

// ---------------------------------------------------------------------------
// Lazy connect: user clicks the placeholder
// ---------------------------------------------------------------------------

async function handleConnect() {
  if (!pane.value || pane.value.status !== 'disconnected') return

  // Dispose stale terminal — its DOM element was removed when v-if switched
  // to the placeholder. Must re-create on the fresh div.
  if (term) {
    term.dispose()
    term = null
    fitAddon = null
    searchAddon = null
  }

  const sessionId = await terminalStore.connectPane(props.paneId)
  if (!sessionId) return

  await nextTick()  // wait for Vue to render the terminal div
  await document.fonts.ready
  if (disposed) return
  initTerminal()
  connectWs(sessionId)
}

// ---------------------------------------------------------------------------
// Export HTML
// ---------------------------------------------------------------------------

function handleExportHTML() {
  if (!isActiveTab()) return
  if (!serializeAddon) return
  const content = serializeAddon.serializeAsHTML()
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const stamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  const html = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Terminal Export - ${stamp}</title>
  <style>
    body { margin: 0; padding: 16px; background: #1e1e1e; }
  </style>
</head>
<body>
  ${content}
</body>
</html>`
  const blob = new Blob([html], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `terminal-export-${stamp}.html`
  a.click()
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

function handleBroadcast(e: Event) {
  const { data, sourcePaneId } = (e as CustomEvent).detail
  if (sourcePaneId === props.paneId) return
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(new TextEncoder().encode(data))
  }
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

onMounted(async () => {
  window.addEventListener('terminal:broadcast', handleBroadcast)
  window.addEventListener('terminal:export-html', handleExportHTML)

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
  window.removeEventListener('terminal:broadcast', handleBroadcast)
  window.removeEventListener('terminal:export-html', handleExportHTML)
  if (cwdPollTimer) { clearInterval(cwdPollTimer); cwdPollTimer = null }
  if (cwdStartTimer) { clearTimeout(cwdStartTimer); cwdStartTimer = null }
  disposed = true
  connectedSessionId = null
  watchStopHandles.forEach(stop => stop())
  watchStopHandles.length = 0
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeObserver?.disconnect()
  if (ws) {
    ws.onclose = null
    ws.close()
  }
  term?.dispose()
  term = null
  ws = null
  fitAddon = null
  searchAddon = null
  serializeAddon = null
  resizeObserver = null
  resizeTimer = null
  reconnectTimer = null
})

// Watch for sessionId changes (e.g., reconnect after exit, or attachSession from panel)
watch(
  () => pane.value?.sessionId,
  (newId) => {
    if (newId && term) {
      connectWs(newId)  // idempotent — skips if already connected to this session
    }
  },
)

// Force reconnect when ConsoleView reactivates (reclaim output after BottomTerminal stole it)
watch(
  () => terminalStore.reconnectSignal,
  () => {
    if (!pane.value?.sessionId || !term || disposed) return
    const sessionId = pane.value.sessionId
    if (ws) {
      intentionalClose = true
      ws.onclose = null
      ws.close()
      ws = null
    }
    connectedSessionId = null
    connectWs(sessionId)
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
  <div v-else class="terminal-wrapper">
    <div v-if="searchVisible" class="search-bar">
      <input
        ref="searchInputEl"
        v-model="searchQuery"
        class="search-input"
        placeholder="Search..."
        @input="handleSearchInput"
        @keydown="handleSearchKeydown"
      />
      <button class="search-btn" @click="searchPrev" title="Previous">&#9650;</button>
      <button class="search-btn" @click="searchNext" title="Next">&#9660;</button>
      <button class="search-btn" @click="toggleSearch" title="Close">&#10005;</button>
    </div>
    <div ref="terminalEl" class="web-terminal"></div>
    <div v-if="pane?.cwd" class="cwd-badge" :title="pane.cwd">
      {{ shortCwd(pane.cwd) }}
    </div>
  </div>
</template>

<style scoped>
.terminal-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
}

.web-terminal {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.cwd-badge {
  position: absolute;
  top: 4px;
  right: 12px;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  opacity: 0.5;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid var(--border);
  pointer-events: none;
  z-index: 1;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.web-terminal :deep(.xterm) {
  height: 100%;
  padding: 4px 4px 4px 8px;
}

.web-terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 3px 8px;
  outline: none;
}

.search-input:focus {
  border-color: var(--accent-blue);
}

.search-btn {
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
  font-size: 11px;
}

.search-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
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
