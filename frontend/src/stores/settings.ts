import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import type { ServerSettings, UISettings, TerminalTheme } from '../types'
import { terminalThemes } from '../data/terminal-themes'
import { siteThemes } from '../data/site-themes'

const UI_SETTINGS_KEY = 'devhub-ui-settings'

const defaultUI: UISettings = {
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
  scrollback: 10000,
  cursorBlink: true,
  themeName: 'github-dark',
  siteThemeName: 'github-dark',
  editorEngine: 'codemirror',
  editorMinimap: true,
  editorFontSize: 13,
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

  function applySiteTheme(themeName: string) {
    const theme = siteThemes[themeName]
    if (!theme) return
    const root = document.documentElement
    for (const [key, value] of Object.entries(theme)) {
      root.style.setProperty(key, value)
    }
  }

  // Apply site theme on init and when changed
  applySiteTheme(ui.value.siteThemeName)
  watch(() => ui.value.siteThemeName, (name) => applySiteTheme(name))

  return { server, ui, shells, currentTheme, fetchSettings, saveSettings, fetchShells, updateUI, applySiteTheme }
})
