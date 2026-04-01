<script setup lang="ts">
import { ref, watch, computed, onBeforeUnmount } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import MarkdownIt from 'markdown-it'
import TurndownService from 'turndown'
import { gfm } from 'turndown-plugin-gfm'

interface NoteItem {
  slug: string
  title: string
  updated_at: string
}

const projectsStore = useProjectsStore()
const currentProject = computed(() => projectsStore.currentProject)

const notes = ref<NoteItem[]>([])
const selectedSlug = ref<string | null>(null)
const loading = ref(false)
const saveStatus = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const newNoteTitle = ref('')
const showNewNoteInput = ref(false)
let saveTimeout: ReturnType<typeof setTimeout> | null = null

// --- Markdown conversion ---
const md = new MarkdownIt({ html: false, linkify: true, typographer: true })

const turndown = new TurndownService({
  headingStyle: 'atx',
  bulletListMarker: '-',
  codeBlockStyle: 'fenced',
})
turndown.use(gfm)

// Custom turndown rule for Tiptap task list items
turndown.addRule('taskListItem', {
  filter: (node) =>
    node.nodeName === 'LI' &&
    node.getAttribute('data-type') === 'taskItem',
  replacement: (content, node) => {
    const checked = (node as HTMLElement).getAttribute('data-checked') === 'true'
    const text = content.replace(/^\n+/, '').replace(/\n+$/, '')
    return `- [${checked ? 'x' : ' '}] ${text}\n`
  },
})

// Custom turndown rule for task list container (remove default bullet rendering)
turndown.addRule('taskList', {
  filter: (node) =>
    node.nodeName === 'UL' &&
    node.getAttribute('data-type') === 'taskList',
  replacement: (content) => '\n' + content + '\n',
})

function markdownToHtml(markdown: string): string {
  let html = md.render(markdown)

  // Convert GFM task list items to Tiptap format
  // markdown-it renders: <li>[ ] text</li> or <li>[x] text</li>
  html = html.replace(
    /<li>\s*\[([ xX])\]\s*/g,
    (_, check) => {
      const checked = check.toLowerCase() === 'x'
      return `<li data-type="taskItem" data-checked="${checked}">`
    }
  )

  // Wrap parent <ul> of task items as taskList
  html = html.replace(
    /<ul>\s*(<li data-type="taskItem")/g,
    '<ul data-type="taskList">$1'
  )

  return html
}

function htmlToMarkdown(html: string): string {
  return turndown.turndown(html)
}

// --- Tiptap Editor ---
const editor = useEditor({
  extensions: [
    StarterKit,
    TaskList,
    TaskItem.configure({ nested: true }).extend({
      addKeyboardShortcuts() {
        return {
          ...this.parent?.(),
          Enter: () => {
            return this.editor.chain()
              .splitListItem('taskItem')
              .updateAttributes('taskItem', { checked: false })
              .run()
          },
        }
      },
    }),
  ],
  editorProps: {
    attributes: { class: 'notes-editor-content' },
  },
  onUpdate: () => scheduleSave(),
})

// --- API ---
async function fetchNotes() {
  if (!currentProject.value) return
  try {
    const res = await fetch(`/api/projects/${currentProject.value.name}/notes`)
    if (res.ok) notes.value = await res.json()
  } catch {
    notes.value = []
  }
}

async function fetchNote(slug: string) {
  if (!currentProject.value) return
  loading.value = true
  try {
    const res = await fetch(`/api/projects/${currentProject.value.name}/notes/${slug}`)
    if (!res.ok) throw new Error('Not found')
    const markdown = await res.text()
    editor.value?.commands.setContent(markdownToHtml(markdown))
    selectedSlug.value = slug
  } catch {
    selectedSlug.value = null
  } finally {
    loading.value = false
  }
}

