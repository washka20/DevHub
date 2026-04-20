<script setup lang="ts">
defineOptions({ name: 'SettingsView' })

import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useSettingsStore } from '../stores/settings'
import { terminalThemes, themeNames } from '../data/terminal-themes'
import { useTheme } from '../composables/useTheme'
import ThemeToggle from '../components/ThemeToggle.vue'
import type { ServerSettings, UISettings } from '../types'

const settingsStore = useSettingsStore()
const { theme } = useTheme()

type Section = 'general' | 'appearance' | 'editor' | 'terminal' | 'theme' | 'about'
const activeSection = ref<Section>('general')

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
    localUI.editorKeymap !== u.editorKeymap ||
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
      editorKeymap: localUI.editorKeymap,
      editorMinimap: localUI.editorMinimap,
      editorFontSize: localUI.editorFontSize,
    })
  } finally {
    saving.value = false
  }
}

function reset() { syncFromStore() }

/* ---------- font family options ---------- */
const fontFamilies = [
  { label: 'JetBrains Mono', value: "'JetBrains Mono', 'SF Mono', 'Fira Code', 'Cascadia Code', monospace" },
  { label: 'Fira Code', value: "'Fira Code', 'JetBrains Mono', monospace" },
  { label: 'SF Mono', value: "'SF Mono', 'JetBrains Mono', monospace" },
  { label: 'Cascadia Code', value: "'Cascadia Code', 'JetBrains Mono', monospace" },
  { label: 'monospace', value: "monospace" },
]

const selectedTerm = computed(() => terminalThemes[localUI.themeName] || terminalThemes['github-dark'])

function selectTerm(key: string) { localUI.themeName = key }

const sections: { id: Section; label: string; group: string }[] = [
  { id: 'general',    label: 'General',         group: 'Preferences' },
  { id: 'appearance', label: 'Appearance',      group: 'Preferences' },
  { id: 'editor',     label: 'Editor',          group: 'Preferences' },
  { id: 'terminal',   label: 'Terminal',        group: 'Preferences' },
  { id: 'theme',      label: 'Terminal theme',  group: 'Preferences' },
  { id: 'about',      label: 'About',           group: 'System' },
]

const groupedSections = computed(() => {
  const groups = new Map<string, { id: Section; label: string }[]>()
  for (const s of sections) {
    const arr = groups.get(s.group) ?? []
    arr.push({ id: s.id, label: s.label })
    groups.set(s.group, arr)
  }
  return Array.from(groups.entries()).map(([name, items]) => ({ name, items }))
})
</script>

