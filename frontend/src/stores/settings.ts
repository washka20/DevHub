import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { settingsApi } from '../api/projects'
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
  editorKeymap: 'default',
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
  const toast = useToast()

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
    try {
      server.value = await settingsApi.fetch()
    } catch (e) {
      toast.show('error', `Failed to load settings: ${getErrorMessage(e)}`)
    }
  }

  async function saveSettings(updates: Partial<ServerSettings>) {
    try {
      await settingsApi.save(updates)
      await fetchSettings()
      return true
    } catch (e) {
      toast.show('error', `Failed to save settings: ${getErrorMessage(e)}`)
      return false
    }
  }

  async function fetchShells() {
    try {
      shells.value = await settingsApi.shells()
    } catch { /* ignore */ }
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
