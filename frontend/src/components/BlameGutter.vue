<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import type { BlameEntry } from '../types'

const props = defineProps<{
  entries: BlameEntry[]
  editorScrollDom: HTMLElement | null
  lineHeight: number
}>()

const scrollTop = ref(0)
const gutterEl = ref<HTMLDivElement>()
const hoveredHash = ref<string | null>(null)
const tooltipEntry = ref<BlameEntry | null>(null)
const tooltipPos = ref({ top: 0, left: 0 })

const totalLines = computed(() => {
  if (props.entries.length === 0) return 0
  return props.entries[props.entries.length - 1].line_end
})

const lineMap = computed(() => {
  const map: Array<{ entry: BlameEntry; isFirst: boolean }> = []
  for (const entry of props.entries) {
    for (let line = entry.line_start; line <= entry.line_end; line++) {
      map[line] = { entry, isFirst: line === entry.line_start }
    }
  }
  return map
})

function onScroll() {
  if (props.editorScrollDom) {
    scrollTop.value = props.editorScrollDom.scrollTop
  }
}

function showTooltip(entry: BlameEntry, event: MouseEvent) {
  tooltipEntry.value = entry
  tooltipPos.value = { top: event.clientY - 10, left: event.clientX + 12 }
}

function hideTooltip() {
  tooltipEntry.value = null
}

function formatDate(date: string): string {
  return date
}

function shortAuthor(author: string): string {
  const parts = author.split(' ')
  if (parts.length >= 2) {
    return parts[0][0] + '. ' + parts.slice(1).join(' ')
  }
  return author.length > 12 ? author.slice(0, 11) + '\u2026' : author
}

function handleLineEnter(line: number, event: MouseEvent) {
  const info = lineMap.value[line]
  hoveredHash.value = info?.entry.hash ?? null
  if (info) showTooltip(info.entry, event)
}

function handleLineMove(line: number, event: MouseEvent) {
  const info = lineMap.value[line]
  if (info) showTooltip(info.entry, event)
}

function handleLineLeave() {
  hoveredHash.value = null
  hideTooltip()
}

watch(() => props.editorScrollDom, (dom, oldDom) => {
  if (oldDom) oldDom.removeEventListener('scroll', onScroll)
  if (dom) {
    dom.addEventListener('scroll', onScroll, { passive: true })
    scrollTop.value = dom.scrollTop
  }
})

onMounted(() => {
  if (props.editorScrollDom) {
    props.editorScrollDom.addEventListener('scroll', onScroll, { passive: true })
    scrollTop.value = props.editorScrollDom.scrollTop
  }
})

onBeforeUnmount(() => {
  if (props.editorScrollDom) {
    props.editorScrollDom.removeEventListener('scroll', onScroll)
  }
})
</script>

<template>
  <div ref="gutterEl" class="blame-gutter" @mouseleave="hideTooltip">
    <div class="blame-scroll" :style="{ transform: `translateY(-${scrollTop}px)` }">
      <div
        v-for="line in totalLines"
        :key="line"
        class="blame-line"
        :class="{
          'blame-first': lineMap[line]?.isFirst,
          'blame-hovered': lineMap[line]?.entry.hash === hoveredHash,
        }"
        :style="{ height: lineHeight + 'px' }"
        @mouseenter="handleLineEnter(line, $event)"
        @mousemove="handleLineMove(line, $event)"
        @mouseleave="handleLineLeave()"
      >
        <template v-if="lineMap[line]?.isFirst">
          <span class="blame-author">{{ shortAuthor(lineMap[line].entry.author) }}</span>
          <span class="blame-date">{{ formatDate(lineMap[line].entry.date) }}</span>
          <span class="blame-hash">{{ lineMap[line].entry.short_hash }}</span>
        </template>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="tooltipEntry"
        class="blame-tooltip"
        :style="{ top: tooltipPos.top + 'px', left: tooltipPos.left + 'px' }"
      >
        <div class="tooltip-hash">{{ tooltipEntry.short_hash }}</div>
        <div class="tooltip-message">{{ tooltipEntry.message }}</div>
        <div class="tooltip-meta">{{ tooltipEntry.author }} &middot; {{ tooltipEntry.date }}</div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.blame-gutter {
  width: 220px;
  flex-shrink: 0;
  overflow: hidden;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1;
  user-select: none;
  position: relative;
}

.blame-scroll {
  will-change: transform;
}

.blame-line {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  border-top: 1px solid transparent;
  transition: background 0.1s;
}

.blame-line.blame-first {
  border-top-color: var(--border);
}

.blame-line.blame-first:first-child {
  border-top-color: transparent;
}

.blame-line.blame-hovered {
  background: var(--accent-2);
}

.blame-author {
  flex-shrink: 0;
  width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
  opacity: 0.8;
}

.blame-date {
  flex-shrink: 0;
  width: 78px;
  color: var(--text-secondary);
  opacity: 0.6;
}

.blame-hash {
  color: var(--accent-blue);
  opacity: 0.5;
  font-size: 11px;
}

.blame-tooltip {
  position: fixed;
  z-index: 10000;
  padding: 8px 12px;
  background: var(--bg-tertiary, #1c2128);
  border: 1px solid var(--border);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  max-width: 400px;
  pointer-events: none;
  font-family: var(--font-mono);
}

.tooltip-hash {
  font-size: 11px;
  color: var(--accent-blue);
  margin-bottom: 4px;
}

.tooltip-message {
  font-size: 12px;
  color: var(--text-primary);
  margin-bottom: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}

.tooltip-meta {
  font-size: 11px;
  color: var(--text-secondary);
}
</style>
