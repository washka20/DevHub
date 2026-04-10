<script setup lang="ts">
import { ref, computed } from 'vue'
import { useGitStore } from '../../stores/git'

const gitStore = useGitStore()

const branchSearch = ref('')

const filteredBranches = computed(() => {
  const q = branchSearch.value.toLowerCase()
  return gitStore.branches.filter(b => !q || b.name.toLowerCase().includes(q))
})
</script>

<template>
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
</template>

<style scoped>
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
</style>
