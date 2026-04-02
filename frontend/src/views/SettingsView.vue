<script setup lang="ts">
defineOptions({ name: 'SettingsView' })

import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useSettingsStore } from '../stores/settings'
import { terminalThemes, themeNames } from '../data/terminal-themes'
import { siteThemes, siteThemeNames } from '../data/site-themes'
import type { ServerSettings, UISettings } from '../types'

const settingsStore = useSettingsStore()

/* ---------- local form state ---------- */
const localServer = reactive<ServerSettings>(JSON.parse(JSON.stringify(settingsStore.server)))

const localUI = reactive<UISettings>(JSON.parse(JSON.stringify(settingsStore.ui)))

function syncFromStore() {
  Object.assign(localServer, JSON.parse(JSON.stringify(settingsStore.server)))
  Object.assign(localUI, JSON.parse(JSON.stringify(settingsStore.ui)))
}

onMounted(async () => {
  await Promise.all([settingsStore.fetchSettings(), settingsStore.fetchShells()])
  syncFromStore()
})

// Re-sync when store changes externally
watch(() => settingsStore.server, () => {
  if (!isDirty.value) syncFromStore()
}, { deep: true })

/* ---------- dirty tracking ---------- */
const isDirty = computed(() => {
  const s = settingsStore.server
  const u = settingsStore.ui
  return (
    localServer.port !== s.port ||
    localServer.projects_dir !== s.projects_dir ||
    localServer.default_project !== s.default_project ||
    localServer.terminal.shell !== s.terminal.shell ||
    localServer.terminal.max_sessions !== s.terminal.max_sessions ||
    localUI.fontSize !== u.fontSize ||
    localUI.fontFamily !== u.fontFamily ||
    localUI.scrollback !== u.scrollback ||
    localUI.cursorBlink !== u.cursorBlink ||
    localUI.themeName !== u.themeName ||
    localUI.editorEngine !== u.editorEngine ||
    localUI.editorMinimap !== u.editorMinimap ||
    localUI.editorFontSize !== u.editorFontSize
  )
})

/* ---------- save / reset ---------- */
const saving = ref(false)

async function save() {
  saving.value = true
  try {
    await settingsStore.saveSettings({
      port: localServer.port,
      projects_dir: localServer.projects_dir,
      default_project: localServer.default_project,
      terminal: {
        shell: localServer.terminal.shell,
        max_sessions: localServer.terminal.max_sessions,
      },
    })
    settingsStore.updateUI({
      fontSize: localUI.fontSize,
      fontFamily: localUI.fontFamily,
      scrollback: localUI.scrollback,
      cursorBlink: localUI.cursorBlink,
      themeName: localUI.themeName,
      editorEngine: localUI.editorEngine,
      editorMinimap: localUI.editorMinimap,
      editorFontSize: localUI.editorFontSize,
    })
  } finally {
    saving.value = false
  }
}

function reset() {
  syncFromStore()
}

