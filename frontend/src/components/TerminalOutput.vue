<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'

const props = defineProps<{
  lines: string[]
  running?: boolean
}>()

const container = ref<HTMLDivElement>()
const autoScroll = ref(true)

// Strip ANSI escape codes from a string
function stripAnsi(str: string): string {
  // Matches all ANSI escape sequences including color, cursor, erase codes
  return str.replace(
    // eslint-disable-next-line no-control-regex
    /\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?(?:\x07|\x1b\\)|\x1b[()][AB012]|\x1b[=>]|\x08/g,
    '',
  )
}

const cleanLines = computed(() => props.lines.map(stripAnsi))

// Auto-scroll to bottom on new lines
watch(
  () => props.lines.length,
  async () => {
    if (!autoScroll.value) return
    await nextTick()
    if (container.value) {
      container.value.scrollTop = container.value.scrollHeight
    }
  },
)

// Detect if user scrolled up manually (disable auto-scroll)
function onScroll() {
  if (!container.value) return
  const { scrollTop, scrollHeight, clientHeight } = container.value
  // If within 40px of the bottom, re-enable auto-scroll
  autoScroll.value = scrollHeight - scrollTop - clientHeight < 40
}
</script>

<template>
  <div class="terminal-wrapper">
    <div class="terminal" ref="container" @scroll="onScroll">
      <div v-if="cleanLines.length === 0 && !running" class="terminal-placeholder">
        Output will appear here...
      </div>
      <div v-for="(line, i) in cleanLines" :key="i" class="terminal-line">{{ line }}</div>
      <div v-if="running" class="terminal-line running">
        <span class="spinner">&#9654;</span> Running...
      </div>
    </div>
  </div>
</template>

<style scoped>
.terminal-wrapper {
  position: relative;
}

.terminal {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px 16px;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.6;
  min-height: 120px;
  max-height: 500px;
  overflow-y: auto;
  color: var(--text-primary);
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.3);
}

.terminal-placeholder {
  color: var(--text-secondary);
  font-style: italic;
}

.terminal-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.running {
  color: var(--accent-green);
}

.spinner {
  display: inline-block;
  animation: blink 1s steps(2) infinite;
}

@keyframes blink {
  0% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}
</style>
