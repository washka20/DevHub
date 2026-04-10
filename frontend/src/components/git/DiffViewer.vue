<script setup lang="ts">
import { computed } from 'vue'
import { parseDiff } from '../../composables/useDiffParser'
import type { DiffLine } from '../../types'

const props = defineProps<{
  diff: string
  filename?: string
  loading?: boolean
}>()

const parsedLines = computed<DiffLine[]>(() => {
  if (!props.diff) return []
  return parseDiff(props.diff)
})
</script>

<template>
  <div class="diff-viewer">
    <div v-if="filename" class="diff-header">
      <span class="diff-filename">
        <slot name="header-prefix" />
        {{ filename }}
      </span>
      <slot name="header-actions" />
    </div>
    <div class="diff-content">
      <div v-if="loading" class="diff-loading">
        Loading diff...
      </div>
      <template v-else>
        <div
          v-for="(line, idx) in parsedLines"
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
  </div>
</template>

<style scoped>
.diff-viewer {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  flex: 1;
  min-height: 0;
}

.diff-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
</style>
