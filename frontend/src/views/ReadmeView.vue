<script setup lang="ts">
import { ref, watch, computed, reactive, nextTick } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { markdownApi } from '../api/markdown'
import FileTreeNode from '../components/FileTreeNode.vue'
import type { FileNode } from '../components/FileTreeNode.vue'
import MarkdownIt from 'markdown-it'
import markdownItAnchor from 'markdown-it-anchor'
import markdownItFootnote from 'markdown-it-footnote'
import hljs from 'highlight.js/lib/core'
import langGo from 'highlight.js/lib/languages/go'
import langTypeScript from 'highlight.js/lib/languages/typescript'
import langJavaScript from 'highlight.js/lib/languages/javascript'
import langBash from 'highlight.js/lib/languages/bash'
import langYaml from 'highlight.js/lib/languages/yaml'
import langJson from 'highlight.js/lib/languages/json'
import langCss from 'highlight.js/lib/languages/css'
import langSql from 'highlight.js/lib/languages/sql'
import langPython from 'highlight.js/lib/languages/python'
import langXml from 'highlight.js/lib/languages/xml'
import langRust from 'highlight.js/lib/languages/rust'
import langDiff from 'highlight.js/lib/languages/diff'
import langDockerfile from 'highlight.js/lib/languages/dockerfile'
import langMarkdown from 'highlight.js/lib/languages/markdown'
import mermaid from 'mermaid'

hljs.registerLanguage('go', langGo)
hljs.registerLanguage('typescript', langTypeScript)
hljs.registerLanguage('ts', langTypeScript)
hljs.registerLanguage('javascript', langJavaScript)
hljs.registerLanguage('js', langJavaScript)
hljs.registerLanguage('bash', langBash)
hljs.registerLanguage('sh', langBash)
hljs.registerLanguage('shell', langBash)
hljs.registerLanguage('yaml', langYaml)
hljs.registerLanguage('yml', langYaml)
hljs.registerLanguage('json', langJson)
hljs.registerLanguage('css', langCss)
hljs.registerLanguage('sql', langSql)
hljs.registerLanguage('python', langPython)
hljs.registerLanguage('py', langPython)
hljs.registerLanguage('xml', langXml)
hljs.registerLanguage('html', langXml)
hljs.registerLanguage('vue', langXml)
hljs.registerLanguage('rust', langRust)
hljs.registerLanguage('rs', langRust)
hljs.registerLanguage('diff', langDiff)
hljs.registerLanguage('dockerfile', langDockerfile)
hljs.registerLanguage('docker', langDockerfile)
hljs.registerLanguage('markdown', langMarkdown)
hljs.registerLanguage('md', langMarkdown)

mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  themeVariables: {
    primaryColor: '#58a6ff',
    primaryBorderColor: '#30363d',
    primaryTextColor: '#f0f6fc',
    lineColor: '#8b949e',
    secondaryColor: '#1c2128',
    tertiaryColor: '#161b22',
  },
})

