<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, HighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { vim } from '@replit/codemirror-vim'
import { useSettingsStore } from '../stores/settings'
import { useTheme } from '../composables/useTheme'

import { javascript } from '@codemirror/lang-javascript'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { json } from '@codemirror/lang-json'
import { python } from '@codemirror/lang-python'
import { go } from '@codemirror/lang-go'
import { php } from '@codemirror/lang-php'
import { sql } from '@codemirror/lang-sql'
import { yaml } from '@codemirror/lang-yaml'
import { markdown } from '@codemirror/lang-markdown'
import { xml } from '@codemirror/lang-xml'
import { rust } from '@codemirror/lang-rust'
import { sass } from '@codemirror/lang-sass'
import { vue } from '@codemirror/lang-vue'

const props = defineProps<{
  modelValue: string
  language: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorEl = ref<HTMLDivElement>()
const view = shallowRef<EditorView | null>(null)
const settingsStore = useSettingsStore()
const { theme } = useTheme()
const themeCompartment = new Compartment()
const highlightCompartment = new Compartment()
const fontSizeCompartment = new Compartment()
const keymapCompartment = new Compartment()

// Read CSS variable values from document root
function getCssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function isLightTheme(): boolean {
  return document.documentElement.getAttribute('data-theme') === 'light'
}

function buildEditorTheme() {
  const bg0     = getCssVar('--bg-0') || (isLightTheme() ? '#faf7f0' : '#17140f')
  const bg2     = getCssVar('--bg-2') || (isLightTheme() ? '#f2ece0' : '#2a251c')
  const fg      = getCssVar('--fg')   || (isLightTheme() ? '#1c1810' : '#f5efe0')
  const fg3     = getCssVar('--fg-3') || '#8a8170'
  const accent  = getCssVar('--accent') || (isLightTheme() ? '#bf8138' : '#d7a965')
  const accent2 = getCssVar('--accent-2') || 'rgba(215, 169, 101, .18)'
  const line    = getCssVar('--line') || (isLightTheme() ? '#d9d0b9' : '#3d3528')
  const isLight = isLightTheme()

  return EditorView.theme({
    '&': { backgroundColor: bg0, color: fg, height: '100%' },
    '.cm-gutters': { backgroundColor: bg0, color: fg3, border: 'none', opacity: '0.5' },
    '.cm-activeLineGutter': { backgroundColor: bg2, opacity: '1' },
    '.cm-activeLine': { backgroundColor: isLight ? 'rgba(0,0,0,0.04)' : 'rgba(255,255,255,0.04)' },
    '.cm-cursor': { borderLeftColor: accent },
    '.cm-selectionBackground': { backgroundColor: `${accent2} !important` },
    '&.cm-focused .cm-selectionBackground': { backgroundColor: `${accent2} !important` },
    '.cm-matchingBracket': { backgroundColor: accent2, outline: `1px solid ${line}` },
    '.cm-line': { padding: '0 8px' },
  }, { dark: !isLight })
}

// Syntax highlighting that adapts to theme
function buildHighlightStyle() {
  const accent       = getCssVar('--accent') || '#d7a965'
  const ok           = getCssVar('--ok')     || '#7eb88a'
  const bad          = getCssVar('--bad')    || '#e07a73'
  const warn         = getCssVar('--warn')   || '#d8a85a'
  const info         = getCssVar('--info')   || '#7faecc'
  const mag          = getCssVar('--mag')    || '#b58cc8'
  const fg3          = getCssVar('--fg-3')   || '#8a8170'
  const fg           = getCssVar('--fg')     || (isLightTheme() ? '#1c1810' : '#f5efe0')

  // Warm syntax-color mapping (no GitHub colors hardcoded)
  const keyword   = bad
  const string    = info
  const number    = info
  const comment   = fg3
  const fn        = mag
  const type      = warn
  const variable  = fg
  const tag       = ok
  const attribute = info
  const literal   = info
  const operator  = bad
  const property  = info
  const meta      = accent

  return HighlightStyle.define([
    { tag: tags.keyword, color: keyword },
    { tag: tags.controlKeyword, color: keyword },
    { tag: tags.moduleKeyword, color: keyword },
    { tag: tags.operatorKeyword, color: keyword },
    { tag: tags.definitionKeyword, color: keyword },

    { tag: tags.string, color: string },
    { tag: tags.special(tags.string), color: string },
    { tag: tags.character, color: string },

    { tag: tags.number, color: number },
    { tag: tags.integer, color: number },
    { tag: tags.float, color: number },
    { tag: tags.bool, color: literal },
    { tag: tags.null, color: literal },

    { tag: tags.comment, color: comment, fontStyle: 'italic' },
    { tag: tags.lineComment, color: comment, fontStyle: 'italic' },
    { tag: tags.blockComment, color: comment, fontStyle: 'italic' },

    { tag: tags.function(tags.variableName), color: fn },
    { tag: tags.function(tags.definition(tags.variableName)), color: fn },
    { tag: tags.definition(tags.function(tags.variableName)), color: fn },

    { tag: tags.typeName, color: type },
    { tag: tags.className, color: type },
    { tag: tags.namespace, color: type },

    { tag: tags.variableName, color: variable },
    { tag: tags.definition(tags.variableName), color: variable },
    { tag: tags.propertyName, color: property },
    { tag: tags.definition(tags.propertyName), color: property },

    { tag: tags.tagName, color: tag },
    { tag: tags.attributeName, color: attribute },
    { tag: tags.attributeValue, color: string },

    { tag: tags.operator, color: operator },
    { tag: tags.punctuation, color: fg3 },
    { tag: tags.bracket, color: fg3 },
    { tag: tags.meta, color: meta },

    { tag: tags.regexp, color: info },
    { tag: tags.escape, color: warn },
    { tag: tags.link, color: info, textDecoration: 'underline' },
    { tag: tags.heading, color: accent, fontWeight: 'bold' },
    { tag: tags.emphasis, fontStyle: 'italic' },
    { tag: tags.strong, fontWeight: 'bold' },
  ])
}

function getLanguageExtension(lang: string) {
  switch (lang) {
    case 'javascript': return javascript()
    case 'typescript': return javascript({ typescript: true })
    case 'html': return html()
    case 'vue': return vue()
    case 'css': return css()
    case 'scss': return sass({ indented: false })
    case 'json': return json()
    case 'python': return python()
    case 'go': return go()
    case 'php': return php()
    case 'sql': return sql()
    case 'yaml': return yaml()
    case 'markdown': return markdown()
    case 'xml': return xml()
    case 'rust': return rust()
    default: return []
  }
}

function createState(doc: string) {
  const langExt = getLanguageExtension(props.language)
  return EditorState.create({
    doc,
    extensions: [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      history(),
      bracketMatching(),
      indentOnInput(),
      themeCompartment.of(buildEditorTheme()),
      highlightCompartment.of(syntaxHighlighting(buildHighlightStyle())),
      fontSizeCompartment.of(EditorView.theme({
        '.cm-scroller': { fontSize: settingsStore.ui.editorFontSize + 'px' },
      })),
      keymapCompartment.of(settingsStore.ui.editorKeymap === 'vim' ? vim() : []),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      ...(Array.isArray(langExt) ? langExt : [langExt]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          emit('update:modelValue', update.state.doc.toString())
        }
      }),
      EditorState.readOnly.of(props.readonly || false),
    ],
  })
}

