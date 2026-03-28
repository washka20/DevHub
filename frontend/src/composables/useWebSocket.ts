import { ref, onUnmounted } from 'vue'

// --- Event types ---

export interface ExecOutputEvent {
  type: 'exec_output'
  project: string
  cmd: string
  data: string
}

export interface ExecDoneEvent {
  type: 'exec_done'
  project: string
  cmd: string
  data: { exit_code: number }
}

export interface GitChangedEvent {
  type: 'git_changed'
  project: string
  data: string
}

export interface ContainerStatusEvent {
  type: 'container_status'
  project: string
  data: { name: string; action: string }
}

export type WsEvent = ExecOutputEvent | ExecDoneEvent | GitChangedEvent | ContainerStatusEvent

// --- Composable ---

export function useWebSocket() {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  // Typed callback maps
  type Callback<T> = (event: T) => void
  const execOutputCallbacks: Callback<ExecOutputEvent>[] = []
  const execDoneCallbacks: Callback<ExecDoneEvent>[] = []
  const gitChangedCallbacks: Callback<GitChangedEvent>[] = []
  const containerStatusCallbacks: Callback<ContainerStatusEvent>[] = []
  let genericHandler: ((data: WsEvent) => void) | null = null

  function connect(url = 'ws://localhost:9000/api/ws') {
    // In production, derive ws:// URL from current page host
    if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      url = `${proto}//${window.location.host}/api/ws`
    }

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
    }

    ws.onclose = () => {
      connected.value = false
      reconnectTimer = setTimeout(() => connect(url), 3000)
    }

    ws.onerror = () => {
      ws?.close()
    }

    ws.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data as string) as WsEvent
        dispatch(data)
      } catch {
        // ignore unparseable messages
      }
    }
  }

  function dispatch(event: WsEvent) {
    genericHandler?.(event)

    switch (event.type) {
      case 'exec_output':
        execOutputCallbacks.forEach((cb) => cb(event))
        break
      case 'exec_done':
        execDoneCallbacks.forEach((cb) => cb(event))
        break
      case 'git_changed':
        gitChangedCallbacks.forEach((cb) => cb(event))
        break
      case 'container_status':
        containerStatusCallbacks.forEach((cb) => cb(event))
        break
    }
  }

  // Subscribe to a project's events
  function subscribe(project: string) {
    send({ type: 'subscribe', project })
  }

  // Unsubscribe from a project's events
  function unsubscribe(project: string) {
    send({ type: 'unsubscribe', project })
  }

  // Register typed callbacks
  function onExecOutput(cb: Callback<ExecOutputEvent>) {
    execOutputCallbacks.push(cb)
  }

  function onExecDone(cb: Callback<ExecDoneEvent>) {
    execDoneCallbacks.push(cb)
  }

  function onGitChanged(cb: Callback<GitChangedEvent>) {
    gitChangedCallbacks.push(cb)
  }

  function onContainerStatus(cb: Callback<ContainerStatusEvent>) {
    containerStatusCallbacks.push(cb)
  }

  // Generic message handler (for backwards compat)
  function onMessage(handler: (data: WsEvent) => void) {
    genericHandler = handler
  }

  function send(data: unknown) {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data))
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    ws?.close()
    ws = null
  }

  onUnmounted(disconnect)

  return {
    connected,
    connect,
    disconnect,
    send,
    subscribe,
    unsubscribe,
    onMessage,
    onExecOutput,
    onExecDone,
    onGitChanged,
    onContainerStatus,
  }
}
