<script setup lang="ts">
import { defineAsyncComponent, computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { Splitpanes, Pane } from 'splitpanes'
import 'splitpanes/dist/splitpanes.css'
import FileTree from '../components/FileTree.vue'
import CodeEditor from '../components/CodeEditor.vue'
import ImagePreview from '../components/ImagePreview.vue'
import { getFileIcon } from '../components/FileIcons'
import { useFilesStore } from '../stores/files'
import { useSettingsStore } from '../stores/settings'
import { useProjectsStore } from '../stores/projects'
import { filesApi } from '../api/files'
import { detectLanguageFromFilename } from '../utils/diff'

const filesStore = useFilesStore()
const settingsStore = useSettingsStore()
const projectsStore = useProjectsStore()

const MonacoEditor = defineAsyncComponent(() => import('../components/MonacoEditor.vue'))
const MonacoDiffViewer = defineAsyncComponent(() => import('../components/MonacoDiffViewer.vue'))

const EditorComponent = computed(() =>
  settingsStore.ui.editorEngine === 'monaco' ? MonacoEditor : CodeEditor
)

const isImageFile = computed(() => filesStore.activeFile?.language === 'image')

const imageUrl = computed(() => {
  if (!isImageFile.value || !filesStore.activeFilePath) return ''
  const projectName = projectsStore.currentProject?.name || '_'
  return `/api/projects/${projectName}/files/content/${encodeURIComponent(filesStore.activeFilePath)}?raw=true`
})

const diffMode = ref(false)
const diskContent = ref('')

async function openDiff() {
  if (!filesStore.activeFilePath) return
  const projectName = projectsStore.currentProject?.name || '_'
  try {
    diskContent.value = await filesApi.content(projectName, filesStore.activeFilePath)
    diffMode.value = true
  } catch { /* ignore */ }
}

function closeDiff() {
  diffMode.value = false
  diskContent.value = ''
}

const diffLanguage = computed(() => {
  if (!filesStore.activeFile) return 'plaintext'
  return detectLanguageFromFilename(filesStore.activeFile.name)
})

function handleSave() {
  if (filesStore.activeFile?.dirty) {
    filesStore.saveFile(filesStore.activeFile.path)
  }
}

// Listen for Ctrl+S event from keyboard shortcuts composable
function onEditorSave() { handleSave() }

onMounted(() => {
  filesStore.fetchTree()
  window.addEventListener('editor:save', onEditorSave)
})

onBeforeUnmount(() => {
  window.removeEventListener('editor:save', onEditorSave)
})

function handleContentUpdate(value: string) {
  if (filesStore.activeFilePath) {
    filesStore.updateContent(filesStore.activeFilePath, value)
  }
}
</script>

<template>
  <div class="editor-view">
    <Splitpanes class="default-theme">
      <!-- File tree pane -->
      <Pane :size="20" :min-size="15" :max-size="40">
        <FileTree />
      </Pane>

      <!-- Editor pane -->
      <Pane :size="80">
        <div class="editor-pane">
          <!-- Tab bar -->
          <div class="editor-tabs">
            <div
              v-for="file in filesStore.openFiles"
              :key="file.path"
              class="editor-tab"
              :class="{ active: filesStore.activeFilePath === file.path }"
              @click="filesStore.activeFilePath = file.path"
              @mousedown.middle="filesStore.closeFile(file.path)"
            >
              <!-- File icon (small, inline) -->
              <span class="tab-icon" v-html="getFileIcon(file.name, false, false)"></span>
              <span class="tab-name">{{ file.name }}</span>
              <span v-if="file.dirty" class="tab-unsaved"></span>
              <button
                class="tab-close"
                @click.stop="filesStore.closeFile(file.path)"
                title="Close"
              >×</button>
            </div>
          </div>

          <!-- Disk change banner -->
          <div
            v-if="filesStore.activeFile && filesStore.changedOnDisk.has(filesStore.activeFilePath!)"
            class="disk-change-banner"
          >
            <span>⚠ <strong>{{ filesStore.activeFile.name }}</strong> was modified on disk.</span>
            <div class="banner-actions">
              <button class="banner-btn primary" @click="filesStore.reloadFromDisk(filesStore.activeFilePath!)">Reload</button>
              <button class="banner-btn secondary" @click="filesStore.dismissDiskChange(filesStore.activeFilePath!)">Keep mine</button>
              <button v-if="settingsStore.ui.editorEngine === 'monaco'" class="banner-btn secondary" @click="openDiff">Diff</button>
            </div>
          </div>

          <!-- Diff mode -->
          <div v-if="diffMode && filesStore.activeFile" class="editor-content diff-mode-content">
            <div class="diff-mode-bar">
              <span class="diff-mode-label">Comparing: editor vs disk</span>
              <button class="diff-mode-close" @click="closeDiff">&times; Close diff</button>
            </div>
            <MonacoDiffViewer
              :original="filesStore.activeFile.content"
              :modified="diskContent"
              :language="diffLanguage"
              :filename="filesStore.activeFile.name"
            />
          </div>

          <!-- Image preview -->
          <div v-else-if="isImageFile && filesStore.activeFile" class="editor-content">
            <ImagePreview :src="imageUrl" :filename="filesStore.activeFile.name" />
          </div>

          <!-- Normal code editor -->
          <div v-else-if="filesStore.activeFile" class="editor-content">
            <component
              :is="EditorComponent"
              :model-value="filesStore.activeFile.content"
              :language="filesStore.activeFile.language"
              @update:model-value="handleContentUpdate"
            />
          </div>

          <!-- Empty state -->
          <div v-else class="editor-empty">
            <p>Open a file from the tree</p>
          </div>

          <!-- Status bar -->
          <div class="editor-statusbar">
            <div class="statusbar-left">
              <span v-if="filesStore.activeFile && !isImageFile">{{ filesStore.activeFile.language }}</span>
              <span v-if="isImageFile">image</span>
              <span>UTF-8</span>
            </div>
            <div class="statusbar-right">
              <span v-if="filesStore.activeFile?.dirty" class="unsaved-indicator">● Unsaved</span>
              <span v-if="filesStore.activeFile && !isImageFile">Ctrl+S</span>
            </div>
          </div>
        </div>
      </Pane>
    </Splitpanes>
  </div>
</template>

<style scoped>
.editor-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  margin: -16px -32px;
  background: var(--bg-primary);
}

