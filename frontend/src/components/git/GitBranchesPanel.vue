<script setup lang="ts">
import { ref, computed } from 'vue'
import { useGitStore } from '../../stores/git'

const emit = defineEmits<{
  'select-commit': [hash: string]
  'show-checkout-confirm': [branch: string]
}>()

const gitStore = useGitStore()

const expandedBranch = ref<string | null>(null)
const branchTabSearch = ref('')
const branchTabFilter = ref<'local' | 'remote' | 'all'>('local')

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
</script>

<template>
  <div class="branches-layout">
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
              @click.stop="gitStore.activeTab = 'log'; emit('select-commit', c.hash)"
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
              @click.stop="emit('show-checkout-confirm', branch.name)"
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
</template>

<style scoped>
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

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 0.6; }
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

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.empty-text {
  color: var(--text-secondary);
  font-size: 13px;
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

.btn-sm {
  padding: 4px 10px;
  font-size: 12px;
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

.btn-danger {
  color: var(--accent-red);
  border-color: var(--accent-red);
  background: rgba(248, 81, 73, 0.1);
}

.btn-danger:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.2);
}
</style>