async function createNote() {
  if (!currentProject.value || !newNoteTitle.value.trim()) return
  try {
    const res = await fetch(`/api/projects/${currentProject.value.name}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: newNoteTitle.value.trim() }),
    })
    if (!res.ok) throw new Error('Failed to create')
    const data = await res.json()
    newNoteTitle.value = ''
    showNewNoteInput.value = false
    await fetchNotes()
    await fetchNote(data.slug)
  } catch (e) {
    console.error('Failed to create note:', e)
  }
}

async function saveNote() {
  if (!currentProject.value || !selectedSlug.value || !editor.value) return
  saveStatus.value = 'saving'
  try {
    const html = editor.value.getHTML()
    const markdown = htmlToMarkdown(html)
    const res = await fetch(
      `/api/projects/${currentProject.value.name}/notes/${selectedSlug.value}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: markdown }),
      }
    )
    if (!res.ok) throw new Error('Failed to save')
    saveStatus.value = 'saved'
    await fetchNotes()
    setTimeout(() => {
      if (saveStatus.value === 'saved') saveStatus.value = 'idle'
    }, 2000)
  } catch {
    saveStatus.value = 'error'
  }
}

async function deleteNote(slug: string) {
  const noteTitle = notes.value.find(n => n.slug === slug)?.title || slug
  if (!currentProject.value || !confirm(`Delete "${noteTitle}"?`)) return
  try {
    const res = await fetch(`/api/projects/${currentProject.value.name}/notes/${slug}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('Failed to delete')
    if (selectedSlug.value === slug) {
      selectedSlug.value = null
      editor.value?.commands.clearContent()
    }
    await fetchNotes()
  } catch (e) {
    console.error('Failed to delete note:', e)
  }
}

function scheduleSave() {
  if (saveTimeout) clearTimeout(saveTimeout)
  saveStatus.value = 'saving'
  saveTimeout = setTimeout(() => saveNote(), 2000)
}

async function selectNote(slug: string) {
  if (saveTimeout) {
    clearTimeout(saveTimeout)
    saveTimeout = null
    await saveNote()
  }
  await fetchNote(slug)
}

// --- Toolbar ---
function toggleBold() { editor.value?.chain().focus().toggleBold().run() }
function toggleItalic() { editor.value?.chain().focus().toggleItalic().run() }
function toggleCode() { editor.value?.chain().focus().toggleCode().run() }
function toggleCodeBlock() { editor.value?.chain().focus().toggleCodeBlock().run() }
function toggleBlockquote() { editor.value?.chain().focus().toggleBlockquote().run() }
function toggleBulletList() { editor.value?.chain().focus().toggleBulletList().run() }
function toggleOrderedList() { editor.value?.chain().focus().toggleOrderedList().run() }
function toggleTaskList() { editor.value?.chain().focus().toggleTaskList().run() }
function setHeading(level: 1 | 2 | 3) { editor.value?.chain().focus().toggleHeading({ level }).run() }

// --- Lifecycle ---
async function init() {
  await fetchNotes()
  if (notes.value.length > 0) {
    await fetchNote(notes.value[0].slug)
  } else {
    selectedSlug.value = null
    editor.value?.commands.clearContent()
  }
}

watch(() => currentProject.value?.name, () => init(), { immediate: true })

onBeforeUnmount(() => {
  if (saveTimeout) {
    clearTimeout(saveTimeout)
    saveTimeout = null
    saveNote()
  }
})

function formatDate(iso: string): string {
  const d = new Date(iso)
  const diffMs = Date.now() - d.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 7) return `${diffDays}d ago`
  return d.toLocaleDateString()
}
</script>

<template>
  <div class="notes-view">
    <!-- Left: notes list -->
    <aside class="notes-panel">
      <div class="notes-panel-header">
        <span class="notes-panel-title">Notes</span>
        <button class="btn-new-note" @click="showNewNoteInput = !showNewNoteInput" title="New note" aria-label="New note">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M7.75 2a.75.75 0 0 1 .75.75V7h4.25a.75.75 0 0 1 0 1.5H8.5v4.25a.75.75 0 0 1-1.5 0V8.5H2.75a.75.75 0 0 1 0-1.5H7V2.75A.75.75 0 0 1 7.75 2z"/>
          </svg>
        </button>
      </div>

      <div v-if="showNewNoteInput" class="new-note-form">
        <input
          v-model="newNoteTitle"
          placeholder="Note title..."
          class="new-note-input"
          @keydown.enter="createNote"
          @keydown.escape="showNewNoteInput = false"
        />
      </div>

      <div class="notes-list">
        <button
          v-for="note in notes"
          :key="note.slug"
          class="note-item"
          :class="{ active: note.slug === selectedSlug }"
          @click="selectNote(note.slug)"
        >
          <div class="note-item-title">{{ note.title }}</div>
          <div class="note-item-meta">{{ formatDate(note.updated_at) }}</div>
        </button>

        <div v-if="notes.length === 0 && !showNewNoteInput" class="notes-empty">
          <p>No notes yet</p>
          <button class="btn-create-first" @click="showNewNoteInput = true">
            Create your first note
          </button>
        </div>
      </div>
    </aside>

    <!-- Right: editor -->
    <div class="notes-editor">
      <template v-if="selectedSlug">
        <div class="editor-toolbar">
          <div class="toolbar-group">
            <button class="toolbar-btn" :class="{ active: editor?.isActive('heading', { level: 1 }) }" @click="setHeading(1)" title="Heading 1 (Ctrl+Alt+1)"><span style="font-size: 14px; font-weight: 700">H1</span></button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('heading', { level: 2 }) }" @click="setHeading(2)" title="Heading 2 (Ctrl+Alt+2)"><span style="font-size: 13px; font-weight: 600">H2</span></button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('heading', { level: 3 }) }" @click="setHeading(3)" title="Heading 3 (Ctrl+Alt+3)">H3</button>
          </div>
          <div class="toolbar-sep"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" :class="{ active: editor?.isActive('bold') }" @click="toggleBold" title="Bold (Ctrl+B)"><strong>B</strong></button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('italic') }" @click="toggleItalic" title="Italic (Ctrl+I)"><em>I</em></button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('code') }" @click="toggleCode" title="Inline Code (Ctrl+E)"><code>&lt;/&gt;</code></button>
          </div>
          <div class="toolbar-sep"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" :class="{ active: editor?.isActive('bulletList') }" @click="toggleBulletList" title="Bullet List (Ctrl+Shift+8)" aria-label="Bullet List">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M2 4a1 1 0 1 0 0-2 1 1 0 0 0 0 2zm3.75-1.5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5zm0 5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5zm0 5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5zM3 8a1 1 0 1 1-2 0 1 1 0 0 1 2 0zm-1 5a1 1 0 1 0 0-2 1 1 0 0 0 0 2z"/></svg>
            </button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('orderedList') }" @click="toggleOrderedList" title="Ordered List (Ctrl+Shift+7)" aria-label="Ordered List">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M2.003 2.5a.5.5 0 0 0-.723-.447l-1.003.5a.5.5 0 0 0 .446.895l.28-.14V6H.5a.5.5 0 0 0 0 1h2.006a.5.5 0 1 0 0-1h-.503V2.5zM5.75 2.5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5zm0 5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5zm0 5a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5z"/></svg>
            </button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('taskList') }" @click="toggleTaskList" title="Task List (Ctrl+Shift+9)" aria-label="Task List">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M2.75 1h10.5c.966 0 1.75.784 1.75 1.75v10.5A1.75 1.75 0 0 1 13.25 15H2.75A1.75 1.75 0 0 1 1 13.25V2.75C1 1.784 1.784 1 2.75 1zm0 1.5a.25.25 0 0 0-.25.25v10.5c0 .138.112.25.25.25h10.5a.25.25 0 0 0 .25-.25V2.75a.25.25 0 0 0-.25-.25H2.75zm8.03 2.72a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0l-2-2a.75.75 0 0 1 1.06-1.06L5.75 8.69l3.72-3.72a.75.75 0 0 1 1.06 0z"/></svg>
            </button>
          </div>
          <div class="toolbar-sep"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" :class="{ active: editor?.isActive('blockquote') }" @click="toggleBlockquote" title="Blockquote (Ctrl+Shift+B)" aria-label="Blockquote">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M1.75 2.5h10.5a.75.75 0 0 1 0 1.5H1.75a.75.75 0 0 1 0-1.5zm4 5h8.5a.75.75 0 0 1 0 1.5h-8.5a.75.75 0 0 1 0-1.5zm0 5h8.5a.75.75 0 0 1 0 1.5h-8.5a.75.75 0 0 1 0-1.5zM2.5 7.75v6a.75.75 0 0 1-1.5 0v-6a.75.75 0 0 1 1.5 0z"/></svg>
            </button>
            <button class="toolbar-btn" :class="{ active: editor?.isActive('codeBlock') }" @click="toggleCodeBlock" title="Code Block (Ctrl+Alt+C)">{ }</button>
          </div>

          <div class="toolbar-right">
            <span class="save-status" :class="saveStatus" role="status" aria-live="polite">
              <template v-if="saveStatus === 'saving'">&#9679; Saving...</template>
              <template v-else-if="saveStatus === 'saved'">&#9679; Saved</template>
              <template v-else-if="saveStatus === 'error'">&#9679; Error</template>
            </span>
            <div class="toolbar-sep"></div>
            <button class="toolbar-btn toolbar-btn-danger" @click="deleteNote(selectedSlug!)" title="Delete note" aria-label="Delete note">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M6.5 1.75a.25.25 0 0 1 .25-.25h2.5a.25.25 0 0 1 .25.25V3h-3V1.75zm4.5 0V3h2.25a.75.75 0 0 1 0 1.5H2.75a.75.75 0 0 1 0-1.5H5V1.75C5 .784 5.784 0 6.75 0h2.5C10.216 0 11 .784 11 1.75zM4.496 6.675a.75.75 0 1 0-1.492.15l.66 6.6A1.75 1.75 0 0 0 5.405 15h5.19c.9 0 1.652-.681 1.741-1.576l.66-6.6a.75.75 0 0 0-1.492-.149l-.66 6.6a.25.25 0 0 1-.249.225h-5.19a.25.25 0 0 1-.249-.225l-.66-6.6z"/></svg>
              <span class="delete-label">Delete</span>
            </button>
          </div>
        </div>

        <div class="editor-container">
          <EditorContent :editor="editor" />
        </div>
      </template>

      <div v-else class="editor-empty">
        <svg width="40" height="40" viewBox="0 0 16 16" fill="currentColor" opacity="0.2">
          <path d="M0 .75A.75.75 0 0 1 .75 0h4.993a.75.75 0 0 1 .53.22l.5.5a.75.75 0 0 1-.53 1.28H.75A.75.75 0 0 1 0 1.25V.75zm0 4A.75.75 0 0 1 .75 4h7.993a.75.75 0 0 1 0 1.5H.75A.75.75 0 0 1 0 4.75zm0 4A.75.75 0 0 1 .75 8h5.993a.75.75 0 0 1 0 1.5H.75A.75.75 0 0 1 0 8.75zm0 4a.75.75 0 0 1 .75-.75h3.993a.75.75 0 0 1 0 1.5H.75a.75.75 0 0 1-.75-.75z"/>
        </svg>
        <p>Select a note or create a new one</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notes-view {
  display: flex;
  height: 100%;
  margin: -16px -32px;
}

/* --- Left panel --- */
.notes-panel {
  width: 260px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
}

.notes-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.notes-panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.btn-new-note {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
}

.btn-new-note:hover {
  background: var(--bg-tertiary);
  color: var(--accent-blue);
}

.new-note-form {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
}

.new-note-input {
  width: 100%;
  padding: 6px 10px;
  font-size: 13px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  outline: none;
}

.new-note-input:focus {
  border-color: var(--accent-blue);
}

.notes-list {
  flex: 1;
  overflow-y: auto;
}

.note-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 10px 16px;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-bottom: 1px solid var(--border);
  border-left: 2px solid transparent;
  transition: background 0.15s;
}

.note-item:hover {
  background: var(--bg-tertiary);
}

.note-item.active {
  background: color-mix(in srgb, var(--accent-blue) 8%, var(--bg-tertiary));
  border-left: 2px solid var(--accent-blue);
}

.note-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-item-meta {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.notes-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 16px;
  text-align: center;
}

.notes-empty p {
  font-size: 13px;
  color: var(--text-secondary);
}

.btn-create-first {
  padding: 6px 14px;
  font-size: 13px;
  border: 1px solid var(--accent-blue);
  background: transparent;
  color: var(--accent-blue);
  border-radius: 6px;
  cursor: pointer;
}

.btn-create-first:hover {
  background: color-mix(in srgb, var(--accent-blue) 10%, transparent);
}

/* --- Right: Editor --- */
.notes-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-secondary);
  flex-shrink: 0;
}

.toolbar-group {
  display: flex;
  gap: 2px;
}

.toolbar-sep {
  width: 2px;
  height: 24px;
  background: var(--border);
  margin: 0 6px;
  opacity: 0.6;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  padding: 0 6px;
  border: none;
  background: none;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  border-radius: 4px;
}

.toolbar-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.toolbar-btn.active {
  background: color-mix(in srgb, var(--accent-blue) 15%, transparent);
  color: var(--accent-blue);
}

.toolbar-btn-danger:hover {
  color: var(--accent-red);
}

.toolbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.save-status {
  font-size: 12px;
  color: var(--text-secondary);
}

.save-status.saving { color: var(--accent-orange); }
.save-status.saved { color: var(--accent-green); }
.save-status.error { color: var(--accent-red); }

.delete-label {
  font-size: 12px;
  margin-left: 4px;
}

.editor-container {
  flex: 1;
  overflow-y: auto;
  display: flex;
  justify-content: center;
}

.editor-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
}

.editor-empty p {
  font-size: 13px;
}

/* --- Focus-visible styles --- */
.toolbar-btn:focus-visible,
.btn-new-note:focus-visible,
.note-item:focus-visible,
.btn-create-first:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 2px;
}

.new-note-input:focus-visible {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent-blue) 30%, transparent);
}
</style>

<!-- Unscoped: Tiptap editor content styles -->
<style>
.notes-editor-content {
  width: 100%;
  max-width: 800px;
  padding: 32px 40px;
  outline: none;
  color: var(--text-primary);
  font-size: 15px;
  line-height: 1.7;
}

.notes-editor-content > *:first-child {
  margin-top: 0 !important;
}

.notes-editor-content h1,
.notes-editor-content h2,
.notes-editor-content h3 {
  margin-top: 24px;
  margin-bottom: 12px;
  font-weight: 600;
  color: var(--text-primary);
}

.notes-editor-content h1 { font-size: 1.8em; }
.notes-editor-content h2 { font-size: 1.4em; }
.notes-editor-content h3 { font-size: 1.2em; }

.notes-editor-content p { margin-bottom: 12px; }

.notes-editor-content ul,
.notes-editor-content ol {
  padding-left: 1.5em;
  margin-bottom: 12px;
}

.notes-editor-content code {
  padding: 0.2em 0.4em;
  font-size: 85%;
  background: var(--bg-tertiary);
  border-radius: 4px;
  font-family: var(--font-mono);
}

.notes-editor-content pre {
  padding: 16px;
  font-size: 13px;
  background: var(--bg-tertiary);
  border-radius: 6px;
  border: 1px solid var(--border);
  margin-bottom: 12px;
  overflow-x: auto;
}

.notes-editor-content pre code {
  padding: 0;
  background: transparent;
  font-size: 100%;
}

.notes-editor-content blockquote {
  padding: 0 1em;
  color: var(--text-secondary);
  border-left: 3px solid var(--accent-blue);
  margin: 0 0 12px 0;
}

.notes-editor-content strong { font-weight: 600; }

/* Task list checkboxes */
.notes-editor-content ul[data-type="taskList"] {
  list-style: none;
  padding-left: 0;
}

.notes-editor-content ul[data-type="taskList"] li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 4px;
}

.notes-editor-content ul[data-type="taskList"] li > label {
  flex-shrink: 0;
  margin-top: 3px;
  display: flex;
  align-items: center;
}

.notes-editor-content ul[data-type="taskList"] li > label input[type="checkbox"] {
  appearance: none;
  -webkit-appearance: none;
  width: 16px !important;
  height: 16px !important;
  max-width: 16px;
  max-height: 16px;
  min-width: 16px;
  min-height: 16px;
  box-sizing: border-box;
  border: 1.5px solid var(--border);
  border-radius: 3px;
  background: var(--bg-primary);
  cursor: pointer;
  display: block;
  position: relative;
  margin: 0;
  padding: 0;
  flex-shrink: 0;
  flex-grow: 0;
}

.notes-editor-content ul[data-type="taskList"] li > label span {
  display: none;
}

.notes-editor-content ul[data-type="taskList"] li > label input[type="checkbox"]:checked {
  background: var(--accent-blue);
  border-color: var(--accent-blue);
}

.notes-editor-content ul[data-type="taskList"] li > label input[type="checkbox"]:checked::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 5px;
  width: 4px;
  height: 8px;
  border: solid var(--bg-primary);
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.notes-editor-content ul[data-type="taskList"] li[data-checked="true"] > div {
  text-decoration: line-through;
  color: var(--text-secondary);
}

/* Placeholder */
.notes-editor-content p.is-editor-empty:first-child::before {
  content: 'Start writing...';
  color: var(--text-secondary);
  opacity: 0.5;
  float: left;
  pointer-events: none;
  height: 0;
}
</style>
