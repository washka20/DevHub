<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  useCommandPalette,
  fuzzyFilter,
  loadRecentIds,
  recordRecent,
  type CommandItem,
} from '../composables/useCommandPalette'
import { useGitStore } from '../stores/git'
import { useDockerStore } from '../stores/docker'
import { useTheme } from '../composables/useTheme'
import { useToast } from '../composables/useToast'

const { open, closeCommandPalette } = useCommandPalette()
const router = useRouter()
const gitStore = useGitStore()
const dockerStore = useDockerStore()
const { setTheme } = useTheme()
const toast = useToast()

const query = ref('')
const selectedIdx = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLDivElement | null>(null)

const isMac = typeof navigator !== 'undefined' && /Mac/i.test(navigator.platform || '')
const metaKey = isMac ? '⌘' : 'Ctrl+'

function navItem(label: string, path: string, shortcut?: string): CommandItem {
  return {
    id: `nav:${path}`,
    label,
    group: 'Navigate',
    keywords: `go open ${label} ${path}`,
    shortcut,
    iconHtml: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 8h10M9 4l4 4-4 4"/></svg>',
    run: () => { router.push(path) },
  }
}

function gitAction(id: string, label: string, keywords: string, run: () => Promise<void> | void): CommandItem {
  return {
    id: `git:${id}`,
    label,
    group: 'Git',
    keywords,
    iconHtml: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="4" cy="4" r="2"/><circle cx="4" cy="12" r="2"/><circle cx="12" cy="8" r="2"/><path d="M4 6v4M6 4h2a2 2 0 012 2v0M6 12h2a2 2 0 002-2v0"/></svg>',
    run,
  }
}

function dockerAction(id: string, label: string, keywords: string, run: () => Promise<void> | void): CommandItem {
  return {
    id: `docker:${id}`,
    label,
    group: 'Docker',
    keywords,
    iconHtml: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 7h2v2H2zM5 7h2v2H5zM8 7h2v2H8zM5 4h2v2H5zM8 4h2v2H8zM8 1h2v2H8z"/><path d="M1 9c0 3 2 5 6 5 5 0 8-3 8-6h-1a2 2 0 00-2 1"/></svg>',
    run,
  }
}

const commands = computed<CommandItem[]>(() => [
  navItem('Dashboard', '/', `${metaKey}1`),
  navItem('Git', '/git', `${metaKey}2`),
  navItem('Commands', '/commands', `${metaKey}3`),
  navItem('Docker', '/docker', `${metaKey}4`),
  navItem('GitLab', '/gitlab'),
  navItem('Console', '/console'),
  navItem('Editor', '/editor'),
  navItem('Notes', '/notes'),
  navItem('README', '/readme'),
  navItem('Settings', '/settings'),

  gitAction('pull', 'Pull', 'fetch merge pull', async () => {
    try { await gitStore.pull(); toast.show('success', 'Pulled') } catch (e) { toast.show('error', (e as Error).message) }
  }),
  gitAction('push', 'Push', 'upload push', async () => {
    try { await gitStore.push(); toast.show('success', 'Pushed') } catch (e) { toast.show('error', (e as Error).message) }
  }),
  gitAction('fetch-status', 'Refresh status', 'refresh reload git status', () => { gitStore.fetchStatus() }),
  gitAction('fetch-log', 'Refresh commit log', 'refresh reload log commits', () => { gitStore.fetchLog() }),

  dockerAction('reload', 'Refresh containers', 'refresh reload containers', () => { dockerStore.fetchContainers() }),

  {
    id: 'theme:dark',
    label: 'Use dark theme',
    group: 'Settings',
    keywords: 'theme appearance dark night',
    iconHtml: '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M13 9.5A5.5 5.5 0 017.5 4a5.5 5.5 0 01.3-1.8A6 6 0 1013.8 9.2a5.5 5.5 0 01-.8.3z"/></svg>',
    run: () => { setTheme('dark') },
  },
  {
    id: 'theme:light',
    label: 'Use light theme',
    group: 'Settings',
    keywords: 'theme appearance light day paper',
    iconHtml: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="3"/><path d="M8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.5 1.5M11.5 11.5L13 13M3 13l1.5-1.5M11.5 4.5L13 3"/></svg>',
    run: () => { setTheme('light') },
  },
])

