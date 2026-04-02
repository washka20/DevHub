<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, HighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { useSettingsStore } from '../stores/settings'

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
const themeCompartment = new Compartment()
const highlightCompartment = new Compartment()
const fontSizeCompartment = new Compartment()

// Read CSS variable values from document root
function getCssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function buildEditorTheme() {
  const bgPrimary = getCssVar('--bg-primary') || '#0d1117'
  const bgSecondary = getCssVar('--bg-secondary') || '#161b22'
  const textPrimary = getCssVar('--text-primary') || '#f0f6fc'
  const textSecondary = getCssVar('--text-secondary') || '#8b949e'
  const accentBlue = getCssVar('--accent-blue') || '#58a6ff'
  const border = getCssVar('--border') || '#30363d'

  // Detect light theme
  const isLight = isLightColor(bgPrimary)

  return EditorView.theme({
    '&': { backgroundColor: bgPrimary, color: textPrimary, height: '100%' },
    '.cm-gutters': { backgroundColor: bgPrimary, color: textSecondary, border: 'none', opacity: '0.5' },
    '.cm-activeLineGutter': { backgroundColor: bgSecondary, opacity: '1' },
    '.cm-activeLine': { backgroundColor: isLight ? 'rgba(0,0,0,0.04)' : 'rgba(255,255,255,0.04)' },
    '.cm-cursor': { borderLeftColor: accentBlue },
    '.cm-selectionBackground': { backgroundColor: isLight ? 'rgba(9,105,218,0.15) !important' : 'rgba(88,166,255,0.2) !important' },
    '&.cm-focused .cm-selectionBackground': { backgroundColor: isLight ? 'rgba(9,105,218,0.25) !important' : 'rgba(88,166,255,0.3) !important' },
    '.cm-matchingBracket': { backgroundColor: isLight ? 'rgba(9,105,218,0.15)' : 'rgba(88,166,255,0.2)', outline: `1px solid ${border}` },
    '.cm-line': { padding: '0 8px' },
  }, { dark: !isLight })
}

function isLightColor(hex: string): boolean {
  hex = hex.replace('#', '')
  if (hex.length === 3) hex = hex.split('').map(c => c + c).join('')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return (r * 299 + g * 587 + b * 114) / 1000 > 128
}

// Syntax highlighting that adapts to theme
function buildHighlightStyle() {
  const bgPrimary = getCssVar('--bg-primary') || '#0d1117'
  const accentBlue = getCssVar('--accent-blue') || '#58a6ff'
  const accentGreen = getCssVar('--accent-green') || '#3fb950'
  const accentRed = getCssVar('--accent-red') || '#f85149'
  const accentOrange = getCssVar('--accent-orange') || '#d29922'
  const accentPurple = getCssVar('--accent-purple') || '#bc8cff'
  const textSecondary = getCssVar('--text-secondary') || '#8b949e'
  const textPrimary = getCssVar('--text-primary') || '#f0f6fc'
  const isLight = isLightColor(bgPrimary)

  // GitHub-style syntax colors adapted for each theme
  const keyword = accentRed
  const string = isLight ? '#0a3069' : '#a5d6ff'
  const number = accentBlue
  const comment = textSecondary
  const fn = accentPurple
  const type = accentOrange
  const variable = textPrimary
  const tag = accentGreen
  const attribute = accentBlue
  const literal = accentBlue
  const operator = isLight ? '#cf222e' : '#ff7b72'
  const property = accentBlue

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
    { tag: tags.punctuation, color: textSecondary },
    { tag: tags.bracket, color: textSecondary },
    { tag: tags.meta, color: textSecondary },

    { tag: tags.regexp, color: accentBlue },
    { tag: tags.escape, color: accentOrange },
    { tag: tags.link, color: accentBlue, textDecoration: 'underline' },
    { tag: tags.heading, color: accentBlue, fontWeight: 'bold' },
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

// Watch site theme changes — reconfigure editor theme + syntax colors
watch(() => settingsStore.ui.siteThemeName, () => {
  if (!view.value) return
  // Wait for CSS vars to update
  requestAnimationFrame(() => {
    view.value!.dispatch({
      effects: [
        themeCompartment.reconfigure(buildEditorTheme()),
        highlightCompartment.reconfigure(syntaxHighlighting(buildHighlightStyle())),
      ],
    })
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
  background: var(--bg-primary);
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
