import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { settingsApi } from '../api/projects'
import type { ServerSettings, UISettings, TerminalTheme } from '../types'
import { terminalThemes } from '../data/terminal-themes'

const UI_SETTINGS_KEY = 'devhub-ui-settings'

const defaultUI: UISettings = {
  fontSize: 14,
  fontFamily: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
  scrollback: 10000,
  cursorBlink: true,
  themeName: 'github-dark',  // xterm terminal palette (separate from app theme)
  siteThemeName: 'devhub',   // legacy field, no longer applied
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

  // Legacy site-theme overrides (`--bg-primary`, `--accent-blue`, etc. inline on
  // <html>) used to fight the design-system tokens, so the user always saw the
  // old GitHub-dark palette regardless of the warm dark/light toggle. We now
  // strip those inline overrides on boot and turn applySiteTheme into a no-op;
  // the design-system dark/light themes are the single source of truth.
  function applySiteTheme(_themeName: string) {
    const root = document.documentElement
    const keys = [
      '--bg-primary', '--bg-secondary', '--bg-tertiary', '--border',
      '--text-primary', '--text-secondary',
      '--accent-blue', '--accent-green', '--accent-red', '--accent-orange', '--accent-purple',
      '--glow-blue', '--glow-green', '--glow-red', '--glow-orange', '--glow-purple',
    ]
    for (const k of keys) root.style.removeProperty(k)
  }

  applySiteTheme(ui.value.siteThemeName)

  return { server, ui, shells, currentTheme, fetchSettings, saveSettings, fetchShells, updateUI, applySiteTheme }
})