const recentIds = ref<string[]>(loadRecentIds())

const filtered = computed<CommandItem[]>(() => {
  const q = query.value.trim()
  if (!q) {
    const byId = new Map(commands.value.map((c) => [c.id, c]))
    const recent = recentIds.value.map((id) => byId.get(id)).filter(Boolean) as CommandItem[]
    const rest = commands.value.filter((c) => !recentIds.value.includes(c.id))
    return [...recent.map((c) => ({ ...c, group: 'Recent' })), ...rest]
  }
  return fuzzyFilter(commands.value, q)
})

const grouped = computed(() => {
  const groups = new Map<string, CommandItem[]>()
  for (const it of filtered.value) {
    const arr = groups.get(it.group) ?? []
    arr.push(it)
    groups.set(it.group, arr)
  }
  return Array.from(groups.entries()).map(([name, items]) => ({ name, items }))
})

const flatItems = computed(() => filtered.value)

watch(filtered, () => { selectedIdx.value = 0 })

function moveSelection(delta: number) {
  const max = flatItems.value.length
  if (!max) return
  selectedIdx.value = (selectedIdx.value + delta + max) % max
  nextTick(() => {
    const el = listRef.value?.querySelector<HTMLElement>(`[data-idx="${selectedIdx.value}"]`)
    el?.scrollIntoView({ block: 'nearest' })
  })
}

async function runAt(idx: number) {
  const item = flatItems.value[idx]
  if (!item) return
  recordRecent(item.id)
  recentIds.value = loadRecentIds()
  closeCommandPalette()
  try { await item.run() } catch (e) { toast.show('error', (e as Error)?.message || 'Failed') }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') { e.preventDefault(); closeCommandPalette(); return }
  if (e.key === 'ArrowDown') { e.preventDefault(); moveSelection(1); return }
  if (e.key === 'ArrowUp')   { e.preventDefault(); moveSelection(-1); return }
  if (e.key === 'Enter') { e.preventDefault(); runAt(selectedIdx.value); return }
}

function itemIdx(group: string, idxInGroup: number): number {
  let start = 0
  for (const g of grouped.value) {
    if (g.name === group) return start + idxInGroup
    start += g.items.length
  }
  return start + idxInGroup
}

watch(open, async (v) => {
  if (v) {
    query.value = ''
    selectedIdx.value = 0
    recentIds.value = loadRecentIds()
    await nextTick()
    inputRef.value?.focus()
  }
})

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="open" class="palette-backdrop" @mousedown.self="closeCommandPalette">
        <div class="palette" role="dialog" aria-label="Command palette">
          <div class="palette-head">
            <svg class="icon" width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="7" cy="7" r="5"/><path d="M14 14l-3.2-3.2"/>
            </svg>
            <input
              ref="inputRef"
              v-model="query"
              placeholder="Type a command or search…"
              spellcheck="false"
              autocomplete="off"
            />
            <span class="kbd">esc</span>
          </div>
          <div ref="listRef" class="palette-body">
            <template v-if="!flatItems.length">
              <div class="empty">
                <div class="glyph">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" width="22" height="22">
                    <circle cx="7" cy="7" r="5"/><path d="M14 14l-3.2-3.2"/>
                  </svg>
                </div>
                <h4>No matches</h4>
                <p>Try <span class="kbd">git pull</span>, <span class="kbd">open settings</span>, or <span class="kbd">dark</span>.</p>
              </div>
            </template>
            <template v-else>
              <div v-for="g in grouped" :key="g.name" class="palette-group">
                <div class="palette-group-label">{{ g.name }}</div>
                <button
                  v-for="(it, i) in g.items"
                  :key="it.id"
                  type="button"
                  class="palette-item"
                  :class="{ active: itemIdx(g.name, i) === selectedIdx }"
                  :data-idx="itemIdx(g.name, i)"
                  @mouseenter="selectedIdx = itemIdx(g.name, i)"
                  @click="runAt(itemIdx(g.name, i))"
                >
                  <span class="palette-item-icon" v-html="it.iconHtml || ''"></span>
                  <span class="palette-item-label">{{ it.label }}</span>
                  <span v-if="it.hint" class="palette-item-hint">{{ it.hint }}</span>
                  <span v-if="it.shortcut" class="kbd palette-item-kbd">{{ it.shortcut }}</span>
                </button>
              </div>
            </template>
          </div>
          <div class="palette-foot">
            <span><span class="kbd">↑↓</span> navigate</span>
            <span><span class="kbd">↵</span> run</span>
            <span><span class="kbd">esc</span> close</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 900;
  background: color-mix(in oklab, black 55%, transparent);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 14vh 16px 0;
  backdrop-filter: blur(2px);
}
[data-theme="light"] .palette-backdrop {
  background: color-mix(in oklab, black 28%, transparent);
}

