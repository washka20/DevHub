<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'

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

const devhubTheme = EditorView.theme({
  '&': { backgroundColor: '#0d1117', color: '#f0f6fc', height: '100%' },
  '.cm-gutters': { backgroundColor: '#0d1117', color: '#484f58', border: 'none' },
  '.cm-activeLineGutter': { backgroundColor: '#161b22' },
  '.cm-activeLine': { backgroundColor: 'rgba(88,166,255,0.06)' },
  '.cm-cursor': { borderLeftColor: '#58a6ff' },
  '.cm-selectionBackground': { backgroundColor: 'rgba(88,166,255,0.2) !important' },
  '&.cm-focused .cm-selectionBackground': { backgroundColor: 'rgba(88,166,255,0.3) !important' },
  '.cm-matchingBracket': { backgroundColor: 'rgba(88,166,255,0.2)', outline: '1px solid rgba(88,166,255,0.5)' },
}, { dark: true })

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
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      devhubTheme,
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

watch(() => props.modelValue, (newVal) => {
  if (view.value && view.value.state.doc.toString() !== newVal) {
    view.value.dispatch({
      changes: { from: 0, to: view.value.state.doc.length, insert: newVal },
    })
  }
})

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
