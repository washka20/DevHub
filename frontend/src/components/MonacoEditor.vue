<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'

self.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'typescript' || label === 'javascript') return new tsWorker()
    if (label === 'json') return new jsonWorker()
    if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
    return new editorWorker()
  },
}

import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { useSettingsStore } from '../stores/settings'
import { useTheme } from '../composables/useTheme'
import { buildMonacoTheme, monacoThemeId } from '../data/monaco-themes'
import { toMonacoLanguage } from '../data/monaco-languages'

const props = defineProps<{
  modelValue: string
  language: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorEl = ref<HTMLDivElement>()
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
const settingsStore = useSettingsStore()
const { theme } = useTheme()
let isUpdating = false

function applyTheme(mode: 'dark' | 'light'): string {
  const id = monacoThemeId(mode)
  monaco.editor.defineTheme(id, buildMonacoTheme(mode) as monaco.editor.IStandaloneThemeData)
  return id
}

onMounted(() => {
  if (!editorEl.value) return

  const themeId = applyTheme(theme.value)

  editor.value = monaco.editor.create(editorEl.value, {
    value: props.modelValue,
    language: toMonacoLanguage(props.language),
    theme: themeId,
    automaticLayout: true,
    minimap: { enabled: settingsStore.ui.editorMinimap },
    fontSize: settingsStore.ui.editorFontSize,
    fontFamily: settingsStore.ui.fontFamily,
    scrollBeyondLastLine: false,
    renderLineHighlight: 'line',
    matchBrackets: 'always',
    bracketPairColorization: { enabled: true },
    guides: { bracketPairs: true },
    overviewRulerBorder: false,
    readOnly: props.readonly || false,
  })

  editor.value.onDidChangeModelContent(() => {
    if (isUpdating) return
    const value = editor.value!.getValue()
    emit('update:modelValue', value)
  })
})

// Watch external content changes with feedback loop guard
watch(() => props.modelValue, (newVal) => {
  if (!editor.value) return
  if (editor.value.getValue() !== newVal) {
    isUpdating = true
    editor.value.setValue(newVal)
    isUpdating = false
  }
})

// Watch language changes
watch(() => props.language, (lang) => {
  if (!editor.value) return
  const model = editor.value.getModel()
  if (model) {
    monaco.editor.setModelLanguage(model, toMonacoLanguage(lang))
  }
})

// Watch warm dark/light theme changes — rebuild + apply Monaco theme
watch(theme, (mode) => {
  if (!editor.value) return
  const id = applyTheme(mode)
  monaco.editor.setTheme(id)
})

// Watch minimap setting
watch(() => settingsStore.ui.editorMinimap, (enabled) => {
  if (!editor.value) return
  editor.value.updateOptions({ minimap: { enabled } })
})

// Watch font size setting
watch(() => settingsStore.ui.editorFontSize, (fontSize) => {
  if (!editor.value) return
  editor.value.updateOptions({ fontSize })
})

defineExpose({
  getScrollDom: () => editor.value?.getDomNode()?.querySelector('.monaco-scrollable-element') as HTMLElement | null,
  getLineHeight: () => editor.value?.getOption(monaco.editor.EditorOption.lineHeight) ?? 20,
})

onBeforeUnmount(() => {
  editor.value?.dispose()
  editor.value = null
})
</script>

<template>
  <div ref="editorEl" class="monaco-editor-wrapper"></div>
</template>

<style scoped>
.monaco-editor-wrapper {
  width: 100%;
  height: 100%;
  background: var(--bg-0);
}
</style>