.palette {
  width: 100%;
  max-width: 640px;
  max-height: 70vh;
  background: var(--bg-1);
  border: 1px solid var(--line);
  border-radius: var(--r3);
  box-shadow: var(--shadow-3);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.palette-head {
  display: flex;
  align-items: center;
  gap: var(--s3);
  padding: 14px 16px;
  border-bottom: 1px solid var(--line-soft);
}
.palette-head .icon { color: var(--fg-3); }
.palette-head input {
  flex: 1;
  border: 0;
  background: transparent;
  outline: 0;
  color: var(--fg);
  font-family: var(--ui);
  font-size: 14px;
  padding: 0;
}
.palette-head input::placeholder { color: var(--fg-3); }

.palette-body {
  overflow: auto;
  padding: 6px;
}

.palette-group + .palette-group { margin-top: 4px; }
.palette-group-label {
  padding: 6px 12px 4px;
  font-size: 10.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--fg-3);
  font-weight: 600;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 8px 12px;
  border: 0;
  background: transparent;
  color: var(--fg-2);
  font-family: var(--ui);
  font-size: 13.5px;
  cursor: pointer;
  border-radius: var(--r1);
  text-align: left;
}
.palette-item:hover,
.palette-item.active {
  background: var(--accent-2);
  color: var(--fg);
}
.palette-item-icon {
  width: 18px; height: 18px;
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--fg-3);
  flex-shrink: 0;
}
.palette-item.active .palette-item-icon { color: var(--accent); }
.palette-item-icon :deep(svg) { width: 14px; height: 14px; }
.palette-item-label { flex: 1; font-weight: 500; }
.palette-item-hint { font-size: 12px; color: var(--fg-3); }
.palette-item-kbd { margin-left: auto; }

.palette-foot {
  display: flex;
  gap: 16px;
  padding: 10px 14px;
  border-top: 1px solid var(--line-soft);
  font-size: 11px;
  color: var(--fg-3);
  font-family: var(--mono);
  background: var(--bg-2);
}

.empty {
  padding: 36px 24px;
  text-align: center;
  color: var(--fg-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}
.empty .glyph {
  width: 48px; height: 48px;
  border-radius: 14px;
  background: var(--bg-2);
  border: 1px solid var(--line);
  display: flex; align-items: center; justify-content: center;
}
.empty h4 { margin: 0; color: var(--fg); font-size: 14px; font-weight: 600; }
.empty p { margin: 0; font-size: 12.5px; }

.palette-enter-active, .palette-leave-active { transition: opacity 0.15s ease; }
.palette-enter-from, .palette-leave-to { opacity: 0; }
.palette-enter-active .palette, .palette-leave-active .palette {
  transition: transform 0.18s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.18s ease;
}
.palette-enter-from .palette { transform: translateY(8px) scale(.98); opacity: 0; }
.palette-leave-to .palette   { transform: translateY(4px) scale(.98); opacity: 0; }
</style>
