<script setup lang="ts">
import { ref, computed } from 'vue'
import { useGitStore } from '../../stores/git'
import { useProject } from '../../composables/useProject'
import ShimmerBlock from '../ShimmerBlock.vue'
import type { StashEntry } from '../../types'

interface FileEntry {
  file: string
  status: 'staged' | 'modified' | 'untracked'
  statusChar: string
}

const props = defineProps<{
  selectedFile: string | null
  selectedStashIndex: number | null
}>()

const emit = defineEmits<{
  'select-file': [file: string]
  'toggle-check': [file: string]
  'stage-selected': []
  'select-all': []
  'unstage-all': []
  'select-stash': [index: number]
  'stash-apply': [index: number]
  'stash-pop': [index: number]
  'stash-drop': [index: number]
  'open-stash-dialog': []
}>()

const gitStore = useGitStore()
const { switching } = useProject()

const stagedCollapsed = ref(false)
const changesCollapsed = ref(false)
const stashCollapsed = ref(false)

const stagedChanges = computed<FileEntry[]>(() =>
  (gitStore.status.staged ?? []).map((f: string) => ({
    file: f,
    status: 'staged' as const,
    statusChar: 'S',
  })),
)

const unstagedChanges = computed<FileEntry[]>(() => {
  const modified = (gitStore.status.modified ?? []).map((f: string) => ({
    file: f,
    status: 'modified' as const,
    statusChar: 'M',
  }))
  const untracked = (gitStore.status.untracked ?? []).map((f: string) => ({
    file: f,
    status: 'untracked' as const,
    statusChar: 'N',
  }))
  return [...modified, ...untracked]
})

const totalChanges = computed(() =>
  (gitStore.status.modified?.length ?? 0)
  + (gitStore.status.untracked?.length ?? 0)
  + (gitStore.status.staged?.length ?? 0),
)

function splitPath(fullPath: string): { dir: string; name: string } {
  const lastSlash = fullPath.lastIndexOf('/')
  if (lastSlash === -1) return { dir: '', name: fullPath }
  return {
    dir: fullPath.substring(0, lastSlash + 1),
    name: fullPath.substring(lastSlash + 1),
  }
}

function getStatusColor(entry: FileEntry): string {
  switch (entry.status) {
    case 'staged':
      return 'var(--accent-blue)'
    case 'modified':
      return 'var(--accent-orange)'
    case 'untracked':
      return 'var(--accent-green)'
    default:
      return 'var(--text-secondary)'
  }
}

function getStatusBgColor(entry: FileEntry): string {
  switch (entry.status) {
    case 'staged':
      return 'rgba(88, 166, 255, 0.15)'
    case 'modified':
      return 'rgba(210, 153, 34, 0.15)'
    case 'untracked':
      return 'rgba(63, 185, 80, 0.15)'
    default:
      return 'rgba(139, 148, 158, 0.15)'
  }
}

function formatRelativeDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return dateStr
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)
  if (diffSec < 60) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  if (diffHour < 24) return `${diffHour}h ago`
  if (diffDay < 30) return `${diffDay}d ago`
  const months = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек']
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}, ${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
}
</script>

