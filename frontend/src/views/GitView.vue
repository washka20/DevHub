<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useGitStore } from '../stores/git'
import { useProjectsStore } from '../stores/projects'
import { useProject } from '../composables/useProject'
import { gitApi } from '../api/git'
import ShimmerBlock from '../components/ShimmerBlock.vue'
import type { DiffLine } from '../types'

const gitStore = useGitStore()
const projectsStore = useProjectsStore()
const { switching } = useProject()
const selectedFile = ref<string | null>(null)
const branchDropdownOpen = ref(false)
const stagedCollapsed = ref(false)
const changesCollapsed = ref(false)
const stashCollapsed = ref(false)
const showStashDialog = ref(false)
const stashMessage = ref('')
const stashDiffContent = ref('')
const selectedStashIndex = ref<number | null>(null)
const commitDiffContent = ref('')
const commitDiffFile = ref<string | null>(null)

// Resizable files panel
const filesPanelWidth = ref(320)
const isResizing = ref(false)

function startResize(e: MouseEvent) {
  isResizing.value = true
  const startX = e.clientX
  const startWidth = filesPanelWidth.value

  function onMouseMove(ev: MouseEvent) {
    const newWidth = startWidth + (ev.clientX - startX)
    filesPanelWidth.value = Math.max(200, Math.min(600, newWidth))
  }

  function onMouseUp() {
    isResizing.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Split file path into dir + filename
function splitPath(fullPath: string): { dir: string; name: string } {
  const lastSlash = fullPath.lastIndexOf('/')
  if (lastSlash === -1) return { dir: '', name: fullPath }
  return {
    dir: fullPath.substring(0, lastSlash + 1),
    name: fullPath.substring(lastSlash + 1),
  }
}
const showCommitDiffModal = ref(false)

// ----- Computed: all changed files grouped -----

interface FileEntry {
  file: string
  status: 'staged' | 'modified' | 'untracked'
  statusChar: string
}

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

// ----- Status summary -----

const statusSummary = computed(() => {
  const parts: string[] = []
  const modCount =
    (gitStore.status.modified?.length ?? 0) +
    (gitStore.status.untracked?.length ?? 0)
  const stagedCount = gitStore.status.staged?.length ?? 0
  const ahead = gitStore.status.ahead ?? 0
  const behind = gitStore.status.behind ?? 0

  if (modCount > 0) parts.push(`${modCount} modified`)
  if (stagedCount > 0) parts.push(`${stagedCount} staged`)
  if (ahead > 0) parts.push(`${ahead} ahead`)
  if (behind > 0) parts.push(`${behind} behind`)

  return parts.length > 0 ? parts.join(', ') : 'Clean'
})

// ----- Diff parsing -----

const parsedDiff = computed<DiffLine[]>(() => {
  if (!gitStore.diff) return []
  return parseDiff(gitStore.diff)
})

const parsedCommitDiff = computed<DiffLine[]>(() => {
  if (!commitDiffContent.value) return []
  return parseDiff(commitDiffContent.value)
})

function parseDiff(raw: string): DiffLine[] {
  const lines = raw.split('\n')
  const result: DiffLine[] = []
  let oldLine = 0
  let newLine = 0

  for (const line of lines) {
    if (line.startsWith('@@')) {
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
      if (match) {
        oldLine = parseInt(match[1], 10)
        newLine = parseInt(match[2], 10)
      }
      result.push({ type: 'header', content: line, oldLineNo: null, newLineNo: null })
    } else if (line.startsWith('diff ') || line.startsWith('index ') || line.startsWith('---') || line.startsWith('+++')) {
      result.push({ type: 'header', content: line, oldLineNo: null, newLineNo: null })
    } else if (line.startsWith('+')) {
      result.push({ type: 'add', content: line.substring(1), oldLineNo: null, newLineNo: newLine })
      newLine++
    } else if (line.startsWith('-')) {
      result.push({ type: 'remove', content: line.substring(1), oldLineNo: oldLine, newLineNo: null })
      oldLine++
    } else {
      const content = line.startsWith(' ') ? line.substring(1) : line
      if (line === '') continue
      result.push({ type: 'context', content, oldLineNo: oldLine, newLineNo: newLine })
      oldLine++
      newLine++
    }
  }

  return result
}

// ----- Tab counts -----

const tabCounts = computed(() => ({
  changes: totalChanges.value,
  log: gitStore.totalCommits,
  branches: gitStore.branches.length,
}))

// ----- Virtual scroll -----

const ROW_HEIGHT = 28
const OVERSCAN = 10

const scrollTop = ref(0)
const containerHeight = ref(600)

const totalHeight = computed(() => gitStore.totalCommits * ROW_HEIGHT)

const visibleRange = computed(() => {
  const startIdx = Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - OVERSCAN)
  const endIdx = Math.min(
    gitStore.totalCommits,
    Math.ceil((scrollTop.value + containerHeight.value) / ROW_HEIGHT) + OVERSCAN
  )
  return { startIdx, endIdx }
})

const offsetY = computed(() => visibleRange.value.startIdx * ROW_HEIGHT)

const visibleNodes = computed(() =>
  gitStore.graphNodes.slice(visibleRange.value.startIdx, visibleRange.value.endIdx)
)

// ----- Ref badge helpers -----

function getRefType(refStr: string): 'branch' | 'tag' | 'hotfix' | 'head' {
  if (refStr === 'HEAD') return 'head'
  if (refStr.startsWith('tag:')) return 'tag'
  if (refStr.includes('hotfix')) return 'hotfix'
  return 'branch'
}

function getRefLabel(refStr: string): string {
  return refStr.replace('tag: ', '').replace('HEAD -> ', '')
}

function getRefClass(refStr: string): string {
  const t = getRefType(refStr)
  switch (t) {
    case 'tag': return 'ref-tag'
    case 'hotfix': return 'ref-hotfix'
    case 'head': return 'ref-head'
    default: return 'ref-branch'
  }
}

// ----- Actions -----

function selectFile(file: string) {
  selectedFile.value = file
  selectedStashIndex.value = null
  stashDiffContent.value = ''
  gitStore.fetchDiff(file)
}

function toggleFileCheck(file: string) {
  gitStore.toggleSelectFile(file)
}

function stageSelected() {
  gitStore.stageSelected()
}

function selectAllForStage() {
  gitStore.selectAllUnstaged()
}

function unstageAll() {
  gitStore.unstageAll()
}

function canCommit(): boolean {
  const hasMessage = gitStore.commitMessage.trim().length > 0
  const hasFiles = (gitStore.status.staged?.length ?? 0) > 0
  return hasMessage && hasFiles
}

async function doCommit() {
  if (!canCommit()) return
  const files = [
    ...gitStore.stagedFiles,
    ...(gitStore.status.staged ?? []),
  ]
  const unique = [...new Set(files)]
  await gitStore.commit(gitStore.commitMessage, unique)
  selectedFile.value = null
  gitStore.diff = ''
}

async function selectBranch(branch: string) {
  branchDropdownOpen.value = false
  if (branch !== gitStore.status.branch) {
    await gitStore.checkout(branch)
  }
}

async function selectCommit(hash: string) {
  await gitStore.fetchCommitDetail(hash)
}

async function viewCommitFileDiff(hash: string, filePath: string) {
  commitDiffFile.value = filePath
  showCommitDiffModal.value = true
  try {
    const projectName = projectsStore.currentProject?.name || 'default'
    const base = `/api/projects/${encodeURIComponent(projectName)}`
    const data = await gitApi.commitDiff(base, hash, filePath)
    commitDiffContent.value = data.diff ?? ''
  } catch {
    commitDiffContent.value = ''
  }
}

function closeCommitDiffModal() {
  showCommitDiffModal.value = false
  commitDiffContent.value = ''
  commitDiffFile.value = null
}

async function doGenerateCommit() {
  await gitStore.generateCommitMessage()
}

function onLogScroll(e: Event) {
  const el = e.target as HTMLElement
  scrollTop.value = el.scrollTop
  containerHeight.value = el.clientHeight

  // Предзагрузка метаданных
  const { endIdx } = visibleRange.value
  if (endIdx > gitStore.metadataLoaded - 20 && !gitStore.metadataLoading) {
    gitStore.fetchMetadata(gitStore.metadataLoaded, 50)
  }
}

const hashCopied = ref(false)

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
    // Summary line: "14 files changed, 1723 insertions(+), 19 deletions(-)"
    if (line.includes('files changed') || line.includes('file changed')) {
      summary = line.trim()
      continue
    }
    // File line: " path/to/file | 380 ++++++++-----"
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

// Close dropdown when clicking outside
function onClickOutside() {
  branchDropdownOpen.value = false
}

// ----- Branch filter sidebar (Log tab) -----

const branchSearch = ref('')
const filteredBranches = computed(() => {
  const q = branchSearch.value.toLowerCase()
  return gitStore.branches.filter(b => !q || b.name.toLowerCase().includes(q))
})

// ----- Branch cards (Branches tab) -----

const expandedBranch = ref<string | null>(null)
const branchTabSearch = ref('')
const branchTabFilter = ref<'local' | 'remote' | 'all'>('local')
const showCheckoutConfirm = ref<string | null>(null)

const filteredBranchCards = computed(() => {
  const q = branchTabSearch.value.toLowerCase()
  return gitStore.branches.filter(b => {
    if (q && !b.name.toLowerCase().includes(q)) return false
    if (branchTabFilter.value === 'local') return !b.name.startsWith('remotes/')
    if (branchTabFilter.value === 'remote') return b.name.startsWith('remotes/')
    return true
  })
})

function toggleBranchExpand(name: string) {
  if (expandedBranch.value === name) {
    expandedBranch.value = null
  } else {
    expandedBranch.value = name
    gitStore.fetchBranchCommits(name)
  }
}

function viewBranchLog(name: string) {
  gitStore.setViewingBranch(name)
  gitStore.activeTab = 'log'
}

async function confirmCheckout(name: string) {
  showCheckoutConfirm.value = null
  await gitStore.checkout(name)
}

// ----- Stash actions -----

async function openStashDialog() {
  stashMessage.value = ''
  showStashDialog.value = true
}

async function doStashPush() {
  await gitStore.stashPush(stashMessage.value)
  showStashDialog.value = false
  stashMessage.value = ''
}

async function doStashApply(index: number) {
  await gitStore.stashApply(index)
}

async function doStashPop(index: number) {
  await gitStore.stashPop(index)
  selectedStashIndex.value = null
  stashDiffContent.value = ''
}

async function doStashDrop(index: number) {
  await gitStore.stashDrop(index)
  if (selectedStashIndex.value === index) {
    selectedStashIndex.value = null
    stashDiffContent.value = ''
  }
}

async function selectStash(index: number) {
  selectedStashIndex.value = index
  selectedFile.value = null
  gitStore.diff = ''
  stashDiffContent.value = await gitStore.stashDiff(index)
}

const parsedStashDiff = computed<DiffLine[]>(() => {
  if (!stashDiffContent.value) return []
  return parseDiff(stashDiffContent.value)
})

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
  return formatDate(dateStr)
}

