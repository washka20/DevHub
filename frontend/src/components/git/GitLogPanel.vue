<script setup lang="ts">
import { computed } from 'vue'
import { useGitStore } from '../../stores/git'
import { useVirtualScroll } from '../../composables/useVirtualScroll'

const emit = defineEmits<{
  'select-commit': [hash: string]
}>()

const gitStore = useGitStore()

const ROW_HEIGHT = 28

const totalItems = computed(() => gitStore.totalCommits)
const { visibleRange, offsetY, totalHeight, onScroll } = useVirtualScroll(totalItems, ROW_HEIGHT)

const visibleNodes = computed(() =>
  gitStore.graphNodes.slice(visibleRange.value.startIdx, visibleRange.value.endIdx),
)

function handleScroll(e: Event) {
  onScroll(e)

  const { endIdx } = visibleRange.value
  if (endIdx > gitStore.metadataLoaded - 20 && !gitStore.metadataLoading) {
    gitStore.fetchMetadata(gitStore.metadataLoaded, 50)
  }
}

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
</script>

<template>
  <div class="log-main" @scroll="handleScroll">
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
          @click="emit('select-commit', node.id)"
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
</template>

<style scoped>
.log-main {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.empty-text {
  color: var(--text-secondary);
  font-size: 13px;
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
  animation: data-appear 0.2s ease;
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
</style>