<template>
  <div class="settings-view">
    <header class="page-head">
      <div>
        <h1>Settings</h1>
        <p class="sub">User and workspace preferences — stored in <code>~/.config/devhub/config.yaml</code>.</p>
      </div>
      <span class="chip mute">route: /settings</span>
    </header>

    <div class="card settings-card">
      <div class="settings-grid">
        <aside class="set-nav">
          <template v-for="g in groupedSections" :key="g.name">
            <h6>{{ g.name }}</h6>
            <a
              v-for="s in g.items"
              :key="s.id"
              class="set-nav-item"
              :class="{ on: activeSection === s.id }"
              @click="activeSection = s.id"
            >{{ s.label }}</a>
          </template>
        </aside>

        <div class="panel">
          <!-- GENERAL -->
          <section v-if="activeSection === 'general'">
            <h2>General</h2>
            <p class="lede">Core behaviour of the hub. Changes are saved after pressing the Save button below.</p>

            <div class="field">
              <div><label>Projects directory</label><div class="hint">Root folder DevHub scans for projects.</div></div>
              <div class="ctl">
                <div class="ds-input">
                  <input v-model="localServer.projects_dir" type="text" spellcheck="false" />
                </div>
              </div>
            </div>

            <div class="field">
              <div><label>Default project</label><div class="hint">Loaded on startup.</div></div>
              <div class="ctl">
                <div class="ds-input">
                  <input v-model="localServer.default_project" type="text" spellcheck="false" />
                </div>
              </div>
            </div>

            <div class="field">
              <div><label>Server port</label><div class="hint">Backend API port. Requires restart.</div></div>
              <div class="ctl">
                <div class="ds-input" style="max-width: 160px">
                  <input v-model.number="localServer.port" type="number" />
                </div>
              </div>
            </div>
          </section>

          <!-- APPEARANCE -->
          <section v-else-if="activeSection === 'appearance'">
            <h2>Appearance</h2>
            <p class="lede">Warm dark is the default. The palette uses OKLCH so both modes stay harmonious.</p>

            <div class="field">
              <div><label>Theme</label><div class="hint">Switch at any time — <span class="kbd">{{ theme === 'dark' ? 'dark' : 'light' }}</span> is active.</div></div>
              <div class="ctl row-ctl">
                <ThemeToggle />
              </div>
            </div>

          </section>

          <!-- EDITOR -->
          <section v-else-if="activeSection === 'editor'">
            <h2>Editor</h2>
            <p class="lede">Code-editor engine and input behaviour.</p>

            <div class="field">
              <div><label>Engine</label><div class="hint">Switch between two editors.</div></div>
              <div class="ctl">
                <div class="choices two">
                  <button
                    type="button"
                    class="choice"
                    :class="{ sel: localUI.editorEngine === 'codemirror' }"
                    @click="localUI.editorEngine = 'codemirror'"
                  >
                    <span class="t">CodeMirror 6</span>
                    <span class="d">Lightweight, fast. Best for most files.</span>
                  </button>
                  <button
                    type="button"
                    class="choice"
                    :class="{ sel: localUI.editorEngine === 'monaco' }"
                    @click="localUI.editorEngine = 'monaco'"
                  >
                    <span class="t">Monaco</span>
                    <span class="d">VS Code engine. Minimap, IntelliSense.</span>
                  </button>
                </div>
              </div>
            </div>

            <div class="field">
              <div><label>Keymap</label><div class="hint">Keyboard scheme inside the editor.</div></div>
              <div class="ctl">
                <select v-model="localUI.editorKeymap" class="select-native">
                  <option value="default">Default</option>
                  <option value="vim">Vim</option>
                </select>
              </div>
            </div>

            <div class="field">
              <div><label>Font size</label></div>
              <div class="ctl row-ctl">
                <div class="ds-input" style="width: 100px">
                  <input v-model.number="localUI.editorFontSize" type="number" min="10" max="24" />
                </div>
                <span class="unit">px</span>
              </div>
            </div>

            <div v-if="localUI.editorEngine === 'monaco'" class="field">
              <div><label>Minimap</label><div class="hint">Code overview on the right.</div></div>
              <div class="ctl">
                <button
                  type="button"
                  class="tgl"
                  :class="{ on: localUI.editorMinimap }"
                  aria-label="Minimap"
                  @click="localUI.editorMinimap = !localUI.editorMinimap"
                ></button>
              </div>
            </div>
          </section>

          <!-- TERMINAL (shell + session limits) -->
          <section v-else-if="activeSection === 'terminal'">
            <h2>Terminal</h2>
            <p class="lede">Shell, font, and scrollback for the web console.</p>

            <div class="field">
              <div><label>Shell</label><div class="hint">Default shell for new sessions.</div></div>
              <div class="ctl">
                <select v-model="localServer.terminal.shell" class="select-native">
                  <option v-for="sh in settingsStore.shells" :key="sh" :value="sh">{{ sh }}</option>
                </select>
              </div>
            </div>

            <div class="field">
              <div><label>Font family</label></div>
              <div class="ctl">
                <select v-model="localUI.fontFamily" class="select-native">
                  <option v-for="f in fontFamilies" :key="f.value" :value="f.value">{{ f.label }}</option>
                </select>
              </div>
            </div>

            <div class="field">
              <div><label>Font size</label></div>
              <div class="ctl row-ctl">
                <div class="ds-input" style="width: 100px">
                  <input v-model.number="localUI.fontSize" type="number" min="8" max="32" />
                </div>
                <span class="unit">px</span>
              </div>
            </div>

            <div class="field">
              <div><label>Scrollback</label><div class="hint">Lines kept in history.</div></div>
              <div class="ctl row-ctl">
                <div class="ds-input" style="width: 120px">
                  <input v-model.number="localUI.scrollback" type="number" step="1000" />
                </div>
                <span class="unit">lines</span>
              </div>
            </div>

            <div class="field">
              <div><label>Cursor blink</label></div>
              <div class="ctl">
                <button
                  type="button"
                  class="tgl"
                  :class="{ on: localUI.cursorBlink }"
                  aria-label="Cursor blink"
                  @click="localUI.cursorBlink = !localUI.cursorBlink"
                ></button>
              </div>
            </div>

            <div class="field">
              <div><label>Max sessions</label><div class="hint">Concurrent terminal sessions allowed.</div></div>
              <div class="ctl row-ctl">
                <div class="ds-input" style="width: 100px">
                  <input v-model.number="localServer.terminal.max_sessions" type="number" min="1" max="20" />
                </div>
              </div>
            </div>
          </section>

          <!-- TERMINAL THEME -->
          <section v-else-if="activeSection === 'theme'">
            <h2>Terminal theme</h2>
            <p class="lede">Colour palette for the terminal pane only — app chrome uses the warm dark / warm paper tokens.</p>

            <div class="choices grid">
              <button
                v-for="(term, key) in terminalThemes"
                :key="key"
                type="button"
                class="choice term-choice"
                :class="{ sel: localUI.themeName === key }"
                @click="selectTerm(key as string)"
              >
                <span class="t">{{ themeNames[key as string] || key }}</span>
                <span class="preview" :style="{ background: term.background, color: term.foreground }">
                  <span :style="{ color: term.green }">$</span> git status<br />
                  <span :style="{ color: term.red }">M</span> app.vue
                </span>
              </button>
            </div>

            <div class="preview-terminal">
              <div class="preview-header">
                <span class="chip info">live preview</span>
              </div>
              <div
                class="preview-body"
                :style="{
                  background: selectedTerm.background,
                  color: selectedTerm.foreground,
                  fontFamily: localUI.fontFamily,
                  fontSize: localUI.fontSize + 'px',
                }"
              >
                <span :style="{ color: selectedTerm.green }">washka@fedora</span>:<span :style="{ color: selectedTerm.blue }">~/project/devhub</span>$ ls -la<br />
                total 42<br />
                -rw-r--r-- 1 washka washka  892 <span :style="{ color: selectedTerm.yellow }">go.mod</span><br />
                drwxr-xr-x 3 washka washka 4096 <span :style="{ color: selectedTerm.blue }">frontend/</span><br />
                <span :style="{ color: selectedTerm.green }">washka@fedora</span>:<span :style="{ color: selectedTerm.blue }">~/project/devhub</span>$ <span style="opacity:.7">_</span>
              </div>
            </div>
          </section>

          <!-- ABOUT -->
          <section v-else-if="activeSection === 'about'">
            <h2>About DevHub</h2>
            <p class="lede">Local-first developer dashboard.</p>
            <div class="field">
              <div><label>Version</label></div>
              <div class="ctl"><span class="mono">0.0.0-dev</span></div>
            </div>
            <div class="field">
              <div><label>Config path</label></div>
              <div class="ctl"><span class="mono">~/.config/devhub/config.yaml</span></div>
            </div>
            <div class="field">
              <div><label>Reset onboarding</label><div class="hint">Re-run the first-time wizard.</div></div>
              <div class="ctl">
                <button class="btn" @click="() => { try { localStorage.removeItem('devhub.onboarded') } catch {} }">Clear flag</button>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>

    <div style="height: 72px"></div>

    <Transition name="save-bar">
      <div v-if="isDirty" class="save-bar">
        <span class="unsaved-dot"></span>
        <span class="save-text">Unsaved changes</span>
        <span style="flex:1"></span>
        <button class="btn" @click="reset">Reset</button>
        <button class="btn primary" :disabled="saving" @click="save">
          {{ saving ? 'Saving…' : 'Save changes' }}
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  flex-direction: column;
  gap: var(--s4);
  width: 100%;
}