/* ---------- font family options ---------- */
const fontFamilies = [
  { label: 'JetBrains Mono', value: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace" },
  { label: 'Fira Code', value: "'Fira Code', 'JetBrains Mono', monospace" },
  { label: 'SF Mono', value: "'SF Mono', 'JetBrains Mono', monospace" },
  { label: 'Cascadia Code', value: "'Cascadia Code', 'JetBrains Mono', monospace" },
  { label: 'monospace', value: "monospace" },
]

/* ---------- selected theme computed ---------- */
const selectedTheme = computed(() => {
  return terminalThemes[localUI.themeName] || terminalThemes['github-dark']
})

function selectTheme(key: string) {
  localUI.themeName = key
}
</script>

<template>
  <div class="settings-view">
    <!-- Page Header -->
    <div class="page-header">
      <h1>Settings</h1>
      <p>Server and terminal configuration</p>
    </div>

    <!-- General Section -->
    <div class="settings-section">
      <div class="section-header">
        <span class="section-icon">&#9881;</span> General
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Projects Directory</div>
          <div class="setting-desc">Root directory to scan for projects</div>
        </div>
        <div class="setting-control">
          <input type="text" v-model="localServer.projects_dir">
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Default Project</div>
          <div class="setting-desc">Project selected on startup</div>
        </div>
        <div class="setting-control">
          <input type="text" v-model="localServer.default_project">
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Server Port</div>
          <div class="setting-desc">Backend API port</div>
          <div class="setting-hint">Requires restart</div>
        </div>
        <div class="setting-control">
          <input type="number" v-model.number="localServer.port" style="width:100px">
        </div>
      </div>
    </div>

    <!-- Editor Section -->
    <div class="settings-section">
      <div class="section-header">
        <span class="section-icon">&#9998;</span> Editor
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Editor Engine</div>
          <div class="setting-desc">Code editor component</div>
        </div>
        <div class="setting-control">
          <select v-model="localUI.editorEngine">
            <option value="codemirror">CodeMirror 6</option>
            <option value="monaco">Monaco Editor</option>
          </select>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Editor Font Size</div>
          <div class="setting-desc">Font size for code editor (px)</div>
        </div>
        <div class="setting-control">
          <input type="number" v-model.number="localUI.editorFontSize" min="10" max="24" style="width:80px">
          <span class="suffix">px</span>
        </div>
      </div>

      <div v-if="localUI.editorEngine === 'monaco'" class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Minimap</div>
          <div class="setting-desc">Code overview on the right side (Monaco only)</div>
        </div>
        <div class="setting-control">
          <label class="toggle">
            <input type="checkbox" v-model="localUI.editorMinimap">
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>
    </div>

    <!-- Terminal Section -->
    <div class="settings-section">
      <div class="section-header">
        <span class="section-icon">&#9002;</span> Terminal
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Shell</div>
          <div class="setting-desc">Default shell for new terminal sessions</div>
        </div>
        <div class="setting-control">
          <select v-model="localServer.terminal.shell">
            <option v-for="sh in settingsStore.shells" :key="sh" :value="sh">{{ sh }}</option>
          </select>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Font Size</div>
          <div class="setting-desc">Terminal font size in pixels</div>
        </div>
        <div class="setting-control">
          <input type="number" v-model.number="localUI.fontSize" min="8" max="32" style="width:80px">
          <span class="suffix">px</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Font Family</div>
          <div class="setting-desc">Monospace font for terminal</div>
        </div>
        <div class="setting-control">
          <select v-model="localUI.fontFamily">
            <option v-for="f in fontFamilies" :key="f.value" :value="f.value">{{ f.label }}</option>
          </select>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Scrollback</div>
          <div class="setting-desc">Maximum lines kept in terminal history</div>
        </div>
        <div class="setting-control">
          <input type="number" v-model.number="localUI.scrollback" step="1000" style="width:100px">
          <span class="suffix">lines</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Cursor Blink</div>
          <div class="setting-desc">Blinking cursor in terminal</div>
        </div>
        <div class="setting-control">
          <label class="toggle">
            <input type="checkbox" v-model="localUI.cursorBlink">
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <div class="setting-label">Max Sessions</div>
          <div class="setting-desc">Maximum concurrent terminal sessions</div>
        </div>
        <div class="setting-control">
          <input type="number" v-model.number="localServer.terminal.max_sessions" min="1" max="20" style="width:80px">
        </div>
      </div>
    </div>

    <!-- Site Theme Section -->
    <div class="settings-section">
      <div class="section-header">
        <span class="section-icon">&#127912;</span> Site Theme
      </div>

      <div class="theme-grid">
        <div
          v-for="(theme, key) in siteThemes"
          :key="key"
          class="theme-card"
          :class="{ active: localUI.siteThemeName === key }"
          @click="localUI.siteThemeName = key as string; settingsStore.updateUI({ siteThemeName: key as string })"
        >
          <div
            class="theme-preview site-theme-preview"
            :style="{ background: theme['--bg-primary'], color: theme['--text-primary'] }"
          >
            <div class="site-preview-sidebar" :style="{ background: theme['--bg-secondary'], borderColor: theme['--border'] }">
              <div class="site-preview-dot" :style="{ background: theme['--accent-green'] }"></div>
              <div class="site-preview-line" :style="{ background: theme['--accent-blue'] }"></div>
              <div class="site-preview-line" :style="{ background: theme['--text-secondary'] }"></div>
            </div>
            <div class="site-preview-content">
              <div class="site-preview-card" :style="{ borderColor: theme['--border'], background: theme['--bg-secondary'] }">
                <div class="site-preview-accent" :style="{ background: theme['--accent-green'] }"></div>
              </div>
              <div class="site-preview-card" :style="{ borderColor: theme['--border'], background: theme['--bg-secondary'] }">
                <div class="site-preview-accent" :style="{ background: theme['--accent-blue'] }"></div>
              </div>
            </div>
          </div>
          <div class="theme-name">{{ siteThemeNames[key] || key }}</div>
        </div>
      </div>
    </div>

    <!-- Terminal Theme Section -->
    <div class="settings-section">
      <div class="section-header">
        <span class="section-icon">&#9002;</span> Terminal Theme
      </div>

      <div class="theme-grid">
        <div
          v-for="(theme, key) in terminalThemes"
          :key="key"
          class="theme-card"
          :class="{ active: localUI.themeName === key }"
          @click="selectTheme(key as string)"
        >
          <div
            class="theme-preview"
            :style="{ background: theme.background, color: theme.foreground }"
          >
            <span :style="{ color: theme.green }">$</span> git status<br>
            <span :style="{ color: theme.red }">modified:</span> app.vue<br>
            <span :style="{ color: theme.blue }">3 files</span> changed
          </div>
          <div class="theme-name">{{ themeNames[key] || key }}</div>
        </div>
      </div>

      <!-- Live Preview -->
      <div class="preview-terminal">
        <div class="preview-header">
          <span>&#9654;</span> Live Preview
        </div>
        <div
          class="preview-body"
          :style="{
            background: selectedTheme.background,
            color: selectedTheme.foreground,
            fontFamily: localUI.fontFamily,
            fontSize: localUI.fontSize + 'px',
          }"
        >
          <span :style="{ color: selectedTheme.green }">washka@fedora</span>:<span :style="{ color: selectedTheme.blue }">~/project/devhub</span>$ ls -la<br>
          total 42<br>
          drwxr-xr-x  8 washka washka 4096 Mar 28 <span :style="{ color: selectedTheme.blue }">.</span><br>
          -rw-r--r--  1 washka washka  892 Mar 28 <span :style="{ color: selectedTheme.yellow }">go.mod</span><br>
          drwxr-xr-x  3 washka washka 4096 Mar 28 <span :style="{ color: selectedTheme.blue }">frontend/</span><br>
          <span :style="{ color: selectedTheme.green }">washka@fedora</span>:<span :style="{ color: selectedTheme.blue }">~/project/devhub</span>$ <span style="opacity:0.7">_</span>
        </div>
      </div>
    </div>

    <!-- spacer for fixed save bar -->
    <div style="height:60px"></div>

    <!-- Save Bar -->
    <Transition name="save-bar">
      <div v-if="isDirty" class="save-bar">
        <span class="unsaved-dot"></span>
        <span class="save-text">Unsaved changes</span>
        <span style="flex:1"></span>
        <button class="btn" @click="reset">Reset</button>
        <button class="btn btn-primary" @click="save" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Changes' }}
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.settings-view {
  padding: 32px 40px;
  max-width: 800px;
}