const projectsStore = useProjectsStore()
const currentProject = computed(() => projectsStore.currentProject)

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  highlight: (str: string, lang: string): string => {
    // Mermaid diagrams — render as special container
    if (lang === 'mermaid') {
      return `<div class="mermaid-container"><pre class="mermaid">${str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')}</pre></div>`
    }
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang }).value}</code></pre>`
      } catch { /* fallthrough */ }
    }
    const escaped = str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
    return `<pre class="hljs"><code>${escaped}</code></pre>`
  },
})

// Task list plugin: renders - [ ] / - [x] as interactive checkboxes with data-line
// eslint-disable-next-line @typescript-eslint/no-explicit-any
md.core.ruler.after('inline', 'task-lists', (state: any) => {
  const tokens = state.tokens
  for (let i = 0; i < tokens.length; i++) {
    if (tokens[i].type !== 'inline') continue
    const content = tokens[i].content
    const match = content.match(/^\[([ xX])\]\s/)
    if (!match) continue

    // Find the source line (markdown-it map gives [startLine, endLine])
    let sourceLine = -1
    // Walk up to find parent token with map
    for (let j = i; j >= 0; j--) {
      if (tokens[j].map) {
        sourceLine = tokens[j].map![0] + 1 // 1-based
        break
      }
    }

    const checked = match[1].toLowerCase() === 'x'
    const checkbox = `<input type="checkbox" data-line="${sourceLine}" ${checked ? 'checked' : ''} class="md-checkbox">`
    tokens[i].content = content.slice(match[0].length)
    tokens[i].children = md.parseInline(tokens[i].content, state.env)[0]?.children || []

    // Insert checkbox HTML before the inline content
    const checkboxToken = new state.Token('html_inline', '', 0)
    checkboxToken.content = checkbox
    if (tokens[i].children) {
      tokens[i].children!.unshift(checkboxToken)
    }

    // Mark parent <li> with a class
    if (i >= 2 && tokens[i - 2].type === 'list_item_open') {
      tokens[i - 2].attrJoin('class', 'task-list-item')
    }
    // Mark grandparent <ul> with a class
    for (let j = i - 2; j >= 0; j--) {
      if (tokens[j].type === 'bullet_list_open') {
        tokens[j].attrJoin('class', 'task-list')
        break
      }
    }
  }
})

// Alert/admonition blocks: > [!NOTE], > [!WARNING], etc.
const alertTypes: Record<string, { icon: string; label: string }> = {
  NOTE: { icon: 'ℹ️', label: 'Note' },
  TIP: { icon: '💡', label: 'Tip' },
  IMPORTANT: { icon: '❗', label: 'Important' },
  WARNING: { icon: '⚠️', label: 'Warning' },
  CAUTION: { icon: '🔴', label: 'Caution' },
}

md.core.ruler.after('block', 'alerts', (state: any) => {
  const tokens = state.tokens
  for (let i = 0; i < tokens.length - 2; i++) {
    if (tokens[i].type !== 'blockquote_open') continue
    let inlineIdx = -1
    for (let j = i + 1; j < tokens.length && tokens[j].type !== 'blockquote_close'; j++) {
      if (tokens[j].type === 'inline') { inlineIdx = j; break }
    }
    if (inlineIdx === -1) continue
    const inlineContent = tokens[inlineIdx].content
    const match = inlineContent.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*\n?/)
    if (!match) continue
    const type = match[1]
    const alert = alertTypes[type]
    tokens[i].attrJoin('class', `alert alert-${type.toLowerCase()}`)
    tokens[inlineIdx].content = inlineContent.slice(match[0].length)
    if (tokens[inlineIdx].children) {
      tokens[inlineIdx].children = md.parseInline(tokens[inlineIdx].content, state.env)[0]?.children || []
    }
    const headerToken = new state.Token('html_inline', '', 0)
    headerToken.content = `<span class="alert-title">${alert.icon} ${alert.label}</span><br>`
    if (tokens[inlineIdx].children) {
      tokens[inlineIdx].children.unshift(headerToken)
    }
  }
})

md.use(markdownItAnchor, {
  permalink: markdownItAnchor.permalink.headerLink({
    safariReaderFix: true,
    class: 'heading-anchor',
  }),
  slugify: (s: string) => s.toLowerCase().replace(/[^\w]+/g, '-').replace(/^-|-$/g, ''),
})

md.use(markdownItFootnote)

const content = ref('')
const rawMarkdown = ref('')
const loading = ref(false)
const notFound = ref(false)
const mdFiles = ref<string[]>([])
const currentFile = ref('')
const collapsed = reactive<Record<string, boolean>>({})
const panelOpen = ref(true)

const fileTree = computed<FileNode[]>(() => {
  const root: FileNode[] = []

  for (const filePath of mdFiles.value) {
    const parts = filePath.split('/')
    let current = root

    for (let i = 0; i < parts.length; i++) {
      const name = parts[i]
      const isLast = i === parts.length - 1

      if (isLast) {
        current.push({ name, path: filePath, isDir: false, children: [] })
      } else {
        let dir = current.find(n => n.isDir && n.name === name)
        if (!dir) {
          dir = { name, path: parts.slice(0, i + 1).join('/'), isDir: true, children: [] }
          current.push(dir)
        }
        current = dir.children
      }
    }
  }

  function sortNodes(nodes: FileNode[]) {
    nodes.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
    nodes.forEach(n => { if (n.isDir) sortNodes(n.children) })
  }
  sortNodes(root)
  return root
})

const breadcrumb = computed(() => {
  if (!currentFile.value) return []
  return currentFile.value.split('/')
})

function toggleDir(path: string) {
  collapsed[path] = !collapsed[path]
}

async function fetchFileList() {
  if (!currentProject.value) return
  try {
    mdFiles.value = await markdownApi.listFiles(currentProject.value.name)
  } catch {
    mdFiles.value = []
  }
}

async function renderMermaid() {
  await nextTick()
  const containers = document.querySelectorAll('.readme-article .mermaid')
  if (containers.length === 0) return
  try {
    await mermaid.run({ nodes: containers as unknown as ArrayLike<HTMLElement> })
  } catch (e) {
    console.warn('Mermaid render error:', e)
  }
}

async function selectFile(path: string) {
  if (!currentProject.value) return
  loading.value = true
  notFound.value = false
  content.value = ''
  currentFile.value = path

  try {
    const res = await markdownApi.getFile(currentProject.value.name, path)
    const ct = res.headers.get('content-type') || ''
    if (ct.includes('text/html')) {
      notFound.value = true
      return
    }
    const text = await res.text()
    rawMarkdown.value = text
    content.value = md.render(text)
    renderMermaid()
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

function resolvePath(href: string): string {
  const dir = currentFile.value.includes('/')
    ? currentFile.value.substring(0, currentFile.value.lastIndexOf('/'))
    : ''

  if (href.startsWith('./')) {
    href = href.substring(2)
  }

  const parts = (dir ? dir + '/' + href : href).split('/')
  const resolved: string[] = []
  for (const part of parts) {
    if (part === '..') resolved.pop()
    else if (part !== '.') resolved.push(part)
  }
  return resolved.join('/')
}

function handleContentClick(e: MouseEvent) {
  const target = e.target as HTMLElement

  // Handle checkbox clicks
  if (target.tagName === 'INPUT' && target.classList.contains('md-checkbox')) {
    e.preventDefault()
    const line = parseInt(target.getAttribute('data-line') || '0')
    if (line > 0) toggleCheckbox(line)
    return
  }

  const link = target.closest('a')
  if (!link) return

  const href = link.getAttribute('href')
  if (!href) return

  if (href.endsWith('.md') && !href.startsWith('http')) {
    e.preventDefault()
    const resolved = resolvePath(href)
    if (mdFiles.value.includes(resolved)) {
      selectFile(resolved)
    }
  }
}

async function toggleCheckbox(line: number) {
  if (!currentProject.value || !currentFile.value) return
  try {
    await markdownApi.toggleCheckbox(currentProject.value.name, currentFile.value, line)

    // Toggle locally without re-fetching — no scroll reset
    const lines = rawMarkdown.value.split('\n')
    const idx = line - 1
    if (idx < lines.length) {
      const l = lines[idx]
      if (l.includes('- [ ]')) lines[idx] = l.replace('- [ ]', '- [x]')
      else if (l.includes('- [x]')) lines[idx] = l.replace('- [x]', '- [ ]')
      else if (l.includes('- [X]')) lines[idx] = l.replace('- [X]', '- [ ]')
      rawMarkdown.value = lines.join('\n')
      content.value = md.render(rawMarkdown.value)
      renderMermaid()
    }
  } catch (err) {
    console.error('Failed to toggle checkbox:', err)
  }
}

async function init() {
  await fetchFileList()
  const readme = mdFiles.value.find(f => /^readme\.md$/i.test(f))
  if (readme) {
    selectFile(readme)
  } else if (mdFiles.value.length > 0) {
    selectFile(mdFiles.value[0])
  } else {
    notFound.value = true
    loading.value = false
  }
}

watch(() => currentProject.value?.name, () => init(), { immediate: true })
</script>

<template>
  <div class="readme-view">
    <!-- Toolbar -->
    <div class="readme-toolbar">
      <button
        v-if="mdFiles.length > 1"
        class="toolbar-toggle"
        :class="{ active: panelOpen }"
        @click="panelOpen = !panelOpen"
        title="Файлы проекта"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M1.75 1A1.75 1.75 0 0 0 0 2.75v10.5C0 14.216.784 15 1.75 15h12.5A1.75 1.75 0 0 0 16 13.25v-8.5A1.75 1.75 0 0 0 14.25 3H7.5a.25.25 0 0 1-.2-.1l-.9-1.2C6.07 1.26 5.55 1 5 1H1.75z"/>
        </svg>
      </button>

      <div class="toolbar-breadcrumb">
        <svg class="breadcrumb-icon" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
          <path d="M0 1.75A.75.75 0 0 1 .75 1h4.253c1.227 0 2.317.59 3 1.501A3.743 3.743 0 0 1 11.006 1h4.245a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75h-4.507a2.25 2.25 0 0 0-1.591.659l-.622.621a.75.75 0 0 1-1.06 0l-.622-.621A2.25 2.25 0 0 0 5.258 13H.75a.75.75 0 0 1-.75-.75zm7.251 10.324l.004-5.073-.002-2.253A2.25 2.25 0 0 0 5.003 2.5H1.5v9h3.757a3.75 3.75 0 0 1 1.994.574zM8.755 4.75l-.004 7.322a3.752 3.752 0 0 1 1.992-.572H14.5v-9h-3.495a2.25 2.25 0 0 0-2.25 2.25z"/>
        </svg>
        <template v-for="(segment, i) in breadcrumb" :key="i">
          <span class="breadcrumb-sep" v-if="i > 0">/</span>
          <span :class="['breadcrumb-segment', { 'is-file': i === breadcrumb.length - 1 }]">{{ segment }}</span>
        </template>
      </div>

      <span class="toolbar-count" v-if="mdFiles.length > 0">{{ mdFiles.length }} md</span>
    </div>

    <!-- Content area -->
    <div class="readme-body">
      <!-- File panel (left) -->
      <aside v-if="mdFiles.length > 1 && panelOpen" class="file-panel">
        <div class="file-tree">
          <FileTreeNode
            :nodes="fileTree"
            :current-file="currentFile"
            :collapsed="collapsed"
            @select="selectFile"
            @toggle="toggleDir"
          />
        </div>
      </aside>

      <!-- Main content -->
      <div class="readme-main">
        <div v-if="loading" class="readme-loading">
          <div class="spinner"></div>
        </div>

        <div v-else-if="notFound && mdFiles.length === 0" class="readme-empty">
          <svg width="40" height="40" viewBox="0 0 16 16" fill="currentColor" opacity="0.2">
            <path d="M0 1.75A.75.75 0 0 1 .75 1h4.253c1.227 0 2.317.59 3 1.501A3.743 3.743 0 0 1 11.006 1h4.245a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75h-4.507a2.25 2.25 0 0 0-1.591.659l-.622.621a.75.75 0 0 1-1.06 0l-.622-.621A2.25 2.25 0 0 0 5.258 13H.75a.75.75 0 0 1-.75-.75zm7.251 10.324l.004-5.073-.002-2.253A2.25 2.25 0 0 0 5.003 2.5H1.5v9h3.757a3.75 3.75 0 0 1 1.994.574zM8.755 4.75l-.004 7.322a3.752 3.752 0 0 1 1.992-.572H14.5v-9h-3.495a2.25 2.25 0 0 0-2.25 2.25z"/>
          </svg>
          <p>Нет markdown файлов</p>
        </div>

        <div v-else-if="notFound" class="readme-empty">
          <p>Файл не найден</p>
        </div>

        <article v-else class="readme-article markdown-body" v-html="content" @click="handleContentClick"></article>
      </div>
    </div>
  </div>
</template>

<style scoped>
.readme-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px -32px;
}

/* Toolbar */
.readme-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  height: 40px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border);
  background: var(--bg-secondary);
}

.toolbar-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.15s, color 0.15s;
}

.toolbar-toggle:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.toolbar-toggle.active {
  color: var(--accent-blue);
}

.toolbar-breadcrumb {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  min-width: 0;
  overflow: hidden;
}

.breadcrumb-icon {
  flex-shrink: 0;
  color: var(--text-secondary);
  opacity: 0.5;
}

.breadcrumb-sep {
  color: var(--text-secondary);
  opacity: 0.4;
}

.breadcrumb-segment {
  color: var(--text-secondary);
  white-space: nowrap;
}

.breadcrumb-segment.is-file {
  color: var(--text-primary);
}

.toolbar-count {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.6;
  white-space: nowrap;
}

/* Body */
.readme-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

/* File panel */
.file-panel {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  background: var(--bg-secondary);
  overflow-y: auto;
  padding: 8px 0;
}

.file-tree {
  display: flex;
  flex-direction: column;
}

/* Main content */
.readme-main {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  display: flex;
  justify-content: center;
}

.readme-article {
  width: 100%;
  max-width: 860px;
  padding: 32px 40px;
}

.readme-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  width: 100%;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.readme-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 80px 0;
  width: 100%;
  color: var(--text-secondary);
}

.readme-empty p {
  margin: 0;
  font-size: 13px;
}
</style>

<style>
/* Tree styles (global for recursive component) */
.tree-children {
  padding-left: 20px;
  position: relative;
}

.tree-children::before {
  content: '';
  position: absolute;
  left: 17px;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--border);
  opacity: 0.5;
}

.tree-dir,
.tree-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 12px;
  border: none;
  background: none;
  color: var(--text-secondary);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  border-radius: 0;
  transition: background 0.1s, color 0.1s;
  width: 100%;
  position: relative;
}

.tree-dir:hover,
.tree-file:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.tree-dir {
  font-weight: 500;
}

.tree-file {
  padding-left: 30px;
}

.tree-file.active {
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
}

.tree-chevron {
  flex-shrink: 0;
  opacity: 0.35;
  transition: transform 0.15s;
}

.tree-chevron.collapsed {
  transform: rotate(-90deg);
}

.tree-folder-icon {
  flex-shrink: 0;
  opacity: 0.6;
  color: var(--accent-orange);
}

.tree-file-icon {
  flex-shrink: 0;
  opacity: 0.35;
}

.tree-file.active .tree-file-icon {
  opacity: 0.8;
}

.tree-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Markdown styles */
.markdown-body {
  color: var(--text-primary);
  font-size: 15px;
  line-height: 1.7;
  word-wrap: break-word;
}

.markdown-body > *:first-child {
  margin-top: 0 !important;
}

.markdown-body h1,
.markdown-body h2,
.markdown-body h3,
.markdown-body h4,
.markdown-body h5,
.markdown-body h6 {
  margin-top: 28px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--text-primary);
  position: relative;
}

.markdown-body h1 {
  font-size: 1.8em;
  padding-bottom: 0.3em;
  border-bottom: 1px solid var(--border);
}

.markdown-body h2 {
  font-size: 1.4em;
  padding-bottom: 0.25em;
  border-bottom: 1px solid var(--border);
}

.markdown-body h3 { font-size: 1.2em; }
.markdown-body h4 { font-size: 1em; }

.markdown-body p {
  margin-top: 0;
  margin-bottom: 16px;
}

.markdown-body a {
  color: var(--accent-blue);
  text-decoration: none;
}

.markdown-body a:hover {
  text-decoration: underline;
}

.markdown-body ul,
.markdown-body ol {
  padding-left: 2em;
  margin-bottom: 16px;
}

.markdown-body li + li {
  margin-top: 4px;
}

.markdown-body code {
  padding: 0.2em 0.4em;
  margin: 0;
  font-size: 85%;
  background: var(--bg-tertiary);
  border-radius: 4px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.markdown-body pre:not(.hljs) {
  padding: 16px;
  overflow: auto;
  font-size: 13px;
  line-height: 1.5;
  background: var(--bg-tertiary);
  border-radius: 6px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
}

.markdown-body pre:not(.hljs) code {
  padding: 0;
  margin: 0;
  background: transparent;
  border-radius: 0;
  font-size: 100%;
}

.markdown-body blockquote {
  padding: 0 1em;
  color: var(--text-secondary);
  border-left: 3px solid var(--accent-blue);
  margin: 0 0 16px 0;
}

.markdown-body table {
  border-spacing: 0;
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 16px;
}

.markdown-body table th,
.markdown-body table td {
  padding: 8px 14px;
  border: 1px solid var(--border);
}

.markdown-body table th {
  font-weight: 600;
  background: var(--bg-secondary);
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.markdown-body table tr:nth-child(2n) {
  background: rgba(22, 27, 34, 0.5);
}

.markdown-body hr {
  height: 1px;
  padding: 0;
  margin: 28px 0;
  background: var(--border);
  border: 0;
}

.markdown-body img {
  max-width: 100%;
  border-radius: 6px;
}

.markdown-body strong {
  color: var(--text-primary);
  font-weight: 600;
}

/* Task list checkboxes (GitLab-style) */
.markdown-body ul.task-list {
  list-style: none;
  padding-left: 0;
}

.markdown-body ul.task-list ul.task-list {
  padding-left: 2em;
}

.markdown-body li.task-list-item {
  list-style: none;
  position: relative;
  padding-left: 28px;
  margin-bottom: 4px;
}

.markdown-body li.task-list-item > p {
  margin: 0;
  display: inline;
}

.markdown-body .md-checkbox {
  appearance: none;
  -webkit-appearance: none;
  width: 18px !important;
  height: 18px !important;
  max-width: 18px;
  max-height: 18px;
  min-width: 18px;
  min-height: 18px;
  box-sizing: border-box;
  border: 2px solid var(--border);
  border-radius: 4px;
  background: var(--bg-primary);
  cursor: pointer;
  position: absolute;
  left: 0;
  top: 2px;
  margin: 0;
  padding: 0;
  transition: background 0.15s, border-color 0.15s;
}

.markdown-body .md-checkbox:hover {
  border-color: var(--accent-blue);
}

.markdown-body .md-checkbox:checked {
  background: var(--accent-blue);
  border-color: var(--accent-blue);
}

.markdown-body .md-checkbox:checked::after {
  content: '';
  position: absolute;
  top: 1px;
  left: 5px;
  width: 4px;
  height: 8px;
  border: solid var(--bg-primary);
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

/* Syntax highlighting (highlight.js tokens) */
.markdown-body pre.hljs {
  padding: 16px;
  font-size: 13px;
  line-height: 1.5;
  background: var(--bg-tertiary);
  border-radius: 6px;
  border: 1px solid var(--border);
  margin-bottom: 16px;
  overflow-x: auto;
}

.markdown-body pre.hljs code {
  padding: 0;
  background: transparent;
  font-size: 100%;
  font-family: var(--font-mono);
  color: var(--text-primary);
}

.hljs-comment,
.hljs-quote { color: var(--text-secondary); font-style: italic; }

.hljs-keyword,
.hljs-selector-tag { color: #ff7b72; }

.hljs-string,
.hljs-doctag,
.hljs-regexp { color: #a5d6ff; }

.hljs-number,
.hljs-literal,
.hljs-variable,
.hljs-template-variable { color: #79c0ff; }

.hljs-title,
.hljs-section,
.hljs-name { color: #d2a8ff; }

.hljs-type,
.hljs-built_in { color: #ffa657; }

.hljs-attr,
.hljs-attribute { color: #79c0ff; }

.hljs-symbol,
.hljs-bullet,
.hljs-link { color: #a5d6ff; }

.hljs-meta { color: #79c0ff; }

.hljs-deletion { color: #ffa198; background: rgba(248, 81, 73, 0.1); }
.hljs-addition { color: #7ee787; background: rgba(63, 185, 80, 0.1); }

.hljs-emphasis { font-style: italic; }
.hljs-strong { font-weight: 700; }

/* Alert/admonition blocks */
.markdown-body blockquote.alert {
  border-left-width: 3px;
  border-radius: 6px;
  padding: 12px 16px;
  margin: 0 0 16px 0;
}

.markdown-body .alert-title {
  font-weight: 600;
  font-size: 14px;
  display: block;
  margin-bottom: 4px;
}

.markdown-body blockquote.alert-note {
  border-left-color: var(--accent-blue);
  background: color-mix(in srgb, var(--accent-blue) 8%, transparent);
}
.markdown-body blockquote.alert-note .alert-title { color: var(--accent-blue); }

.markdown-body blockquote.alert-tip {
  border-left-color: var(--accent-green);
  background: color-mix(in srgb, var(--accent-green) 8%, transparent);
}
.markdown-body blockquote.alert-tip .alert-title { color: var(--accent-green); }

.markdown-body blockquote.alert-important {
  border-left-color: var(--accent-purple);
  background: color-mix(in srgb, var(--accent-purple) 8%, transparent);
}
.markdown-body blockquote.alert-important .alert-title { color: var(--accent-purple); }

.markdown-body blockquote.alert-warning {
  border-left-color: var(--accent-orange);
  background: color-mix(in srgb, var(--accent-orange) 8%, transparent);
}
.markdown-body blockquote.alert-warning .alert-title { color: var(--accent-orange); }

.markdown-body blockquote.alert-caution {
  border-left-color: var(--accent-red);
  background: color-mix(in srgb, var(--accent-red) 8%, transparent);
}
.markdown-body blockquote.alert-caution .alert-title { color: var(--accent-red); }

/* Heading anchor links */
.markdown-body .heading-anchor {
  text-decoration: none;
  color: inherit;
}

.markdown-body .heading-anchor::before {
  content: '#';
  position: absolute;
  left: -1.5em;
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity 0.15s;
  font-weight: 400;
}

.markdown-body h1:hover .heading-anchor::before,
.markdown-body h2:hover .heading-anchor::before,
.markdown-body h3:hover .heading-anchor::before,
.markdown-body h4:hover .heading-anchor::before,
.markdown-body h5:hover .heading-anchor::before,
.markdown-body h6:hover .heading-anchor::before {
  opacity: 1;
}

/* Collapsible sections */
.markdown-body details {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0;
  margin-bottom: 16px;
}

.markdown-body details summary {
  padding: 10px 16px;
  font-weight: 600;
  cursor: pointer;
  color: var(--text-primary);
  background: var(--bg-secondary);
  border-radius: 6px;
  list-style: none;
}

.markdown-body details summary::-webkit-details-marker { display: none; }

.markdown-body details summary::before {
  content: '▶';
  display: inline-block;
  margin-right: 8px;
  font-size: 12px;
  transition: transform 0.15s;
  color: var(--text-secondary);
}

.markdown-body details[open] summary::before { transform: rotate(90deg); }

.markdown-body details[open] summary {
  border-bottom: 1px solid var(--border);
  border-radius: 6px 6px 0 0;
}

.markdown-body details > *:not(summary) { padding: 0 16px; }
.markdown-body details > p:last-child { margin-bottom: 16px; }

/* Footnotes */
.markdown-body .footnotes-sep {
  border: none;
  border-top: 1px solid var(--border);
  margin: 32px 0 16px;
}

.markdown-body .footnotes {
  font-size: 13px;
  color: var(--text-secondary);
}

.markdown-body .footnote-ref a,
.markdown-body .footnote-backref {
  color: var(--accent-blue);
  text-decoration: none;
}

.markdown-body .footnote-ref a:hover,
.markdown-body .footnote-backref:hover {
  text-decoration: underline;
}

/* Mermaid diagrams */
.markdown-body .mermaid-container {
  margin-bottom: 16px;
  overflow-x: auto;
}

.markdown-body .mermaid-container pre.mermaid {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 16px;
  text-align: center;
}

.markdown-body .mermaid-container svg {
  max-width: 100%;
}
</style>