// ----- Lifecycle -----

onMounted(() => {
  gitStore.fetchStatus()
  gitStore.fetchBranches()
  gitStore.fetchGraph()
})

// Refetch when the branch changes
watch(() => gitStore.status.branch, () => {
  selectedFile.value = null
  gitStore.diff = ''
})
</script>

<template>
  <div class="git-view" @click="onClickOutside">
    <!-- TopBar -->
    <header class="top-bar">
      <div class="top-bar-left">
        <span class="top-bar-title">Source Control</span>
        <div class="branch-selector" @click.stop>
          <button
            class="branch-button"
            @click="branchDropdownOpen = !branchDropdownOpen"
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" class="branch-icon">
              <path d="M11.75 2.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zm-2.25.75a2.25 2.25 0 1 1 3 2.122V6A2.5 2.5 0 0 1 10 8.5H6a1 1 0 0 0-1 1v1.128a2.251 2.251 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.5 0v1.836A2.492 2.492 0 0 1 6 7h4a1 1 0 0 0 1-1v-.628A2.25 2.25 0 0 1 9.5 3.25zM4.25 12a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zM3.5 3.25a.75.75 0 1 1 1.5 0 .75.75 0 0 1-1.5 0z"/>
            </svg>
            <span class="branch-name">{{ gitStore.status.branch || '...' }}</span>
            <svg width="10" height="10" viewBox="0 0 12 12" fill="currentColor" class="chevron">
              <path d="M6 8.825a.5.5 0 0 1-.354-.146l-3.5-3.5a.5.5 0 1 1 .708-.708L6 7.618l3.146-3.147a.5.5 0 1 1 .708.708l-3.5 3.5A.5.5 0 0 1 6 8.825z"/>
            </svg>
          </button>
          <div v-if="branchDropdownOpen" class="branch-dropdown">
            <div class="branch-dropdown-header">Switch branch</div>
            <button
              v-for="branch in gitStore.branches"
              :key="branch.name"
              class="branch-option"
              :class="{ 'branch-active': branch.is_current }"
              @click="selectBranch(branch.name)"
            >
              <svg
                v-if="branch.is_current"
                width="14" height="14" viewBox="0 0 16 16" fill="currentColor" class="check-icon"
              >
                <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 0 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0z"/>
              </svg>
              <span v-else class="check-placeholder" />
              {{ branch.name }}
            </button>
          </div>
        </div>
        <span class="status-text">{{ statusSummary }}</span>
        <span v-if="gitStore.viewingBranch" class="viewing-indicator">
          <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor" style="flex-shrink:0">
            <path d="M1.5 8a6.5 6.5 0 1 1 13 0 6.5 6.5 0 0 1-13 0zM8 0a8 8 0 1 0 0 16A8 8 0 0 0 8 0zm.75 4.75a.75.75 0 0 0-1.5 0v2.5h-2.5a.75.75 0 0 0 0 1.5h2.5v2.5a.75.75 0 0 0 1.5 0v-2.5h2.5a.75.75 0 0 0 0-1.5h-2.5v-2.5z"/>
          </svg>
          viewing: {{ gitStore.viewingBranch }}
          <span class="viewing-clear" @click.stop="gitStore.setViewingBranch('')">&times;</span>
        </span>
      </div>
      <div class="top-bar-actions">
        <button
          class="btn btn-action"
          :disabled="gitStore.loading.pull"
          @click.stop="gitStore.pull()"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M7.47 12.78a.75.75 0 0 0 1.06 0l3.25-3.25a.75.75 0 0 0-1.06-1.06L8.75 10.44V1.75a.75.75 0 0 0-1.5 0v8.69L5.28 8.47a.75.75 0 0 0-1.06 1.06l3.25 3.25zM3.75 13a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5z"/>
          </svg>
          {{ gitStore.loading.pull ? 'Pulling...' : 'Pull' }}
        </button>
        <button
          class="btn btn-action"
          :disabled="gitStore.loading.push"
          @click.stop="gitStore.push()"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8.53 1.22a.75.75 0 0 0-1.06 0L4.22 4.47a.75.75 0 0 0 1.06 1.06L7.25 3.56v8.69a.75.75 0 0 0 1.5 0V3.56l1.97 1.97a.75.75 0 1 0 1.06-1.06L8.53 1.22zM3.75 13a.75.75 0 0 0 0 1.5h8.5a.75.75 0 0 0 0-1.5h-8.5z"/>
          </svg>
          {{ gitStore.loading.push ? 'Pushing...' : 'Push' }}
        </button>
      </div>
    </header>

    <!-- SubTabs -->
    <nav class="sub-tabs">
      <button
        class="sub-tab"
        :class="{ 'sub-tab-active': gitStore.activeTab === 'changes' }"
        @click="gitStore.activeTab = 'changes'"
      >
        Changes
        <span v-if="tabCounts.changes > 0" class="tab-count">{{ tabCounts.changes }}</span>
      </button>
      <button
        class="sub-tab"
        :class="{ 'sub-tab-active': gitStore.activeTab === 'log' }"
        @click="gitStore.activeTab = 'log'"
      >
        Log
        <span v-if="tabCounts.log > 0" class="tab-count">{{ tabCounts.log }}</span>
      </button>
      <button
        class="sub-tab"
        :class="{ 'sub-tab-active': gitStore.activeTab === 'branches' }"
        @click="gitStore.activeTab = 'branches'"
      >
        Branches
        <span v-if="tabCounts.branches > 0" class="tab-count">{{ tabCounts.branches }}</span>
      </button>
    </nav>

    <!-- Tab Content -->
    <div class="tab-content">

      <!-- ==================== TAB: CHANGES ==================== -->
      <div v-if="gitStore.activeTab === 'changes'" class="changes-layout" :style="{ 'grid-template-columns': filesPanelWidth + 'px 4px 1fr' }">
        <!-- Left: File list -->
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
                <button class="file-group-action" title="Unstage all" @click.stop="unstageAll">
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
                  @click="selectFile(entry.file)"
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
                <button class="file-group-action" title="Select all" @click.stop="selectAllForStage">
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
                  @click="selectFile(entry.file)"
                >
                  <input
                    type="checkbox"
                    class="file-checkbox"
                    :checked="gitStore.isSelected(entry.file)"
                    @click.stop
                    @change="toggleFileCheck(entry.file)"
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
              <button class="stash-push-btn" title="Stash changes" @click.stop="openStashDialog">
                + Stash
              </button>
            </div>
            <template v-if="!stashCollapsed">
              <div
                v-for="entry in gitStore.stashEntries"
                :key="'stash-' + entry.index"
                class="stash-item"
                :class="{ 'stash-item-active': selectedStashIndex === entry.index }"
                @click="selectStash(entry.index)"
              >
                <div class="stash-item-top">
                  <span class="stash-index">stash@{{ '{' }}{{ entry.index }}{{ '}' }}</span>
                  <span class="stash-message">{{ entry.message }}</span>
                </div>
                <div class="stash-item-bottom">
                  <span class="stash-date">{{ formatRelativeDate(entry.date) }}</span>
                  <div class="stash-actions">
                    <button class="stash-action-btn apply" title="Apply" @click.stop="doStashApply(entry.index)">Apply</button>
                    <button class="stash-action-btn pop" title="Pop" @click.stop="doStashPop(entry.index)">Pop</button>
                    <button class="stash-action-btn drop" title="Drop" @click.stop="doStashDrop(entry.index)">Drop</button>
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
              <button class="stash-push-btn" title="Stash changes" @click.stop="openStashDialog">
                + Stash
              </button>
            </div>
          </div>
        </div>

        <!-- Resize handle -->
        <div class="resize-handle" @mousedown.prevent="startResize"></div>

        <!-- Right: Diff viewer -->
        <div class="diff-panel">
          <!-- Stash diff -->
          <template v-if="selectedStashIndex !== null && stashDiffContent">
            <div class="diff-header">
              <span class="diff-filename stash-diff-label">stash@{{ '{' }}{{ selectedStashIndex }}{{ '}' }} diff</span>
              <button class="stash-diff-close" @click="selectedStashIndex = null; stashDiffContent = ''">&times;</button>
            </div>
            <div class="diff-content">
              <div
                v-for="(line, idx) in parsedStashDiff"
                :key="idx"
                class="diff-line"
                :class="{
                  'diff-line-add': line.type === 'add',
                  'diff-line-remove': line.type === 'remove',
                  'diff-line-header': line.type === 'header',
                  'diff-line-context': line.type === 'context',
                }"
              >
                <span class="diff-line-no old-no">{{ line.oldLineNo ?? '' }}</span>
                <span class="diff-line-no new-no">{{ line.newLineNo ?? '' }}</span>
                <span class="diff-line-prefix">{{
                  line.type === 'add' ? '+' : line.type === 'remove' ? '-' : line.type === 'header' ? '' : ' '
                }}</span>
                <span class="diff-line-text">{{ line.content }}</span>
              </div>
            </div>
          </template>
          <!-- File diff -->
          <template v-else-if="selectedFile && gitStore.diff">
            <div class="diff-header">
              <span class="diff-filename">{{ selectedFile }}</span>
            </div>
            <div class="diff-content">
              <div v-if="gitStore.loading.diff" class="diff-loading">
                Loading diff...
              </div>
              <template v-else>
                <div
                  v-for="(line, idx) in parsedDiff"
                  :key="idx"
                  class="diff-line"
                  :class="{
                    'diff-line-add': line.type === 'add',
                    'diff-line-remove': line.type === 'remove',
                    'diff-line-header': line.type === 'header',
                    'diff-line-context': line.type === 'context',
                  }"
                >
                  <span class="diff-line-no old-no">{{ line.oldLineNo ?? '' }}</span>
                  <span class="diff-line-no new-no">{{ line.newLineNo ?? '' }}</span>
                  <span class="diff-line-prefix">{{
                    line.type === 'add' ? '+' : line.type === 'remove' ? '-' : line.type === 'header' ? '' : ' '
                  }}</span>
                  <span class="diff-line-text">{{ line.content }}</span>
                </div>
              </template>
            </div>
          </template>
          <div v-else class="diff-placeholder">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="diff-placeholder-icon">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
              <line x1="16" y1="13" x2="8" y2="13"/>
              <line x1="16" y1="17" x2="8" y2="17"/>
              <polyline points="10 9 9 9 8 9"/>
            </svg>
            <span class="placeholder-title">Select a file to view diff</span>
            <span v-if="totalChanges > 0" class="placeholder-summary">
              {{ totalChanges }} file{{ totalChanges !== 1 ? 's' : '' }} changed
            </span>
            <span v-else class="placeholder-summary">Working tree is clean</span>
          </div>
        </div>

        <!-- Bottom: Commit bar -->
        <div class="commit-bar">
          <button
            v-if="gitStore.selectedFiles.size > 0"
            class="btn btn-stage"
            @click="stageSelected"
          >
            Add {{ gitStore.selectedFiles.size }} files
          </button>
          <span class="commit-bar-staged">
            {{ gitStore.status.staged?.length ?? 0 }} staged
          </span>
          <button
            class="btn-generate"
            :disabled="gitStore.generatingMessage"
            title="Generate commit message with Claude"
            @click="doGenerateCommit"
          >
            <svg
              v-if="gitStore.generatingMessage"
              class="spinner"
              width="14" height="14" viewBox="0 0 16 16" fill="none"
            >
              <circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="2" stroke-dasharray="28" stroke-dashoffset="8" />
            </svg>
            <span v-else class="generate-icon">&#10024;</span>
          </button>
          <input
            v-model="gitStore.commitMessage"
            class="commit-input"
            type="text"
            placeholder="fix: describe your changes"
            @keydown.enter="doCommit"
          />
          <button
            class="btn btn-commit"
            :disabled="!canCommit() || gitStore.loading.commit"
            @click="doCommit"
          >
            {{ gitStore.loading.commit ? 'Committing...' : 'Commit' }}
          </button>
        </div>
      </div>

      <!-- ==================== TAB: LOG ==================== -->
      <div v-if="gitStore.activeTab === 'log'" class="log-layout">
        <!-- Branch filter sidebar -->
        <div class="branch-filter-sidebar">
          <div class="sidebar-header">Branches</div>
          <input
            class="sidebar-search"
            v-model="branchSearch"
            placeholder="Filter..."
          />
          <div class="sidebar-list">
            <div
              class="sidebar-item"
              :class="{ 'sidebar-item-active': gitStore.viewingBranch === '' }"
              @click="gitStore.setViewingBranch('')"
            >
              <span class="sidebar-dot dot-blue"></span>
              All branches
            </div>
            <div
              v-for="b in filteredBranches"
              :key="b.name"
              class="sidebar-item"
              :class="{ 'sidebar-item-active': gitStore.viewingBranch === b.name }"
              @click="gitStore.setViewingBranch(b.name)"
            >
              <span class="sidebar-dot" :class="b.is_current ? 'dot-green' : 'dot-purple'"></span>
              <span class="sidebar-item-name">{{ b.name }}</span>
              <span v-if="b.is_current" class="sidebar-badge">HEAD</span>
            </div>
          </div>
        </div>

        <!-- Log content -->
        <div class="log-main" @scroll="onLogScroll">
          <div v-if="gitStore.viewingBranch" class="log-branch-banner">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" style="flex-shrink:0">
              <path d="M11.75 2.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zm-2.25.75a2.25 2.25 0 1 1 3 2.122V6A2.5 2.5 0 0 1 10 8.5H6a1 1 0 0 0-1 1v1.128a2.251 2.251 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.5 0v1.836A2.492 2.492 0 0 1 6 7h4a1 1 0 0 0 1-1v-.628A2.25 2.25 0 0 1 9.5 3.25zM4.25 12a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zM3.5 3.25a.75.75 0 1 1 1.5 0 .75.75 0 0 1-1.5 0z"/>
            </svg>
            <span>Showing commits from <strong>{{ gitStore.viewingBranch }}</strong></span>
            <button class="log-banner-clear" @click="gitStore.setViewingBranch('')">&times;</button>
          </div>
          <div v-if="gitStore.totalCommits === 0 && !gitStore.loading.log" class="empty-state">
            <span class="empty-text">No commits</span>
          </div>
          <div v-else class="log-virtual-container" :style="{ height: totalHeight + 'px', position: 'relative' }">
            <div :style="{ transform: `translateY(${offsetY}px)` }">
              <div
                v-for="node in visibleNodes"
                :key="node.id"
                class="log-row"
                :class="{
                  'log-row-selected': gitStore.selectedCommit?.hash === node.id,
                }"
                @click="selectCommit(node.id)"
              >
                <div class="log-commit-col">
                  <template v-if="gitStore.getMetadata(node.id)">
                    <span class="log-hash">{{ gitStore.getMetadata(node.id)!.short_hash }}</span>
                    <span v-for="r in (gitStore.getMetadata(node.id)!.refs || [])" :key="r" class="ref-badge" :class="getRefClass(r)">{{ getRefLabel(r) }}</span>
                    <span class="log-msg">{{ gitStore.getMetadata(node.id)!.message }}</span>
                  </template>
                  <template v-else>
                    <span class="log-hash skeleton-text">-------</span>
                    <span class="log-msg skeleton-text">Loading...</span>
                  </template>
                </div>
                <div class="log-meta-col">
                  <template v-if="gitStore.getMetadata(node.id)">
                    <span class="log-author">{{ gitStore.getMetadata(node.id)!.author }}</span>
                    <span class="log-time">{{ gitStore.getMetadata(node.id)!.date }}</span>
                  </template>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Commit detail panel -->
        <div v-if="gitStore.selectedCommit" class="commit-detail-panel">
          <div class="commit-detail-header">
            <span class="commit-detail-title">Commit Details</span>
            <button class="commit-detail-close" @click="gitStore.selectedCommit = null">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
                <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06z"/>
              </svg>
            </button>
          </div>
          <div class="commit-detail-body">
            <!-- Message as heading -->
            <div class="detail-message">{{ gitStore.selectedCommit.message }}</div>

            <!-- Body -->
            <div v-if="gitStore.selectedCommit.body" class="detail-body-text">{{ gitStore.selectedCommit.body }}</div>

            <!-- Meta row: hash + date + author -->
            <div class="detail-meta">
              <div class="detail-meta-row">
                <span
                  class="detail-hash"
                  :title="hashCopied ? 'Copied!' : 'Click to copy full hash'"
                  @click="copyToClipboard(gitStore.selectedCommit.hash)"
                >
                  {{ hashCopied ? 'Copied!' : gitStore.selectedCommit.hash.slice(0, 7) }}
                </span>
                <span class="detail-date">{{ formatDate(gitStore.selectedCommit.date) }}</span>
              </div>
              <div class="detail-author-row">
                <span class="detail-author-name">{{ gitStore.selectedCommit.author }}</span>
                <span class="detail-author-email">{{ gitStore.selectedCommit.email }}</span>
              </div>
            </div>

            <!-- Stats - parsed into visual bars -->
            <div v-if="gitStore.selectedCommit.stats" class="detail-stats">
              <div class="detail-files-header">
                Changes
                <span v-if="parseStats(gitStore.selectedCommit.stats).summary" class="detail-stats-summary">
                  {{ parseStats(gitStore.selectedCommit.stats).summary }}
                </span>
              </div>
              <div
                v-for="stat in parseStats(gitStore.selectedCommit.stats).lines"
                :key="stat.file"
                class="detail-stat-line"
                @click="viewCommitFileDiff(gitStore.selectedCommit!.hash, stat.file)"
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

            <!-- Changed files (fallback if no stats / always show full paths) -->
            <div class="detail-files-header">Files</div>
            <div
              v-for="f in gitStore.selectedCommit.files"
              :key="f.path"
              class="detail-file-item"
              @click="viewCommitFileDiff(gitStore.selectedCommit!.hash, f.path)"
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
      </div>

      <!-- ==================== TAB: BRANCHES ==================== -->
      <div v-if="gitStore.activeTab === 'branches'" class="branches-layout">
        <!-- Search and filter bar -->
        <div class="branches-toolbar">
          <input
            class="branches-search"
            v-model="branchTabSearch"
            placeholder="Search branches..."
          />
          <div class="branches-filter-group">
            <button
              class="branches-filter-btn"
              :class="{ 'branches-filter-active': branchTabFilter === 'local' }"
              @click="branchTabFilter = 'local'"
            >Local</button>
            <button
              class="branches-filter-btn"
              :class="{ 'branches-filter-active': branchTabFilter === 'remote' }"
              @click="branchTabFilter = 'remote'"
            >Remote</button>
            <button
              class="branches-filter-btn"
              :class="{ 'branches-filter-active': branchTabFilter === 'all' }"
              @click="branchTabFilter = 'all'"
            >All</button>
          </div>
        </div>

        <div v-if="filteredBranchCards.length === 0" class="empty-state">
          <span class="empty-text">No branches</span>
        </div>
        <div v-else class="branches-list">
          <div
            v-for="branch in filteredBranchCards"
            :key="branch.name"
            class="branch-card"
            :class="{
              'branch-card-current': branch.is_current,
              'branch-card-expanded': expandedBranch === branch.name,
            }"
          >
            <div class="branch-card-top" @click="toggleBranchExpand(branch.name)">
              <div class="branch-card-left">
                <svg
                  width="10" height="10" viewBox="0 0 12 12" fill="currentColor"
                  class="branch-expand-chevron"
                  :class="{ 'branch-expand-chevron-open': expandedBranch === branch.name }"
                >
                  <path d="M4.7 2.4a.5.5 0 0 1 .7 0l3.15 3.15a.5.5 0 0 1 0 .7L5.4 9.4a.5.5 0 0 1-.7-.7L7.54 5.85 4.7 3.1a.5.5 0 0 1 0-.7z"/>
                </svg>
                <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" class="branch-card-icon">
                  <path d="M11.75 2.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zm-2.25.75a2.25 2.25 0 1 1 3 2.122V6A2.5 2.5 0 0 1 10 8.5H6a1 1 0 0 0-1 1v1.128a2.251 2.251 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.5 0v1.836A2.492 2.492 0 0 1 6 7h4a1 1 0 0 0 1-1v-.628A2.25 2.25 0 0 1 9.5 3.25zM4.25 12a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5zM3.5 3.25a.75.75 0 1 1 1.5 0 .75.75 0 0 1-1.5 0z"/>
                </svg>
                <span class="branch-card-name-text">{{ branch.name }}</span>
              </div>
              <div class="branch-card-badges">
                <span v-if="branch.is_current" class="badge badge-current">CURRENT</span>
                <span v-if="branch.is_merged" class="badge badge-merged">MERGED</span>
                <span v-if="branch.ahead > 0" class="badge badge-ahead">{{ branch.ahead }} ahead</span>
                <span v-if="branch.behind > 0" class="badge badge-behind">{{ branch.behind }} behind</span>
              </div>
            </div>

            <div v-if="branch.message || branch.date" class="branch-card-meta">
              <span v-if="branch.short_hash" class="branch-card-hash">{{ branch.short_hash }}</span>
              <span v-if="branch.message" class="branch-card-msg">{{ branch.message }}</span>
              <span v-if="branch.date" class="branch-card-date">{{ branch.date }}</span>
              <span v-if="branch.author" class="branch-card-author">{{ branch.author }}</span>
            </div>

            <!-- Expanded content: recent commits -->
            <div v-if="expandedBranch === branch.name" class="branch-card-expanded-content">
              <div class="branch-commits-header">Recent commits</div>
              <div v-if="!gitStore.branchCommits.get(branch.name)" class="branch-commits-loading">
                Loading...
              </div>
              <div v-else-if="gitStore.branchCommits.get(branch.name)!.length === 0" class="branch-commits-empty">
                No commits
              </div>
              <div v-else class="branch-commits-preview">
                <div
                  v-for="c in gitStore.branchCommits.get(branch.name)"
                  :key="c.hash"
                  class="branch-commit-row"
                  @click.stop="gitStore.activeTab = 'log'; selectCommit(c.hash)"
                >
                  <span class="branch-commit-hash">{{ c.short_hash }}</span>
                  <span class="branch-commit-msg">{{ c.message }}</span>
                  <span class="branch-commit-date">{{ c.date }}</span>
                </div>
              </div>

              <!-- Action buttons -->
              <div class="branch-card-actions">
                <button class="btn btn-sm btn-view-log" @click.stop="viewBranchLog(branch.name)">
                  <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M1.5 1.75V13.5h13.75a.75.75 0 0 1 0 1.5H.75a.75.75 0 0 1-.75-.75V1.75a.75.75 0 0 1 1.5 0zm14.28 2.53-5.25 5.25a.75.75 0 0 1-1.06 0L7 7.06 4.28 9.78a.75.75 0 0 1-1.06-1.06l3.25-3.25a.75.75 0 0 1 1.06 0L10 7.94l4.72-4.72a.75.75 0 1 1 1.06 1.06z"/>
                  </svg>
                  View Log
                </button>
                <button
                  v-if="!branch.is_current"
                  class="btn btn-sm btn-checkout-action"
                  @click.stop="showCheckoutConfirm = branch.name"
                >
                  <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.75.75 0 0 1 1.06-1.06L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0z"/>
                  </svg>
                  Checkout
                </button>
                <button
                  v-if="branch.is_merged && !branch.is_current"
                  class="btn btn-sm btn-danger"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Checkout confirmation dialog -->
    <Teleport to="body">
      <div v-if="showCheckoutConfirm" class="confirm-overlay" @click.self="showCheckoutConfirm = null">
        <div class="confirm-dialog">
          <div class="confirm-title">Checkout branch</div>
          <div class="confirm-text">Switch to <code>{{ showCheckoutConfirm }}</code>?</div>
          <div class="confirm-warning">Uncommitted changes may be lost</div>
          <div class="confirm-actions">
            <button class="btn" @click="showCheckoutConfirm = null">Cancel</button>
            <button class="btn btn-checkout" @click="confirmCheckout(showCheckoutConfirm!)">Checkout</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Commit diff modal -->
    <div v-if="showCommitDiffModal" class="modal-overlay" @click.self="closeCommitDiffModal">
      <div class="modal-content">
        <div class="modal-header">
          <span class="modal-title">{{ commitDiffFile }}</span>
          <button class="modal-close" @click="closeCommitDiffModal">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06z"/>
            </svg>
          </button>
        </div>
        <div class="modal-body diff-content">
          <div
            v-for="(line, idx) in parsedCommitDiff"
            :key="idx"
            class="diff-line"
            :class="{
              'diff-line-add': line.type === 'add',
              'diff-line-remove': line.type === 'remove',
              'diff-line-header': line.type === 'header',
              'diff-line-context': line.type === 'context',
            }"
          >
            <span class="diff-line-no old-no">{{ line.oldLineNo ?? '' }}</span>
            <span class="diff-line-no new-no">{{ line.newLineNo ?? '' }}</span>
            <span class="diff-line-prefix">{{
              line.type === 'add' ? '+' : line.type === 'remove' ? '-' : line.type === 'header' ? '' : ' '
            }}</span>
            <span class="diff-line-text">{{ line.content }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Stash dialog -->
    <Teleport to="body">
      <div v-if="showStashDialog" class="confirm-overlay" @click.self="showStashDialog = false">
        <div class="stash-dialog">
          <div class="stash-dialog-title">Stash Changes</div>
          <input
            v-model="stashMessage"
            class="stash-dialog-input"
            type="text"
            placeholder="Stash message (optional)"
            @keydown.enter="doStashPush"
          />
          <div class="stash-dialog-actions">
            <button class="btn" @click="showStashDialog = false">Cancel</button>
            <button
              class="btn btn-stash-confirm"
              :disabled="gitStore.stashLoading"
              @click="doStashPush"
            >
              {{ gitStore.stashLoading ? 'Stashing...' : 'Stash' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Error bar -->
    <div v-if="gitStore.error" class="error-bar" @click="gitStore.error = null">
      {{ gitStore.error }}
    </div>
  </div>
</template>

<style scoped>
.git-view {
  display: flex;
  flex-direction: column;
  gap: 0;
  overflow: hidden;
}

/* ===== TopBar ===== */

.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.top-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.top-bar-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
}

.status-text {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.top-bar-actions {
  display: flex;
  gap: 8px;
}

/* Branch selector */
.branch-selector {
  position: relative;
}

.branch-button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--border);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: border-color 0.15s;
}

.branch-button:hover {
  border-color: var(--accent-blue);
}

.branch-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.branch-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.chevron {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.branch-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  min-width: 220px;
  max-height: 300px;
  overflow-y: auto;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  z-index: 100;
}

.branch-dropdown-header {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.branch-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  text-align: left;
}

.branch-option:hover {
  background: var(--border);
}

.branch-active {
  font-weight: 600;
}

.check-icon {
  color: var(--accent-blue);
  flex-shrink: 0;
}

.check-placeholder {
  width: 14px;
  flex-shrink: 0;
}

/* ===== SubTabs ===== */

.sub-tabs {
  display: flex;
  gap: 0;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  padding: 0 16px;
}

.sub-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
  white-space: nowrap;
}

