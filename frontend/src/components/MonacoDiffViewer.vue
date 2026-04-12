<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'

if (!self.MonacoEnvironment) {
  self.MonacoEnvironment = {
    getWorker(_, label) {
      if (label === 'typescript' || label === 'javascript') return new tsWorker()
      if (label === 'json') return new jsonWorker()
      if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
      if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
      return new editorWorker()
    },
  }
}

import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { useSettingsStore } from '../stores/settings'
import { buildMonacoTheme, monacoThemeId } from '../data/monaco-themes'
import { toMonacoLanguage } from '../data/monaco-languages'

const props = withDefaults(defineProps<{
  original: string
  modified: string
  language: string
  filename?: string
  inline?: boolean
}>(), {
  inline: false,
})

const editorEl = ref<HTMLDivElement>()
const diffEditor = shallowRef<monaco.editor.IStandaloneDiffEditor | null>(null)
const originalModel = shallowRef<monaco.editor.ITextModel | null>(null)
const modifiedModel = shallowRef<monaco.editor.ITextModel | null>(null)
const settingsStore = useSettingsStore()
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
  const lang = toMonacoLanguage(props.language)

  originalModel.value = monaco.editor.createModel(props.original, lang)
  modifiedModel.value = monaco.editor.createModel(props.modified, lang)

  diffEditor.value = monaco.editor.createDiffEditor(editorEl.value, {
    theme: themeId,
    automaticLayout: true,
    readOnly: true,
    renderSideBySide: !props.inline,
    fontSize: settingsStore.ui.editorFontSize,
    fontFamily: settingsStore.ui.fontFamily,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    overviewRulerBorder: false,
    renderOverviewRuler: false,
    originalEditable: false,
  })

  diffEditor.value.setModel({
    original: originalModel.value,
    modified: modifiedModel.value,
  })
})

watch(() => props.original, (val) => {
  if (originalModel.value && originalModel.value.getValue() !== val) {
    originalModel.value.setValue(val)
  }
})

watch(() => props.modified, (val) => {
  if (modifiedModel.value && modifiedModel.value.getValue() !== val) {
    modifiedModel.value.setValue(val)
  }
})

watch(() => props.language, (lang) => {
  const monacoLang = toMonacoLanguage(lang)
  if (originalModel.value) monaco.editor.setModelLanguage(originalModel.value, monacoLang)
  if (modifiedModel.value) monaco.editor.setModelLanguage(modifiedModel.value, monacoLang)
})

watch(() => props.inline, (inline) => {
  diffEditor.value?.updateOptions({ renderSideBySide: !inline })
})

watch(() => settingsStore.ui.siteThemeName, (themeName) => {
  const id = monacoThemeId(themeName)
  monaco.editor.defineTheme(id, buildMonacoTheme(themeName) as monaco.editor.IStandaloneThemeData)
  registeredThemes.add(id)
  monaco.editor.setTheme(id)
})

watch(() => settingsStore.ui.editorFontSize, (fontSize) => {
  diffEditor.value?.updateOptions({ fontSize })
})

onBeforeUnmount(() => {
  diffEditor.value?.dispose()
  originalModel.value?.dispose()
  modifiedModel.value?.dispose()
  diffEditor.value = null
  originalModel.value = null
  modifiedModel.value = null
})
</script>

<template>
  <div ref="editorEl" class="monaco-diff-wrapper"></div>
</template>

<style scoped>
.monaco-diff-wrapper {
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
}
</style>
