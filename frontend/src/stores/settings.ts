import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ServerSettings, UISettings, TerminalTheme } from '../types'
import { terminalThemes } from '../data/terminal-themes'

const UI_SETTINGS_KEY = 'devhub-ui-settings'

const defaultUI: UISettings = {
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
  scrollback: 10000,
  cursorBlink: true,
  themeName: 'github-dark',
}

function loadUI(): UISettings {
  try {
    const raw = localStorage.getItem(UI_SETTINGS_KEY)
    if (raw) return { ...defaultUI, ...JSON.parse(raw) }
  } catch { /* ignore */ }
  return { ...defaultUI }
}

export const useSettingsStore = defineStore('settings', () => {
  const server = ref<ServerSettings>({
    port: 9000, projects_dir: '~/project', default_project: 'cfa',
    terminal: { max_sessions: 10, shell: '' },
  })
  const ui = ref<UISettings>(loadUI())
  const shells = ref<string[]>([])

  const currentTheme = computed<TerminalTheme>(() => {
    return terminalThemes[ui.value.themeName] || terminalThemes['github-dark']
  })

  async function fetchSettings() {
    const res = await fetch('/api/settings')
    if (res.ok) server.value = await res.json()
  }

  async function saveSettings(updates: Partial<ServerSettings>) {
    const res = await fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates),
    })
    if (res.ok) await fetchSettings()
    return res.ok
  }

  async function fetchShells() {
    const res = await fetch('/api/settings/shells')
    if (res.ok) shells.value = await res.json()
  }

  function updateUI(partial: Partial<UISettings>) {
    ui.value = { ...ui.value, ...partial }
    localStorage.setItem(UI_SETTINGS_KEY, JSON.stringify(ui.value))
  }

  return { server, ui, shells, currentTheme, fetchSettings, saveSettings, fetchShells, updateUI }
})