.settings-card { padding: 0; overflow: hidden; }

.settings-grid {
  display: grid;
  grid-template-columns: 220px 1fr;
  min-height: 620px;
}
@media (max-width: 900px) {
  .settings-grid { grid-template-columns: 180px 1fr; }
}

.set-nav {
  background: var(--bg-1);
  border-right: 1px solid var(--line-soft);
  padding: 10px 6px;
}
.set-nav h6 {
  margin: 12px 8px 4px;
  font-size: 10.5px;
  letter-spacing: .14em;
  text-transform: uppercase;
  color: var(--fg-3);
  font-weight: 600;
}
.set-nav-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: var(--r1);
  font-size: 13px;
  color: var(--fg-2);
  text-decoration: none;
  cursor: pointer;
  transition: background var(--t-fast), color var(--t-fast);
  position: relative;
}
.set-nav-item:hover { background: var(--bg-2); color: var(--fg); }
.set-nav-item.on { background: var(--accent-2); color: var(--fg); }
.set-nav-item.on::before {
  content: "";
  position: absolute;
  left: -6px; top: 8px; bottom: 8px;
  width: 3px;
  border-radius: 3px;
  background: var(--accent);
  box-shadow: 0 0 10px var(--accent);
}

.panel { padding: 24px 28px; overflow: auto; min-width: 0; }
.panel h2 { margin: 0 0 4px; font-size: 22px; font-weight: 700; color: var(--fg); letter-spacing: -0.01em; }
.panel .lede { color: var(--fg-3); margin: 0 0 22px; font-size: 13.5px; }
.panel code {
  font-family: var(--mono);
  font-size: 12px;
  background: var(--bg-2);
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--fg-2);
}