.sub-tab:hover {
  color: var(--text-primary);
}

.sub-tab-active {
  color: var(--text-primary);
  border-bottom-color: var(--accent-orange);
}

.tab-count {
  background: var(--border);
  color: var(--text-primary);
  padding: 0 6px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
}

/* ===== Tab Content ===== */

.tab-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

/* ===== Changes Layout ===== */

.changes-layout {
  display: grid;
  grid-template-columns: 320px 4px 1fr;
  grid-template-rows: 1fr auto;
  height: 100%;
}

.files-panel {
  grid-row: 1;
  grid-column: 1;
  background: var(--bg-secondary);
  overflow-y: auto;
}

.resize-handle {
  grid-row: 1;
  grid-column: 2;
  background: var(--border);
  cursor: col-resize;
  transition: background 0.15s;
  position: relative;
}

.resize-handle::after {
  content: '';
  position: absolute;
  inset: -2px -4px;
}

.resize-handle:hover {
  background: var(--accent-blue);
}

.diff-panel {
  grid-row: 1;
  grid-column: 3;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
  min-width: 0;
}

.commit-bar {
  grid-row: 2;
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.btn-stage {
  padding: 4px 12px;
  background: var(--accent-green);
  border: 1px solid var(--accent-green);
  border-radius: 6px;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}

.btn-stage:hover {
  background: var(--accent-green);
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

/* Diff viewer */
.diff-header {
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.diff-filename {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 600;
}

.diff-content {
  flex: 1;
  overflow: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 20px;
}

.diff-line {
  display: flex;
  align-items: stretch;
  min-height: 20px;
  padding-right: 16px;
}

.diff-line-no {
  display: inline-block;
  width: 50px;
  min-width: 50px;
  text-align: right;
  padding-right: 8px;
  color: var(--text-secondary);
  user-select: none;
  flex-shrink: 0;
  font-size: 12px;
  line-height: 20px;
}

.diff-line-prefix {
  display: inline-block;
  width: 16px;
  min-width: 16px;
  text-align: center;
  flex-shrink: 0;
  user-select: none;
  line-height: 20px;
}

.diff-line-text {
  flex: 1;
  white-space: pre;
  overflow: hidden;
  line-height: 20px;
}

.diff-line-add {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.diff-line-add .diff-line-no {
  background: rgba(63, 185, 80, 0.1);
  color: var(--accent-green);
}

.diff-line-add .diff-line-prefix {
  color: var(--accent-green);
}

.diff-line-remove {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
}

.diff-line-remove .diff-line-no {
  background: rgba(248, 81, 73, 0.1);
  color: var(--accent-red);
}

.diff-line-remove .diff-line-prefix {
  color: var(--accent-red);
}

.diff-line-header {
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  font-weight: 600;
  padding: 4px 0;
}

.diff-line-header .diff-line-no {
  background: transparent;
}

.diff-line-context {
  color: var(--text-secondary);
}

.diff-loading {
  padding: 24px;
  text-align: center;
  color: var(--text-secondary);
}

.diff-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
  font-size: 14px;
}

.diff-placeholder-icon {
  opacity: 0.4;
}

.placeholder-title {
  font-size: 14px;
  color: var(--text-secondary);
}

.placeholder-summary {
  font-size: 12px;
  color: var(--border);
  font-family: var(--font-mono);
}

/* Commit bar */
.commit-bar-staged {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-generate {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--border);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  cursor: pointer;
  flex-shrink: 0;
  transition: border-color 0.15s, background 0.15s;
}

.btn-generate:hover:not(:disabled) {
  border-color: var(--accent-blue);
}

.btn-generate:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.generate-icon {
  font-size: 14px;
  line-height: 1;
}

.spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.commit-bar .commit-input {
  flex: 1;
  height: 32px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0 12px;
  color: var(--text-primary);
  font-size: 13px;
  min-width: 0;
}

.commit-bar .commit-input:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.2);
  outline: none;
}

.commit-bar .commit-input::placeholder {
  color: var(--text-secondary);
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: var(--border);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  white-space: nowrap;
}

.btn:hover:not(:disabled) {
  background: var(--border);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-action svg {
  flex-shrink: 0;
}

.btn-sm {
  padding: 4px 10px;
  font-size: 12px;
}

.btn-commit {
  padding: 6px 20px;
  background: rgba(63, 185, 80, 0.15);
  border: 1px solid var(--accent-green);
  color: var(--accent-green);
  font-weight: 600;
  flex-shrink: 0;
}

.btn-commit:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.25);
}

