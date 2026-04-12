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
let isUpdating = false
const registeredThemes = new Set<string>()

function ensureThemeRegistered(themeName: string): string {
  const id = monacoThemeId(themeName)
  if (!registeredThemes.has(id)) {
    monaco.editor.defineTheme(id, buildMonacoTheme(themeName) as monaco.editor.IStandaloneThemeData)
    registeredThemes.add(id)
  }
  return id
}

onMounted(() => {
  if (!editorEl.value) return

  const themeName = settingsStore.ui.siteThemeName
  const themeId = ensureThemeRegistered(themeName)

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

// Watch site theme changes — re-define and apply theme
watch(() => settingsStore.ui.siteThemeName, (themeName) => {
  if (!editor.value) return
  // Re-register to pick up any CSS var changes
  const id = monacoThemeId(themeName)
  monaco.editor.defineTheme(id, buildMonacoTheme(themeName) as monaco.editor.IStandaloneThemeData)
  registeredThemes.add(id)
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
  background: var(--bg-primary);
}
</style>