/* Page Header */
.page-header { margin-bottom: 32px; }
.page-header h1 { font-size: 28px; font-weight: 700; }
.page-header p { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }

/* Settings Sections */
.settings-section { margin-bottom: 32px; }
.section-header {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 8px;
}
.section-icon { color: var(--text-secondary); font-size: 14px; }

/* Setting Row */
.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid rgba(48, 54, 61, 0.4);
}
.setting-row:last-child { border-bottom: none; }
.setting-info { flex: 1; }
.setting-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}
.setting-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}
.setting-hint {
  font-size: 11px;
  color: var(--accent-orange);
  margin-top: 2px;
}

/* Inputs */
.setting-control {
  min-width: 200px;
  text-align: right;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}
.setting-control input[type="text"],
.setting-control input[type="number"],
.setting-control select {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-primary);
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 6px 12px;
  outline: none;
  width: 200px;
  transition: all var(--transition-fast);
}
.setting-control input:focus,
.setting-control select:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.3);
}
.setting-control select {
  cursor: pointer;
  appearance: auto;
}
.suffix {
  color: var(--text-secondary);
  font-size: 12px;
}

/* Toggle Switch */
.toggle { position: relative; display: inline-block; width: 40px; height: 22px; }
.toggle input { opacity: 0; width: 0; height: 0; }
.toggle-slider {
  position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0;
  background: var(--border); border-radius: 11px; transition: 0.2s;
}
.toggle-slider:before {
  position: absolute; content: ""; height: 16px; width: 16px; left: 3px; bottom: 3px;
  background: var(--text-secondary); border-radius: 50%; transition: 0.2s;
}
.toggle input:checked + .toggle-slider { background: var(--accent-blue); }
.toggle input:checked + .toggle-slider:before { transform: translateX(18px); background: #fff; }

/* Theme Cards */
.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  margin-top: 12px;
}
.theme-card {
  border: 2px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.15s;
}
.theme-card:hover { border-color: var(--text-secondary); transform: translateY(-1px); }
.theme-card.active { border-color: var(--accent-blue); box-shadow: var(--glow-blue); }
.theme-preview {
  height: 60px;
  padding: 8px 10px;
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.4;
  overflow: hidden;
}
.theme-name {
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  text-align: center;
}