.btn-commit:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.btn-danger {
  color: var(--accent-red);
  border-color: var(--accent-red);
  background: rgba(248, 81, 73, 0.1);
}

.btn-danger:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.2);
}

/* ===== Log Layout ===== */

.log-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.log-main {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
}

.log-list {
  display: flex;
  flex-direction: column;
}

.log-row {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0 12px;
  height: 28px;
  cursor: pointer;
  transition: background 0.1s;
  border-bottom: 1px solid transparent;
}

.log-row:hover {
  background: var(--bg-tertiary);
}

.log-row-selected {
  background: rgba(31, 111, 235, 0.08);
  border-bottom-color: var(--border);
}

.log-commit-col {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  min-width: 0;
}

.log-hash {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-blue);
  flex-shrink: 0;
  font-weight: 500;
}

.log-msg {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
  font-size: 13px;
}

.log-meta-col {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  padding-left: 12px;
}

.log-author {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-time {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
  white-space: nowrap;
}


.log-virtual-container {
  will-change: transform;
}

.skeleton-text {
  color: var(--border);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 0.6; }
}

.shimmer-pad {
  padding: 12px;
}

.log-commit-col {
  animation: data-appear 0.2s ease;
}

/* Ref badges */
.ref-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.ref-branch {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  border: 1px solid rgba(88, 166, 255, 0.3);
}