<template>
  <div class="files-panel">
    <div v-if="switching || gitStore.loading.status" class="shimmer-pad">
      <ShimmerBlock variant="row" :lines="3" />
    </div>
    <div v-else-if="totalChanges === 0" class="empty-state">
      <span class="empty-text">No changes</span>
    </div>
    <template v-else>
      <!-- Staged Changes -->
      <div v-if="stagedChanges.length > 0" class="file-group">
        <div class="file-group-header" @click="stagedCollapsed = !stagedCollapsed">
          <div class="file-group-header-left">
            <svg
              width="12" height="12" viewBox="0 0 12 12" fill="currentColor"
              class="collapse-chevron"
              :class="{ 'collapse-chevron-collapsed': stagedCollapsed }"
            >
              <path d="M6 8.825a.5.5 0 0 1-.354-.146l-3.5-3.5a.5.5 0 1 1 .708-.708L6 7.618l3.146-3.147a.5.5 0 1 1 .708.708l-3.5 3.5A.5.5 0 0 1 6 8.825z"/>
            </svg>
            Staged Changes
            <span class="file-count">{{ stagedChanges.length }}</span>
          </div>
          <button class="file-group-action" title="Unstage all" @click.stop="emit('unstage-all')">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
              <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06z"/>
            </svg>
          </button>
        </div>
        <template v-if="!stagedCollapsed">
          <div
            v-for="entry in stagedChanges"
            :key="'staged-' + entry.file"
            class="file-item"
            :class="{ 'file-active': selectedFile === entry.file }"
            @click="emit('select-file', entry.file)"
          >
            <input
              type="checkbox"
              class="file-checkbox"
              checked
              disabled
              title="Already staged"
            />
            <span
              class="file-badge"
              :style="{ color: getStatusColor(entry), backgroundColor: getStatusBgColor(entry) }"
            >
              {{ entry.statusChar }}
            </span>
            <span class="file-path" :title="entry.file">
              <span class="file-name">{{ splitPath(entry.file).name }}</span>
              <span class="file-dir">{{ splitPath(entry.file).dir }}</span>
            </span>
          </div>
        </template>
      </div>

      <!-- Unstaged Changes -->
      <div v-if="unstagedChanges.length > 0" class="file-group">
        <div class="file-group-header" @click="changesCollapsed = !changesCollapsed">
          <div class="file-group-header-left">
            <svg
              width="12" height="12" viewBox="0 0 12 12" fill="currentColor"
              class="collapse-chevron"
              :class="{ 'collapse-chevron-collapsed': changesCollapsed }"
            >
              <path d="M6 8.825a.5.5 0 0 1-.354-.146l-3.5-3.5a.5.5 0 1 1 .708-.708L6 7.618l3.146-3.147a.5.5 0 1 1 .708.708l-3.5 3.5A.5.5 0 0 1 6 8.825z"/>
            </svg>
            Changes
            <span class="file-count">{{ unstagedChanges.length }}</span>
          </div>
          <button class="file-group-action" title="Select all" @click.stop="emit('select-all')">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
              <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 0 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0z"/>
            </svg>
          </button>
        </div>
        <template v-if="!changesCollapsed">
          <div
            v-for="entry in unstagedChanges"
            :key="'unstaged-' + entry.file"
            class="file-item"
            :class="{ 'file-active': selectedFile === entry.file }"
            @click="emit('select-file', entry.file)"
          >
            <input
              type="checkbox"
              class="file-checkbox"
              :checked="gitStore.isSelected(entry.file)"
              @click.stop
              @change="emit('toggle-check', entry.file)"
            />
            <span
              class="file-badge"
              :style="{ color: getStatusColor(entry), backgroundColor: getStatusBgColor(entry) }"
            >
              {{ entry.statusChar }}
            </span>
            <span class="file-path" :title="entry.file">
              <span class="file-name">{{ splitPath(entry.file).name }}</span>
              <span class="file-dir">{{ splitPath(entry.file).dir }}</span>
            </span>
          </div>
        </template>
      </div>
    </template>

    <!-- Stash section -->
    <div v-if="gitStore.stashEntries.length > 0" class="stash-group">
      <div class="stash-header file-group-header" @click="stashCollapsed = !stashCollapsed">
        <div class="file-group-header-left">
          <svg
            width="12" height="12" viewBox="0 0 12 12" fill="currentColor"
            class="collapse-chevron"
            :class="{ 'collapse-chevron-collapsed': stashCollapsed }"
          >
            <path d="M6 8.825a.5.5 0 0 1-.354-.146l-3.5-3.5a.5.5 0 1 1 .708-.708L6 7.618l3.146-3.147a.5.5 0 1 1 .708.708l-3.5 3.5A.5.5 0 0 1 6 8.825z"/>
          </svg>
          Stash
          <span class="stash-badge">{{ gitStore.stashEntries.length }}</span>
        </div>
        <button class="stash-push-btn" title="Stash changes" @click.stop="emit('open-stash-dialog')">
          + Stash
        </button>
      </div>
      <template v-if="!stashCollapsed">
        <div
          v-for="entry in gitStore.stashEntries"
          :key="'stash-' + entry.index"
          class="stash-item"
          :class="{ 'stash-item-active': selectedStashIndex === entry.index }"
          @click="emit('select-stash', entry.index)"
        >
          <div class="stash-item-top">
            <span class="stash-index">stash@{{ '{' }}{{ entry.index }}{{ '}' }}</span>
            <span class="stash-message">{{ entry.message }}</span>
          </div>
          <div class="stash-item-bottom">
            <span class="stash-date">{{ formatRelativeDate(entry.date) }}</span>
            <div class="stash-actions">
              <button class="stash-action-btn apply" title="Apply" @click.stop="emit('stash-apply', entry.index)">Apply</button>
              <button class="stash-action-btn pop" title="Pop" @click.stop="emit('stash-pop', entry.index)">Pop</button>
              <button class="stash-action-btn drop" title="Drop" @click.stop="emit('stash-drop', entry.index)">Drop</button>
            </div>
          </div>
        </div>
      </template>
    </div>
    <!-- Stash push button when no stash entries exist but there are changes -->
    <div v-else class="stash-group stash-group-empty">
      <div class="stash-header file-group-header">
        <div class="file-group-header-left">
          Stash
          <span class="stash-badge">0</span>
        </div>
        <button class="stash-push-btn" title="Stash changes" @click.stop="emit('open-stash-dialog')">
          + Stash
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.files-panel {
  background: var(--bg-secondary);
  overflow-y: auto;
  height: 100%;
}