.editor-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary);
}

/* Tab bar */
.editor-tabs {
  display: flex;
  align-items: center;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  height: 36px;
  flex-shrink: 0;
  overflow-x: auto;
}

.editor-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 100%;
  font-size: 13px;
  color: var(--text-secondary);
  border-right: 1px solid var(--border);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
}

.editor-tab:hover { background: var(--bg-tertiary); }

.editor-tab.active {
  background: var(--bg-primary);
  color: var(--text-primary);
  border-bottom: 2px solid var(--accent-blue);
  margin-bottom: -1px;
}

.tab-icon { width: 16px; height: 16px; flex-shrink: 0; display: flex; align-items: center; }
.tab-icon :deep(svg) { width: 16px; height: 16px; }

.tab-unsaved {
  width: 8px; height: 8px; border-radius: 50%;
  background: var(--accent-orange); flex-shrink: 0;
}

.tab-close {
  width: 18px; height: 18px;
  display: flex; align-items: center; justify-content: center;
  border: none; background: none;
  color: var(--text-secondary); cursor: pointer;
  border-radius: 3px; font-size: 14px;
  opacity: 0; flex-shrink: 0;
}

.editor-tab:hover .tab-close,
.editor-tab.active .tab-close { opacity: 0.5; }
.tab-close:hover { opacity: 1 !important; background: rgba(248,81,73,0.15); color: var(--accent-red); }

/* Disk change banner */
.disk-change-banner {
  padding: 8px 16px;
  background: rgba(210,153,34,0.1);
  border-bottom: 1px solid rgba(210,153,34,0.25);
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: var(--accent-orange);
  flex-shrink: 0;
}
.banner-actions { display: flex; gap: 8px; }
.banner-btn {
  padding: 4px 12px; border-radius: 6px; border: 1px solid;
  font-size: 12px; cursor: pointer; font-weight: 500;
}
.banner-btn.primary { background: var(--accent-blue); color: #fff; border-color: var(--accent-blue); }
.banner-btn.secondary { background: none; color: var(--text-secondary); border-color: var(--border); }

/* Editor content */
.editor-content { flex: 1; min-height: 0; overflow: hidden; }

/* Diff mode */
.diff-mode-content {
  display: flex;
  flex-direction: column;
}

.diff-mode-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 16px;
  background: rgba(88, 166, 255, 0.08);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.diff-mode-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--accent-blue);
}

.diff-mode-close {
  padding: 2px 10px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.diff-mode-close:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

/* Empty state */
.editor-empty {
  flex: 1;
  display: flex; align-items: center; justify-content: center;
  color: var(--text-secondary); font-size: 14px;
}

/* Status bar */
.editor-statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  height: 24px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  font-size: 11px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  flex-shrink: 0;
}
.statusbar-left, .statusbar-right { display: flex; gap: 16px; align-items: center; }
.unsaved-indicator { color: var(--accent-orange); }

/* Splitpanes overrides */
:deep(.default-theme .splitpanes__splitter) {
  background: var(--border); min-width: 4px;
}
:deep(.default-theme .splitpanes__splitter:hover) {
  background: var(--accent-blue);
}
:deep(.default-theme .splitpanes__splitter::before),
:deep(.default-theme .splitpanes__splitter::after) {
  display: none;
}
</style>