.ref-tag {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
  border: 1px solid rgba(63, 185, 80, 0.3);
}

.ref-hotfix {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
  border: 1px solid rgba(248, 81, 73, 0.3);
}

.ref-head {
  background: rgba(188, 140, 255, 0.15);
  color: var(--accent-purple);
  border: 1px solid rgba(188, 140, 255, 0.3);
}

/* Commit detail panel */
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
  padding: 4px 0;
  cursor: pointer;
  transition: background 0.1s;
  border-radius: 4px;
  padding: 4px 6px;
  margin: 0 -6px;
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

/* ===== Branch Filter Sidebar (Log tab) ===== */

.branch-filter-sidebar {
  width: 200px;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 10px 12px;
  font-size: 11px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.sidebar-search {
  width: 100%;
  padding: 6px 12px;
  background: var(--bg-primary);
  border: none;
  border-bottom: 1px solid var(--border);
  color: var(--text-primary);
  font-size: 12px;
  outline: none;
  flex-shrink: 0;
}

.sidebar-search::placeholder {
  color: var(--text-secondary);
}

.sidebar-search:focus {
  background: var(--bg-secondary);
}

.sidebar-list {
  flex: 1;
  overflow-y: auto;
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: 12px;
  color: var(--text-primary);
  cursor: pointer;
  transition: background 0.1s;
  overflow: hidden;
}

.sidebar-item:hover {
  background: var(--bg-tertiary);
}

.sidebar-item-active {
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  border-left: 2px solid var(--accent-blue);
  padding-left: 10px;
}

.sidebar-item-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.sidebar-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-green { background: var(--accent-green); }
.dot-purple { background: var(--accent-purple); }
.dot-blue { background: var(--accent-blue); }

.sidebar-badge {
  font-size: 9px;
  font-weight: 700;
  background: rgba(63, 185, 80, 0.2);
  color: var(--accent-green);
  padding: 1px 5px;
  border-radius: 3px;
  flex-shrink: 0;
}

/* Viewing indicator (top bar) */
.viewing-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  background: rgba(88, 166, 255, 0.1);
  border: 1px solid rgba(88, 166, 255, 0.3);
  border-radius: 12px;
  font-size: 12px;
  color: var(--accent-blue);
  white-space: nowrap;
}