.shimmer-pad {
  padding: 12px;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.empty-text {
  color: var(--text-secondary);
  font-size: 13px;
}

/* File groups */
.file-group {
  border-bottom: 1px solid var(--border);
}

.file-group:last-child {
  border-bottom: none;
}

.file-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: var(--bg-primary);
  position: sticky;
  top: 0;
  z-index: 1;
  cursor: pointer;
  user-select: none;
}

.file-group-header:hover {
  background: var(--bg-secondary);
}

.file-group-header-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.collapse-chevron {
  flex-shrink: 0;
  transition: transform 0.15s;
}

.collapse-chevron-collapsed {
  transform: rotate(-90deg);
}

.file-group-action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.file-group-action:hover {
  background: var(--border);
  color: var(--text-primary);
}

.file-count {
  background: var(--border);
  color: var(--text-primary);
  padding: 0 6px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  cursor: pointer;
  transition: background 0.1s;
}

.file-item:hover {
  background: var(--bg-tertiary);
}

.file-active {
  background: var(--bg-tertiary);
  border-left: 2px solid var(--accent-blue);
  padding-left: 10px;
}

.file-checkbox {
  width: 14px;
  height: 14px;
  accent-color: var(--accent-blue);
  cursor: pointer;
  flex-shrink: 0;
}

.file-badge {
  font-size: 10px;
  font-weight: 700;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 3px;
  flex-shrink: 0;
  font-family: var(--font-mono);
}

.file-path {
  font-family: var(--font-mono);
  font-size: 12px;
  display: flex;
  align-items: baseline;
  gap: 6px;
  overflow: hidden;
  min-width: 0;
  flex: 1;
}

.file-name {
  color: var(--text-primary);
  font-weight: 500;
  white-space: nowrap;
  flex-shrink: 0;
}

.file-dir {
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

/* Stash Section */
.stash-group {
  border-top: 1px solid var(--border);
  margin-top: 8px;
  padding-top: 4px;
}

.stash-group-empty {
  opacity: 0.7;
}

.stash-header {
  color: var(--accent-purple);
}

.stash-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 700;
  background: rgba(188, 140, 255, 0.15);
  color: var(--accent-purple);
}

.stash-push-btn {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  background: transparent;
  border: 1px solid rgba(188, 140, 255, 0.4);
  border-radius: 4px;
  color: var(--accent-purple);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.stash-push-btn:hover {
  background: rgba(188, 140, 255, 0.1);
  border-color: var(--accent-purple);
}

.stash-item {
  padding: 6px 12px;
  cursor: pointer;
  transition: background 0.1s;
  border-left: 2px solid transparent;
}

.stash-item:hover {
  background: var(--bg-tertiary);
}

.stash-item-active {
  background: rgba(188, 140, 255, 0.06);
  border-left-color: var(--accent-purple);
}

.stash-item-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.stash-index {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-purple);
  font-weight: 500;
  flex-shrink: 0;
}

.stash-message {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.stash-item-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.stash-date {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.stash-actions {
  display: flex;
  gap: 4px;
}

.stash-action-btn {
  padding: 1px 8px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.stash-action-btn.apply {
  color: var(--accent-green);
  border-color: rgba(63, 185, 80, 0.3);
}

.stash-action-btn.apply:hover {
  background: rgba(63, 185, 80, 0.1);
  border-color: var(--accent-green);
}

.stash-action-btn.pop {
  color: var(--accent-blue);
  border-color: rgba(88, 166, 255, 0.3);
}

.stash-action-btn.pop:hover {
  background: rgba(88, 166, 255, 0.1);
  border-color: var(--accent-blue);
}

.stash-action-btn.drop {
  color: var(--accent-red);
  border-color: rgba(248, 81, 73, 0.3);
}

.stash-action-btn.drop:hover {
  background: rgba(248, 81, 73, 0.1);
  border-color: var(--accent-red);
}
</style>
