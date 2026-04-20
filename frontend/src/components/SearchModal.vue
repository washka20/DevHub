<script setup lang="ts">
import { ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectsStore } from '../stores/projects'
import { useFilesStore } from '../stores/files'
import { searchApi, type SearchResult } from '../api/search'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()
const projectsStore = useProjectsStore()
const filesStore = useFilesStore()

const query = ref('')
const glob = ref('')
const results = ref<SearchResult[]>([])
const loading = ref(false)
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const resultsRef = ref<HTMLDivElement | null>(null)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

function projectName(): string {
  return projectsStore.currentProject?.name ?? '_'
}

async function doSearch() {
  const q = query.value.trim()
  if (q.length < 2) {
    results.value = []
    return
  }

  loading.value = true
  try {
    results.value = await searchApi.search(projectName(), q, glob.value || undefined)
    selectedIndex.value = 0
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

function onInput() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(doSearch, 300)
}

function scrollToSelected() {
  nextTick(() => {
    const container = resultsRef.value
    if (!container) return
    const selected = container.querySelector('.search-result-item.selected')
    if (selected) {
      selected.scrollIntoView({ block: 'nearest' })
    }
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (selectedIndex.value < results.value.length - 1) {
      selectedIndex.value++
      scrollToSelected()
    }
    return
  }

  if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (selectedIndex.value > 0) {
      selectedIndex.value--
      scrollToSelected()
    }
    return
  }

  if (e.key === 'Enter' && results.value.length > 0) {
    e.preventDefault()
    openResult(results.value[selectedIndex.value])
    return
  }
}

function openResult(result: SearchResult) {
  emit('close')
  filesStore.openFile(result.file)
  router.push('/editor')
}

function highlightMatch(content: string, q: string): string {
  if (!q) return escapeHtml(content)
  const htmlContent = escapeHtml(content)
  const htmlQuery = escapeHtml(q).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return htmlContent.replace(
    new RegExp(`(${htmlQuery})`, 'gi'),
    '<mark class="search-highlight">$1</mark>',
  )
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

watch(() => props.visible, (v) => {
  if (v) {
    query.value = ''
    glob.value = ''
    results.value = []
    selectedIndex.value = 0
    nextTick(() => inputRef.value?.focus())
  }
})

onBeforeUnmount(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
})

defineExpose({ query, onInput })
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="modal-overlay" @click.self="emit('close')">
        <div class="modal-content" @keydown="handleKeydown">
          <div class="modal-header">
            <span class="modal-title">Search in Files</span>
            <span class="modal-esc" @click="emit('close')">ESC</span>
          </div>

          <div class="search-inputs">
            <input
              ref="inputRef"
              v-model="query"
              class="search-input"
              placeholder="Search text..."
              @input="onInput"
            />
            <input
              v-model="glob"
              class="search-input search-glob"
              placeholder="File filter (e.g. *.go, *.ts)"
              @input="onInput"
            />
          </div>

          <div class="search-status">
            <span v-if="loading" class="search-loading">Searching...</span>
            <span v-else-if="query.trim().length >= 2 && results.length === 0" class="search-empty">No results</span>
            <span v-else-if="results.length > 0" class="search-count">{{ results.length }} result{{ results.length === 1 ? '' : 's' }}</span>
          </div>

          <div ref="resultsRef" class="search-results">
            <div
              v-for="(r, i) in results"
              :key="`${r.file}:${r.line}:${i}`"
              class="search-result-item"
              :class="{ selected: i === selectedIndex }"
              @click="openResult(r)"
              @mouseenter="selectedIndex = i"
            >
              <div class="result-file">
                <span class="result-path">{{ r.file }}</span>
                <span class="result-line">:{{ r.line }}</span>
              </div>
              <div class="result-content" v-html="highlightMatch(r.content, query.trim())"></div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--overlay-soft);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 10vh;
  z-index: 300;
}

.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  width: 600px;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.modal-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-esc {
  font-size: 11px;
  color: var(--text-secondary);
  border: 1px solid var(--border);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: var(--font-mono);
  cursor: pointer;
  transition: color var(--transition-fast), border-color var(--transition-fast);
}

.modal-esc:hover {
  color: var(--text-primary);
  border-color: var(--text-secondary);
}

.search-inputs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 8px;
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color var(--transition-fast);
}

.search-input:focus {
  border-color: var(--accent-blue);
}

.search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.6;
}

.search-glob {
  font-size: 12px;
  padding: 6px 12px;
}

.search-status {
  font-size: 11px;
  color: var(--text-secondary);
  padding: 4px 0;
  min-height: 20px;
}

.search-loading {
  opacity: 0.7;
}

.search-results {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.search-result-item {
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background var(--transition-fast);
}

.search-result-item:hover,
.search-result-item.selected {
  background: var(--bg-tertiary);
}

.result-file {
  font-size: 12px;
  margin-bottom: 2px;
}

.result-path {
  color: var(--accent-blue);
  font-family: var(--font-mono);
}

.result-line {
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.result-content {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.search-highlight) {
  background: rgba(255, 213, 79, 0.3);
  color: var(--text-primary);
  border-radius: 2px;
  padding: 0 1px;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