.viewing-clear {
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  opacity: 0.7;
  transition: opacity 0.15s;
}

.viewing-clear:hover {
  opacity: 1;
}

/* Log branch banner */
.log-branch-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: rgba(88, 166, 255, 0.06);
  border-bottom: 1px solid rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  font-size: 12px;
  flex-shrink: 0;
}

.log-branch-banner strong {
  font-weight: 600;
}

.log-banner-clear {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--accent-blue);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  opacity: 0.6;
  padding: 0 4px;
  transition: opacity 0.15s;
}

.log-banner-clear:hover {
  opacity: 1;
}

/* ===== Branches Layout ===== */

.branches-layout {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
}

.branches-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.branches-search {
  flex: 1;
  height: 32px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0 12px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
}

.branches-search::placeholder {
  color: var(--text-secondary);
}

.branches-search:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.15);
}

.branches-filter-group {
  display: flex;
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
  flex-shrink: 0;
}

.branches-filter-btn {
  padding: 5px 12px;
  background: var(--border);
  border: none;
  border-right: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.branches-filter-btn:last-child {
  border-right: none;
}

.branches-filter-btn:hover {
  color: var(--text-primary);
}

.branches-filter-active {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
}

.branches-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.branch-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px 16px;
  transition: border-color 0.15s;
}

