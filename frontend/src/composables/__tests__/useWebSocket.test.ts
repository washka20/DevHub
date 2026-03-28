import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import type { ExecOutputEvent } from '../useWebSocket'

// --- Mock WebSocket class ---
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readonly CONNECTING = 0
  readonly OPEN = 1
  readonly CLOSING = 2
  readonly CLOSED = 3

  url: string
  readyState: number = MockWebSocket.OPEN
  onopen: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  onerror: ((ev: Event) => void) | null = null
  onmessage: ((ev: MessageEvent) => void) | null = null

  send = vi.fn()
  close = vi.fn()

  constructor(url: string) {
    this.url = url
    // Simulate connection open asynchronously
    setTimeout(() => {
      this.onopen?.(new Event('open'))
    }, 0)
  }
}

// Store reference to last created instance for test assertions
let lastWsInstance: MockWebSocket | null = null
const OriginalMockWebSocket = MockWebSocket

function trackingWebSocket(url: string) {
  const instance = new OriginalMockWebSocket(url)
  lastWsInstance = instance
  return instance
}

describe('useWebSocket', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    lastWsInstance = null

    // Mock WebSocket globally since happy-dom does not provide it
    vi.stubGlobal('WebSocket', Object.assign(trackingWebSocket, {
      CONNECTING: 0,
      OPEN: 1,
      CLOSING: 2,
      CLOSED: 3,
    }))

    // Stub window.location for the URL derivation logic
    Object.defineProperty(window, 'location', {
      value: { hostname: 'localhost', protocol: 'http:', host: 'localhost:5173' },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  // Import dynamically so the global WebSocket mock is in place
  async function getComposable() {
    // Use dynamic import to pick up the global mock
    // We need to reset modules so the composable sees our stubbed WebSocket
    const mod = await import('../useWebSocket')

    // useWebSocket calls onUnmounted, which requires a component setup context.
    // We wrap it in a minimal Vue app scope.
    const { createApp, defineComponent } = await import('vue')
    let result: ReturnType<typeof mod.useWebSocket> | undefined
    const App = defineComponent({
      setup() {
        result = mod.useWebSocket()
        return () => null
      },
    })
    const app = createApp(App)
    app.mount(document.createElement('div'))
    return { result: result!, app }
  }

  it('connect creates a new WebSocket', async () => {
    vi.useFakeTimers()
    const { result } = await getComposable()

    result.connect('ws://test:9000/api/ws')

    expect(lastWsInstance).not.toBeNull()
    expect(lastWsInstance!.url).toBe('ws://test:9000/api/ws')

    // Trigger onopen
    await vi.advanceTimersByTimeAsync(0)
    expect(result.connected.value).toBe(true)
  })

  it('subscribe sends subscribe message', async () => {
    vi.useFakeTimers()
    const { result } = await getComposable()

    result.connect('ws://test:9000/api/ws')
    await vi.advanceTimersByTimeAsync(0) // trigger onopen

    result.subscribe('cfa')

    expect(lastWsInstance!.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'subscribe', project: 'cfa' }),
    )
  })

  it('onExecOutput calls callback on exec_output event', async () => {
    vi.useFakeTimers()
    const { result } = await getComposable()

    const callback = vi.fn()
    result.onExecOutput(callback)

    result.connect('ws://test:9000/api/ws')
    await vi.advanceTimersByTimeAsync(0)

    // Simulate incoming message
    const event: ExecOutputEvent = {
      type: 'exec_output',
      project: 'myapp',
      cmd: 'make build',
      data: 'Building...',
    }

    lastWsInstance!.onmessage?.(new MessageEvent('message', {
      data: JSON.stringify(event),
    }))

    expect(callback).toHaveBeenCalledTimes(1)
    expect(callback).toHaveBeenCalledWith(event)
  })

  it('disconnect closes the connection', async () => {
    vi.useFakeTimers()
    const { result } = await getComposable()

    result.connect('ws://test:9000/api/ws')
    await vi.advanceTimersByTimeAsync(0)

    result.disconnect()

    expect(lastWsInstance!.close).toHaveBeenCalled()
  })
})
