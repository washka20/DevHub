<script setup lang="ts">
import { ref, computed } from 'vue'
import { useGitStore } from '../../stores/git'
import type { CommitDetail } from '../../types'

const props = defineProps<{
  commit: CommitDetail
}>()

const emit = defineEmits<{
  close: []
  'view-file-diff': [hash: string, filePath: string]
}>()

const gitStore = useGitStore()

const hashCopied = ref(false)
const showCherryPickConfirm = ref(false)
const cherryPicking = ref(false)

const isHeadCommit = computed(() => {
  const first = gitStore.graphNodes[0]
  return first && first.id === props.commit.hash
})

async function doCherryPick() {
  cherryPicking.value = true
  try {
    await gitStore.cherryPick(props.commit.hash)
    showCherryPickConfirm.value = false
  } finally {
    cherryPicking.value = false
  }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
  hashCopied.value = true
  setTimeout(() => { hashCopied.value = false }, 2000)
}

interface ParsedStatLine {
  file: string
  additions: number
  deletions: number
}

function parseStats(stats: string): { lines: ParsedStatLine[], summary: string } {
  if (!stats) return { lines: [], summary: '' }
  const rawLines = stats.split('\n').filter(l => l.trim())
  const parsed: ParsedStatLine[] = []
  let summary = ''

  for (const line of rawLines) {
    if (line.includes('files changed') || line.includes('file changed')) {
      summary = line.trim()
      continue
    }
    const match = line.match(/^\s*(.+?)\s*\|\s*(\d+)\s*([+-]*)/)
    if (match) {
      const file = match[1].trim()
      const total = parseInt(match[2]) || 0
      const symbols = match[3] || ''
      const adds = (symbols.match(/\+/g) || []).length
      const dels = (symbols.match(/-/g) || []).length
      const ratio = adds + dels > 0 ? adds / (adds + dels) : 0.5
      parsed.push({ file, additions: Math.round(total * ratio), deletions: Math.round(total * (1 - ratio)) })
    }
  }
  return { lines: parsed, summary }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return dateStr
  const months = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек']
  return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}, ${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}`
}

function getFileStatusColor(statusChar: string): string {
  switch (statusChar) {
    case 'A': return '#3fb950'
    case 'M': return '#d29922'
    case 'D': return '#f85149'
    default: return '#8b949e'
  }
}

function getFileStatusBg(statusChar: string): string {
  switch (statusChar) {
    case 'A': return 'rgba(63, 185, 80, 0.15)'
    case 'M': return 'rgba(210, 153, 34, 0.15)'
    case 'D': return 'rgba(248, 81, 73, 0.15)'
    default: return 'rgba(139, 148, 158, 0.15)'
  }
}
</script>

<template>
  <div class="commit-detail-panel">
    <div class="commit-detail-header">
      <span class="commit-detail-title">Commit Details</span>
      <button class="commit-detail-close" @click="emit('close')">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
          <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06z"/>
        </svg>
      </button>
    </div>
    <div class="commit-detail-body">
      <!-- Message as heading -->
      <div class="detail-message">{{ commit.message }}</div>

      <!-- Body -->
      <div v-if="commit.body" class="detail-body-text">{{ commit.body }}</div>

      <!-- Meta row: hash + date + author -->
      <div class="detail-meta">
        <div class="detail-meta-row">
          <span
            class="detail-hash"
            :title="hashCopied ? 'Copied!' : 'Click to copy full hash'"
            @click="copyToClipboard(commit.hash)"
          >
            {{ hashCopied ? 'Copied!' : commit.hash.slice(0, 7) }}
          </span>
          <span class="detail-date">{{ formatDate(commit.date) }}</span>
        </div>
        <div class="detail-author-row">
          <span class="detail-author-name">{{ commit.author }}</span>
          <span class="detail-author-email">{{ commit.email }}</span>
        </div>
        <button
          v-if="!isHeadCommit"
          class="cherry-pick-btn"
          :disabled="cherryPicking"
          @click="showCherryPickConfirm = true"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 1a2.5 2.5 0 0 0-1 4.8V7H5.5a.5.5 0 0 0 0 1H7v2.2a2.5 2.5 0 1 0 2 0V8h1.5a.5.5 0 0 0 0-1H9V5.8A2.5 2.5 0 0 0 8 1zm0 1.5a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm0 8a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
          </svg>
          Cherry-pick
        </button>
      </div>

      <!-- Cherry-pick confirmation -->
      <div v-if="showCherryPickConfirm" class="cherry-pick-confirm">
        <div class="cherry-pick-confirm-text">
          Cherry-pick commit <code>{{ commit.hash.slice(0, 7) }}</code>?
        </div>
        <div class="cherry-pick-confirm-actions">
          <button class="cherry-pick-cancel" @click="showCherryPickConfirm = false">Cancel</button>
          <button class="cherry-pick-ok" :disabled="cherryPicking" @click="doCherryPick">
            {{ cherryPicking ? 'Picking...' : 'Confirm' }}
          </button>
        </div>
      </div>

      <!-- Stats - parsed into visual bars -->
      <div v-if="commit.stats" class="detail-stats">
        <div class="detail-files-header">
          Changes
          <span v-if="parseStats(commit.stats).summary" class="detail-stats-summary">
            {{ parseStats(commit.stats).summary }}
          </span>
        </div>
        <div
          v-for="stat in parseStats(commit.stats).lines"
          :key="stat.file"
          class="detail-stat-line"
          @click="emit('view-file-diff', commit.hash, stat.file)"
        >
          <span class="detail-stat-file">{{ stat.file.split('/').pop() }}</span>
          <span class="detail-stat-bar">
            <span class="stat-add" :style="{ width: Math.min(stat.additions, 100) + 'px' }"></span>
            <span class="stat-del" :style="{ width: Math.min(stat.deletions, 100) + 'px' }"></span>
          </span>
          <span class="detail-stat-nums">
            <span v-if="stat.additions" class="stat-num-add">+{{ stat.additions }}</span>
            <span v-if="stat.deletions" class="stat-num-del">-{{ stat.deletions }}</span>
          </span>
        </div>
      </div>

      <!-- Changed files -->
      <div class="detail-files-header">Files</div>
      <div
        v-for="f in commit.files"
        :key="f.path"
        class="detail-file-item"
        @click="emit('view-file-diff', commit.hash, f.path)"
      >
        <span
          class="detail-file-status"
          :style="{ color: getFileStatusColor(f.status), backgroundColor: getFileStatusBg(f.status) }"
        >
          {{ f.status }}
        </span>
        <span class="detail-file-path">{{ f.path }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.commit-detail-panel {
  width: 350px;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border-left: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.commit-detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.commit-detail-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.commit-detail-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
}

.commit-detail-close:hover {
  background: var(--border);
  color: var(--text-primary);
}

.commit-detail-body {
  flex: 1;
  overflow-y: auto;
  padding: 14px;
}

.detail-message {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.4;
  margin-bottom: 8px;
}

.detail-body-text {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  line-height: 1.5;
  margin-bottom: 12px;
  padding: 8px 10px;
  background: rgba(0,0,0,0.2);
  border-radius: 6px;
  border-left: 2px solid var(--border);
}

.detail-meta {
  padding: 10px 0;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--border);
}

.detail-meta-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
}

.detail-hash {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-blue);
  background: rgba(88,166,255,0.1);
  padding: 2px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.detail-hash:hover {
  background: rgba(88,166,255,0.2);
}

.detail-date {
  font-size: 12px;
  color: var(--text-secondary);
}

.detail-author-row {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.detail-author-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.detail-author-email {
  font-size: 11px;
  color: var(--text-secondary);
}

.detail-stats {
  margin-bottom: 4px;
}

.detail-stats-summary {
  font-weight: 400;
  color: var(--text-secondary);
  margin-left: 8px;
  font-size: 10px;
  text-transform: none;
  letter-spacing: 0;
}

.detail-stat-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 6px;
  margin: 0 -6px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.1s;
}

.detail-stat-line:hover {
  background: var(--bg-tertiary);
}

.detail-stat-file {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-primary);
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-stat-bar {
  display: flex;
  gap: 1px;
  flex-shrink: 0;
}

.stat-add {
  height: 8px;
  background: var(--accent-green);
  border-radius: 2px 0 0 2px;
  min-width: 0;
}

.stat-del {
  height: 8px;
  background: var(--accent-red);
  border-radius: 0 2px 2px 0;
  min-width: 0;
}

.detail-stat-nums {
  font-family: var(--font-mono);
  font-size: 11px;
  flex-shrink: 0;
  min-width: 60px;
  text-align: right;
}

.stat-num-add { color: var(--accent-green); }
.stat-num-del { color: var(--accent-red); margin-left: 4px; }

.detail-files-header {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 8px 0;
  margin-top: 8px;
  border-top: 1px solid var(--border);
}

.detail-file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 6px;
  margin: 0 -6px;
  cursor: pointer;
  transition: background 0.1s;
  border-radius: 4px;
}

.detail-file-item:hover {
  background: var(--bg-tertiary);
}

.detail-file-status {
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

.detail-file-path {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cherry-pick-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.cherry-pick-btn:hover {
  color: var(--text-primary);
  background: var(--bg-tertiary);
  border-color: var(--text-secondary);
}

.cherry-pick-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.cherry-pick-confirm {
  padding: 10px 12px;
  margin-top: 4px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 8px;
}

.cherry-pick-confirm-text {
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 10px;
}

.cherry-pick-confirm-text code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-blue);
  background: rgba(88,166,255,0.1);
  padding: 1px 6px;
  border-radius: 3px;
}

.cherry-pick-confirm-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.cherry-pick-cancel,
.cherry-pick-ok {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 6px;
  border: 1px solid var(--border);
  cursor: pointer;
  transition: all 0.15s;
}

.cherry-pick-cancel {
  color: var(--text-secondary);
  background: transparent;
}

.cherry-pick-cancel:hover {
  color: var(--text-primary);
  background: var(--bg-secondary);
}

.cherry-pick-ok {
  color: #fff;
  background: var(--accent-blue);
  border-color: var(--accent-blue);
}

.cherry-pick-ok:hover {
  filter: brightness(1.1);
}

.cherry-pick-ok:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