.branch-card:hover {
  border-color: var(--text-secondary);
}

.branch-card-current {
  border-color: var(--accent-green);
  border-left: 3px solid var(--accent-green);
}

.branch-card-expanded {
  border-color: var(--accent-blue);
}

.branch-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  cursor: pointer;
}

.branch-card-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.branch-expand-chevron {
  color: var(--text-secondary);
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.branch-expand-chevron-open {
  transform: rotate(90deg);
}

.branch-card-name-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.branch-card-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.branch-card-badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.badge-current {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.badge-merged {
  background: rgba(188, 140, 255, 0.15);
  color: var(--accent-purple);
}

.badge-ahead {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.badge-behind {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.branch-card-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.branch-card-hash {
  font-family: var(--font-mono);
  color: var(--accent-blue);
}

.branch-card-msg {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.branch-card-date {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.branch-card-author {
  color: var(--text-secondary);
  flex-shrink: 0;
}

/* Branch card expanded content */
.branch-card-expanded-content {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.branch-commits-header {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.branch-commits-loading,
.branch-commits-empty {
  font-size: 12px;
  color: var(--text-secondary);
  padding: 8px 0;
}

.branch-commits-loading {
  animation: pulse 1.5s ease-in-out infinite;
}

.branch-commits-preview {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: 12px;
}

.branch-commit-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.1s;
}

.branch-commit-row:hover {
  background: var(--bg-tertiary);
}

.branch-commit-hash {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--accent-blue);
  flex-shrink: 0;
  font-weight: 500;
}

.branch-commit-msg {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.branch-commit-date {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
  white-space: nowrap;
}

.branch-card-actions {
  display: flex;
  gap: 6px;
  margin-top: 10px;
}

.btn-view-log {
  background: rgba(88, 166, 255, 0.1);
  border-color: rgba(88, 166, 255, 0.3);
  color: var(--accent-blue);
}

.btn-view-log:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.2);
}

.btn-checkout-action {
  background: rgba(63, 185, 80, 0.1);
  border-color: rgba(63, 185, 80, 0.3);
  color: var(--accent-green);
}

.btn-checkout-action:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.2);
}

/* ===== Checkout Confirm Dialog ===== */

.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 400;
}

.confirm-dialog {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 380px;
  max-width: 90vw;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.confirm-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.confirm-text {
  font-size: 14px;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.confirm-text code {
  font-family: var(--font-mono);
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}

.confirm-warning {
  font-size: 12px;
  color: var(--accent-orange);
  margin-bottom: 20px;
  padding: 8px 10px;
  background: rgba(210, 153, 34, 0.1);
  border-radius: 6px;
  border-left: 2px solid var(--accent-orange);
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-checkout {
  background: rgba(63, 185, 80, 0.15);
  border-color: var(--accent-green);
  color: var(--accent-green);
  font-weight: 600;
}

.btn-checkout:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.25);
}

/* ===== Modal ===== */

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.modal-content {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 12px;
  width: 80vw;
  max-width: 900px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.modal-title {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 600;
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
}

.modal-close:hover {
  background: var(--border);
  color: var(--text-primary);
}

.modal-body {
  flex: 1;
  overflow: auto;
}

/* ===== Empty state ===== */

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.empty-text {
  color: var(--text-secondary);
  font-size: 13px;
}

/* ===== Error bar ===== */

.error-bar {
  position: fixed;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  padding: 10px 20px;
  background: rgba(248, 81, 73, 0.15);
  border: 1px solid var(--accent-red);
  border-radius: 8px;
  color: var(--accent-red);
  font-size: 13px;
  cursor: pointer;
  z-index: 200;
  max-width: 600px;
  text-align: center;
}

/* ===== Stash Section ===== */

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

/* Stash diff header label */
.stash-diff-label {
  color: var(--accent-purple);
}

.stash-diff-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  padding: 0 4px;
  transition: color 0.15s;
}

.stash-diff-close:hover {
  color: var(--text-primary);
}

/* Stash dialog */
.stash-dialog {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 380px;
  max-width: 90vw;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.stash-dialog-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.stash-dialog-input {
  width: 100%;
  height: 36px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0 12px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
  margin-bottom: 20px;
  box-sizing: border-box;
}

.stash-dialog-input:focus {
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 2px rgba(188, 140, 255, 0.2);
}

.stash-dialog-input::placeholder {
  color: var(--text-secondary);
}

.stash-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-stash-confirm {
  background: rgba(188, 140, 255, 0.15);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
  font-weight: 600;
}

.btn-stash-confirm:hover:not(:disabled) {
  background: rgba(188, 140, 255, 0.25);
}

.btn-stash-confirm:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
</style>
