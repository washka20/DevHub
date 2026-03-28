<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionId: string
}>()

const emit = defineEmits<{
  exit: [code: number]
}>()

const terminalEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null

function connectWs(sessionId: string) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const url = `${proto}//${host}/api/terminal/ws/${sessionId}`

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
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
          emit('exit', msg.code)
        }
      } catch {
        // ignore
      }
    }
  }

  ws.onclose = () => {
    ws = null
  }
}

function sendResize(cols: number, rows: number) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  }
}

function initTerminal() {
  if (!terminalEl.value) return

  term = new Terminal({
    cursorBlink: true,
    fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
    fontSize: 14,
    lineHeight: 1.2,
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

  import('@xterm/addon-webgl')
    .then(({ WebglAddon }) => {
      if (term) {
        try {
          term.loadAddon(new WebglAddon())
        } catch {
          // WebGL not available, canvas renderer is fine
        }
      }
    })
    .catch(() => {})

  term.open(terminalEl.value)
  fitAddon.fit()

  term.onData((data: string) => {
    if (ws?.readyState === WebSocket.OPEN) {
      const encoder = new TextEncoder()
      ws.send(encoder.encode(data))
    }
  })

  term.onResize(({ cols, rows }) => {
    sendResize(cols, rows)
  })

  resizeObserver = new ResizeObserver((entries) => {
    // Don't fit when container is hidden (v-show=false → 0 dimensions)
    const { width, height } = entries[0].contentRect
    if (width === 0 || height === 0) return

    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => fitAddon?.fit(), 50)
  })
  resizeObserver.observe(terminalEl.value)

  connectWs(props.sessionId)
}

onMounted(() => {
  initTerminal()
})

onBeforeUnmount(() => {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeObserver?.disconnect()
  ws?.close()
  term?.dispose()
  term = null
  ws = null
  fitAddon = null
  resizeObserver = null
  resizeTimer = null
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
  padding: 4px;
}

.web-terminal :deep(.xterm-viewport) {
  overflow-y: auto !important;
}
</style>