onMounted(() => {
  if (!editorEl.value) return
  view.value = new EditorView({
    state: createState(props.modelValue),
    parent: editorEl.value,
  })
})

// Watch for external content changes (file switch, reload from disk)
watch(() => props.modelValue, (newVal) => {
  if (view.value && view.value.state.doc.toString() !== newVal) {
    view.value.dispatch({
      changes: { from: 0, to: view.value.state.doc.length, insert: newVal },
    })
  }
})

// Watch language changes (when switching between files)
watch(() => props.language, () => {
  if (view.value) {
    const doc = view.value.state.doc.toString()
    view.value.destroy()
    view.value = new EditorView({
      state: createState(doc),
      parent: editorEl.value!,
    })
  }
})

// Watch warm dark/light theme changes — reconfigure editor theme + syntax colors
watch(theme, () => {
  if (!view.value) return
  requestAnimationFrame(() => {
    view.value!.dispatch({
      effects: [
        themeCompartment.reconfigure(buildEditorTheme()),
        highlightCompartment.reconfigure(syntaxHighlighting(buildHighlightStyle())),
      ],
    })
  })
})

watch(() => settingsStore.ui.editorKeymap, (keymap) => {
  if (!view.value) return
  view.value.dispatch({
    effects: keymapCompartment.reconfigure(keymap === 'vim' ? vim() : []),
  })
})

watch(() => settingsStore.ui.editorFontSize, (size) => {
  if (!view.value) return
  view.value.dispatch({
    effects: fontSizeCompartment.reconfigure(EditorView.theme({
      '.cm-scroller': { fontSize: size + 'px' },
    })),
  })
})

defineExpose({
  getScrollDom: () => view.value?.scrollDOM ?? null,
  getLineHeight: () => view.value?.defaultLineHeight ?? 20.8,
})

onBeforeUnmount(() => {
  view.value?.destroy()
  view.value = null
})
</script>

<template>
  <div ref="editorEl" class="code-editor"></div>
</template>

<style scoped>
.code-editor {
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: var(--bg-0);
}
.code-editor :deep(.cm-editor) {
  height: 100%;
}
.code-editor :deep(.cm-scroller) {
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.6;
}
</style>
