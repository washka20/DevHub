<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { useTerminalStore } from '../stores/terminal'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionId: string
}>()

const terminalStore = useTerminalStore()

const terminalEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 5

function connectWs(sessionId: string) {
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
    } else if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'exit') {
          terminalStore.handleSessionExit(props.sessionId)
        }
      } catch {
        // ignore
      }
    }
  }

  ws.onclose = () => {
    ws = null
    if (!disposed) {
      scheduleReconnect(sessionId)
    }
  }
}

function scheduleReconnect(sessionId: string) {
  if (disposed || reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) return
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

function initTerminal() {
  if (!terminalEl.value) return

  term = new Terminal({
    allowProposedApi: true,
    cursorBlink: true,
    customGlyphs: true,
    fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
    fontSize: 14,
    lineHeight: 1.0,
    letterSpacing: 0,
    scrollback: 10000,
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      selectionBackground: 'rgba(88, 166, 255, 0.3)',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39d353',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d364',
      brightWhite: '#f0f6fc',
    },
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

  connectWs(props.sessionId)
}

onMounted(async () => {
  await document.fonts.ready
  if (!disposed) initTerminal()
})

onBeforeUnmount(() => {
  disposed = true
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

watch(
  () => props.sessionId,
  (newId, oldId) => {
    if (newId !== oldId) {
      ws?.close()
      connectWs(newId)
    }
  },
)
</script>

<template>
  <div ref="terminalEl" class="web-terminal"></div>
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
</style>