.field {
  padding: 16px 0;
  border-bottom: 1px solid var(--line-soft);
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 24px;
  align-items: start;
}
.field:last-child { border-bottom: 0; }
.field label { font-weight: 600; font-size: 13.5px; color: var(--fg); }
.field .hint { color: var(--fg-3); font-size: 12.5px; margin-top: 3px; }
.field .ctl { display: flex; flex-direction: column; gap: 8px; align-items: flex-start; min-width: 0; }
.field .ctl.row-ctl { flex-direction: row; align-items: center; gap: 10px; }
.field .unit { color: var(--fg-3); font-size: 12px; }
.field .mono { font-family: var(--mono); font-size: 12.5px; color: var(--fg-2); }

/* Shared input override for scoped usage */
.ds-input {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  background: var(--bg-2);
  border: 1px solid var(--line);
  border-radius: var(--r1);
  width: 100%;
  max-width: 360px;
}
.ds-input input {
  flex: 1;
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--fg);
  font-size: 13px;
  font-family: var(--ui);
  padding: 0;
  width: 100%;
}
.ds-input:focus-within { border-color: var(--accent); box-shadow: 0 0 0 2px color-mix(in oklab, var(--accent) 25%, transparent); }

.select-native {
  display: inline-flex;
  align-items: center;
  padding: 7px 12px;
  border: 1px solid var(--line);
  background: var(--bg-2);
  border-radius: var(--r1);
  font-size: 13px;
  color: var(--fg);
  font-family: var(--ui);
  cursor: pointer;
  min-width: 220px;
}
.select-native:focus { outline: none; border-color: var(--accent); }

/* Toggle switch */
.tgl {
  position: relative;
  width: 36px;
  height: 20px;
  border-radius: var(--r-pill);
  background: var(--bg-3);
  border: 1px solid var(--line);
  cursor: pointer;
  transition: background var(--t-fast), border-color var(--t-fast);
  padding: 0;
}
.tgl::after {
  content: "";
  position: absolute;
  left: 2px; top: 2px;
  width: 14px; height: 14px;
  border-radius: 50%;
  background: var(--fg-2);
  transition: all var(--t-fast);
}
.tgl.on { background: var(--accent); border-color: var(--accent); }
.tgl.on::after { left: 18px; background: var(--accent-ink); }

/* Choice cards */
.choices {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 10px;
  width: 100%;
}
.choices.two { grid-template-columns: repeat(2, minmax(0, 1fr)); max-width: 480px; }
.choice {
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: var(--r2);
  background: var(--bg-2);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 6px;
  text-align: left;
  font-family: var(--ui);
  color: var(--fg);
  transition: border-color var(--t-fast), background var(--t-fast);
}
.choice:hover { border-color: var(--accent); }
.choice.sel { border-color: var(--accent); background: var(--accent-2); }
.choice .t { font-size: 13px; font-weight: 600; color: var(--fg); }
.choice .d { font-size: 11.5px; color: var(--fg-3); }
.choice .preview {
  display: block;
  height: 56px;
  border-radius: var(--r1);
  border: 1px solid var(--line);
  padding: 6px 8px;
  font-family: var(--mono);
  font-size: 10.5px;
  overflow: hidden;
  line-height: 1.4;
}
.term-choice .preview { line-height: 1.5; }

.preview-terminal {
  margin-top: 18px;
  border: 1px solid var(--line);
  border-radius: var(--r2);
  overflow: hidden;
  background: var(--bg-1);
}
.preview-header {
  padding: 8px 12px;
  background: var(--bg-2);
  border-bottom: 1px solid var(--line-soft);
}
.preview-body {
  padding: 12px 16px;
  font-family: var(--mono);
  line-height: 1.5;
  min-height: 120px;
  transition: background 0.15s, color 0.15s, font-size 0.15s;
}

/* Save bar */
.save-bar {
  position: fixed;
  bottom: 0;
  left: var(--sidebar-width);
  right: 0;
  background: var(--bg-1);
  border-top: 1px solid var(--line);
  padding: 12px 28px;
  display: flex;
  align-items: center;
  gap: 12px;
  z-index: 100;
  box-shadow: 0 -4px 14px rgba(0, 0, 0, 0.12);
}
.unsaved-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--warn);
  box-shadow: 0 0 6px var(--warn);
}
.save-text { font-size: 13px; color: var(--fg-2); }

.save-bar-enter-active, .save-bar-leave-active {
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.22s ease;
}
.save-bar-enter-from, .save-bar-leave-to { transform: translateY(100%); opacity: 0; }
</style>
