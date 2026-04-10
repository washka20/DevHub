<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useGitStore } from '../stores/git'
import { useProjectsStore } from '../stores/projects'
import { useProject } from '../composables/useProject'
import { gitApi } from '../api/git'
import GitChangesPanel from '../components/git/GitChangesPanel.vue'
import GitLogPanel from '../components/git/GitLogPanel.vue'
import GitBranchesPanel from '../components/git/GitBranchesPanel.vue'
import GitCommitDetail from '../components/git/GitCommitDetail.vue'
import GitBranchFilterSidebar from '../components/git/GitBranchFilterSidebar.vue'
import DiffViewer from '../components/git/DiffViewer.vue'

const gitStore = useGitStore()
const projectsStore = useProjectsStore()
const { switching } = useProject()

const selectedFile = ref<string | null>(null)
const branchDropdownOpen = ref(false)
const selectedStashIndex = ref<number | null>(null)
const stashDiffContent = ref('')
const commitDiffContent = ref('')
const commitDiffFile = ref<string | null>(null)
const showCommitDiffModal = ref(false)
const showCheckoutConfirm = ref<string | null>(null)
const showStashDialog = ref(false)
const stashMessage = ref('')

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

// Status summary
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

const totalChanges = computed(() =>
  (gitStore.status.modified?.length ?? 0)
  + (gitStore.status.untracked?.length ?? 0)
  + (gitStore.status.staged?.length ?? 0),
)

const tabCounts = computed(() => ({
  changes: totalChanges.value,
  log: gitStore.totalCommits,
  branches: gitStore.branches.length,
}))

// Actions
function selectFile(file: string) {
  selectedFile.value = file
  selectedStashIndex.value = null
  stashDiffContent.value = ''
  gitStore.fetchDiff(file)
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

async function confirmCheckout(name: string) {
  showCheckoutConfirm.value = null
  await gitStore.checkout(name)
}

// Stash
async function doStashPush() {
  await gitStore.stashPush(stashMessage.value)
  showStashDialog.value = false
  stashMessage.value = ''
}

async function selectStash(index: number) {
  selectedStashIndex.value = index
  selectedFile.value = null
  gitStore.diff = ''
  stashDiffContent.value = await gitStore.stashDiff(index)
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

function onClickOutside() {
  branchDropdownOpen.value = false
}

// Lifecycle
onMounted(() => {
  gitStore.fetchStatus()
  gitStore.fetchBranches()
  gitStore.fetchGraph()
})

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
        <GitChangesPanel
          :selected-file="selectedFile"
          :selected-stash-index="selectedStashIndex"
          @select-file="selectFile"
          @toggle-check="gitStore.toggleSelectFile($event)"
          @stage-selected="gitStore.stageSelected()"
          @select-all="gitStore.selectAllUnstaged()"
          @unstage-all="gitStore.unstageAll()"
          @select-stash="selectStash"
          @stash-apply="gitStore.stashApply($event)"
          @stash-pop="doStashPop"
          @stash-drop="doStashDrop"
          @open-stash-dialog="stashMessage = ''; showStashDialog = true"
        />

        <!-- Resize handle -->
        <div class="resize-handle" @mousedown.prevent="startResize"></div>

        <!-- Right: Diff viewer -->
        <div class="diff-panel">
          <!-- Stash diff -->
          <template v-if="selectedStashIndex !== null && stashDiffContent">
            <DiffViewer
              :diff="stashDiffContent"
              :filename="`stash@{${selectedStashIndex}} diff`"
            >
              <template #header-actions>
                <button class="stash-diff-close" @click="selectedStashIndex = null; stashDiffContent = ''">&times;</button>
              </template>
            </DiffViewer>
          </template>
          <!-- File diff -->
          <template v-else-if="selectedFile && gitStore.diff">
            <DiffViewer
              :diff="gitStore.diff"
              :filename="selectedFile"
              :loading="gitStore.loading.diff"
            />
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
            @click="gitStore.stageSelected()"
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
        <GitBranchFilterSidebar />
        <GitLogPanel @select-commit="selectCommit" />
        <GitCommitDetail
          v-if="gitStore.selectedCommit"
          :commit="gitStore.selectedCommit"
          @close="gitStore.selectedCommit = null"
          @view-file-diff="viewCommitFileDiff"
        />
      </div>

      <!-- ==================== TAB: BRANCHES ==================== -->
      <GitBranchesPanel
        v-if="gitStore.activeTab === 'branches'"
        @select-commit="selectCommit"
        @show-checkout-confirm="showCheckoutConfirm = $event"
      />
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
        <div class="modal-body">
          <DiffViewer :diff="commitDiffContent" />
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

.changes-layout > :first-child {
  grid-row: 1;
  grid-column: 1;
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

/* Diff placeholder */
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

/* ===== Log Layout ===== */

.log-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

/* ===== Viewing indicator ===== */
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

/* Stash diff close */
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

/* ===== Stash dialog ===== */

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
