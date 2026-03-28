<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { useProject } from '../composables/useProject'

const store = useProjectsStore()
const { switchProject } = useProject()

const open = ref(false)
const search = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const selectorRef = ref<HTMLDivElement | null>(null)

const filteredProjects = computed(() => {
  const q = search.value.toLowerCase().trim()
  if (!q) return store.projects
  return store.projects.filter(
    (p) => p.name.toLowerCase().includes(q) || p.path.toLowerCase().includes(q)
  )
})

async function toggle() {
  open.value = !open.value
  if (open.value) {
    search.value = ''
    await nextTick()
    searchInput.value?.focus()
  }
}

async function select(name: string) {
  open.value = false
  search.value = ''
  await switchProject(name)
}

function handleClickOutside(e: MouseEvent) {
  if (selectorRef.value && !selectorRef.value.contains(e.target as Node)) {
    open.value = false
    search.value = ''
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="selectorRef" class="project-selector">
    <button class="selector-btn" @click="toggle">
      <span v-if="store.currentProject" class="selector-info">
        <span class="selector-name">{{ store.currentProject.name }}</span>
        <span class="selector-badges">
          <span v-if="store.currentProject.is_git" class="feature-badge" title="Git">G</span>
          <span v-if="store.currentProject.has_makefile" class="feature-badge" title="Makefile">M</span>
          <span v-if="store.currentProject.has_docker" class="feature-badge" title="Docker">D</span>
        </span>
      </span>
      <span v-else class="selector-name">Select project</span>
      <span class="selector-arrow">{{ open ? '\u25B2' : '\u25BC' }}</span>
    </button>
    <div v-if="open" class="selector-dropdown">
      <div class="search-wrap">
        <input
          ref="searchInput"
          v-model="search"
          type="text"
          class="search-input"
          placeholder="Search..."
        />
      </div>
      <div class="dropdown-list">
        <button
          v-for="project in filteredProjects"
          :key="project.name"
          class="dropdown-item"
          :class="{ active: store.currentProject?.name === project.name }"
          @click="select(project.name)"
        >
          <span class="item-row">
            <span class="item-name">{{ project.name }}</span>
            <span class="item-badges">
              <span v-if="project.is_git" class="feature-badge-sm" title="Git">G</span>
              <span v-if="project.has_makefile" class="feature-badge-sm" title="Makefile">M</span>
              <span v-if="project.has_docker" class="feature-badge-sm" title="Docker">D</span>
            </span>
          </span>
          <span class="item-path">{{ project.path }}</span>
        </button>
        <div v-if="filteredProjects.length === 0" class="no-results">
          No projects found
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.project-selector {
  position: relative;
  padding: 0 12px;
  margin: 8px 0;
}

.selector-btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 10px;
  color: var(--text-primary);
  font-size: 13px;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.selector-btn:hover {
  border-color: var(--accent-blue);
  box-shadow: 0 0 8px rgba(88, 166, 255, 0.12);
}

.selector-info {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  flex: 1;
  min-width: 0;
}

.selector-name {
  font-weight: 700;
  font-family: var(--font-mono);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
}

.selector-badges {
  display: flex;
  gap: 3px;
  flex-shrink: 0;
}

.feature-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 4px;
  font-size: 9px;
  font-weight: 700;
  font-family: var(--font-mono);
}

.feature-badge[title="Git"] {
  background: rgba(248, 129, 102, 0.2);
  color: #f78166;
}

.feature-badge[title="Makefile"] {
  background: rgba(63, 185, 80, 0.2);
  color: var(--accent-green);
}

.feature-badge[title="Docker"] {
  background: rgba(88, 166, 255, 0.2);
  color: var(--accent-blue);
}

.selector-arrow {
  font-size: 10px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.selector-dropdown {
  position: absolute;
  top: 100%;
  left: 12px;
  right: 12px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 6px;
  margin-top: 4px;
  z-index: 100;
  max-height: 300px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.search-wrap {
  padding: 8px;
  border-bottom: 1px solid var(--border);
}

.search-input {
  width: 100%;
  padding: 6px 10px;
  font-size: 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
}

.search-input:focus {
  border-color: var(--accent-blue);
  outline: none;
}

.dropdown-list {
  overflow-y: auto;
  max-height: 240px;
}

.dropdown-item {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: 8px 10px;
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 13px;
  text-align: left;
}

.dropdown-item:hover {
  background: var(--bg-secondary);
}

.dropdown-item.active {
  background: rgba(88, 166, 255, 0.1);
}

.item-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.item-name {
  font-weight: 500;
}

.item-badges {
  display: flex;
  gap: 2px;
  margin-left: auto;
}

.feature-badge-sm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  border-radius: 2px;
  font-size: 8px;
  font-weight: 700;
  background: var(--bg-primary);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.item-path {
  font-size: 11px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.no-results {
  padding: 16px;
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
}
</style>
