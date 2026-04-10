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

const shortcutsListeners = new Set<() => void>()

export function onToggleShortcuts(cb: () => void) {
  shortcutsListeners.add(cb)
  return () => shortcutsListeners.delete(cb)
}

export function useKeyboardShortcuts() {
  const router = useRouter()
  const route = useRoute()
  const terminalStore = useTerminalStore()

  function handleKeydown(e: KeyboardEvent) {
    const inTerminal = (e.target as HTMLElement)?.closest?.('.xterm')
    const inInput = (e.target as HTMLElement)?.closest?.('input, textarea, [contenteditable]')

    // Ctrl+` — toggle bottom terminal (works everywhere including in terminal)
    if (e.ctrlKey && e.code === 'Backquote') {
      e.preventDefault()
      e.stopPropagation()
      if (route.path !== '/console') {
        terminalStore.togglePanel()
      }
      return
    }

    // Terminal-specific shortcuts (only when focus is in terminal)
    if (inTerminal) {
      // Ctrl+Shift+D — toggle split on active tab
      if (e.ctrlKey && e.shiftKey && e.code === 'KeyD') {
        e.preventDefault()
        e.stopPropagation()
        const tab = terminalStore.activeTab
        if (tab) {
          if (tab.panes.length >= 2) {
            terminalStore.detachToTab(tab.id, tab.panes[1].id)
          } else {
            const cwd = tab.panes[0]?.cwd || ''
            terminalStore.splitPane(tab.id, 'horizontal', cwd)
          }
        }
        return
      }

      // Ctrl+Shift+T — new tab
      if (e.ctrlKey && e.shiftKey && e.code === 'KeyT') {
        e.preventDefault()
        e.stopPropagation()
        terminalStore.addTab('')
        return
      }

      // Ctrl+Shift+W — close current tab
      if (e.ctrlKey && e.shiftKey && e.code === 'KeyW') {
        e.preventDefault()
        e.stopPropagation()
        if (terminalStore.activeTabId) {
          terminalStore.closeTab(terminalStore.activeTabId)
        }
        return
      }

      // Ctrl+PageDown — next tab
      if (e.ctrlKey && e.code === 'PageDown') {
        e.preventDefault()
        e.stopPropagation()
        const tabs = terminalStore.tabs
        const idx = tabs.findIndex((t) => t.id === terminalStore.activeTabId)
        if (idx >= 0 && tabs.length > 1) {
          terminalStore.setActiveTab(tabs[(idx + 1) % tabs.length].id)
        }
        return
      }

      // Ctrl+PageUp — previous tab
      if (e.ctrlKey && e.code === 'PageUp') {
        e.preventDefault()
        e.stopPropagation()
        const tabs = terminalStore.tabs
        const idx = tabs.findIndex((t) => t.id === terminalStore.activeTabId)
        if (idx >= 0 && tabs.length > 1) {
          terminalStore.setActiveTab(tabs[(idx - 1 + tabs.length) % tabs.length].id)
        }
        return
      }

      return
    }

    // ? — toggle shortcuts modal (not in input fields)
    if (e.key === '?' && !inInput && !e.ctrlKey && !e.altKey) {
      e.preventDefault()
      shortcutsListeners.forEach(cb => cb())
      return
    }

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
