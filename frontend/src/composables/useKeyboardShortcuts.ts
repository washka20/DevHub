import { onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useTerminalStore } from '../stores/terminal'

const ROUTE_MAP: Record<string, string> = {
  Digit1: '/',
  Digit2: '/git',
  Digit3: '/commands',
  Digit4: '/docker',
  Digit5: '/console',
  Digit6: '/readme',
  Digit7: '/notes',
  Digit8: '/editor',
}

export function useKeyboardShortcuts() {
  const router = useRouter()
  const route = useRoute()
  const terminalStore = useTerminalStore()

  function handleKeydown(e: KeyboardEvent) {
    const inTerminal = (e.target as HTMLElement)?.closest?.('.xterm')

    // Ctrl+` — toggle bottom terminal (works everywhere including in terminal)
    if (e.ctrlKey && e.code === 'Backquote') {
      e.preventDefault()
      e.stopPropagation()
      if (route.path !== '/console') {
        terminalStore.togglePanel()
      }
      return
    }

    // Everything below blocked when focus is in terminal
    if (inTerminal) return

    // Alt+N — navigate
    if (e.altKey && !e.ctrlKey && !e.shiftKey) {
      const target = ROUTE_MAP[e.code]
      if (target && route.path !== target) {
        e.preventDefault()
        router.push(target)
      }
      return
    }

    // Ctrl+Shift+G — go to Git
    if (e.ctrlKey && e.shiftKey && e.code === 'KeyG') {
      e.preventDefault()
      if (route.path !== '/git') router.push('/git')
      return
    }

    // Ctrl+S — emit editor save event
    if (e.ctrlKey && !e.shiftKey && e.code === 'KeyS') {
      if (route.path === '/editor') {
        e.preventDefault()
        window.dispatchEvent(new CustomEvent('editor:save'))
      }
      return
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown, true)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleKeydown, true)
  })
}