.site-theme-preview {
  display: flex;
  gap: 4px;
  padding: 6px !important;
  font-size: 0;
}

.site-preview-sidebar {
  width: 24px;
  border-right: 1px solid;
  border-radius: 3px 0 0 3px;
  padding: 4px 3px;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.site-preview-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  margin-bottom: 2px;
}

.site-preview-line {
  height: 2px;
  border-radius: 1px;
  opacity: 0.7;
}

.site-preview-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 2px;
}

.site-preview-card {
  flex: 1;
  border: 1px solid;
  border-radius: 3px;
  position: relative;
  overflow: hidden;
}

.site-preview-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 2px;
}

/* Live Preview Terminal */
.preview-terminal {
  margin-top: 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}
.preview-header {
  background: var(--bg-secondary);
  padding: 6px 12px;
  font-size: 12px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  gap: 6px;
}
.preview-body {
  padding: 12px 16px;
  font-family: var(--font-mono);
  font-size: 14px;
  line-height: 1.5;
  min-height: 120px;
  transition: background 0.15s, color 0.15s, font-size 0.15s;
}

/* Save Bar */
.save-bar {
  position: fixed;
  bottom: 0;
  left: var(--sidebar-width);
  right: 0;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  padding: 12px 40px;
  display: flex;
  align-items: center;
  gap: 12px;
  z-index: 100;
}
.unsaved-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--accent-orange);
  box-shadow: 0 0 6px var(--accent-orange);
}
.save-text {
  font-size: 13px;
  color: var(--text-secondary);
}
.btn {
  padding: 6px 20px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}
.btn:hover { color: var(--text-primary); border-color: var(--text-secondary); }
.btn-primary {
  background: var(--accent-green);
  border-color: var(--accent-green);
  color: #fff;
}
.btn-primary:hover { box-shadow: 0 0 12px rgba(63, 185, 80, 0.4); }
.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Save bar transition */
.save-bar-enter-active,
.save-bar-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.save-bar-enter-from,
.save-bar-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
