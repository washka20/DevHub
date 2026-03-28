<script setup lang="ts">
defineProps<{
  name: string
  description?: string
  category?: string
  loading?: boolean
  disabled?: boolean
}>()

defineEmits<{
  execute: []
}>()

const categoryColors: Record<string, string> = {
  Docker: 'var(--accent-green)',
  NPM: 'var(--accent-blue)',
  Composer: 'var(--accent-purple)',
  PHP: 'var(--accent-orange)',
  Git: '#79c0ff',
  Init: '#d29922',
}

function getColor(category?: string): string {
  return categoryColors[category ?? ''] ?? 'var(--accent-blue)'
}
</script>

<template>
  <button
    class="cmd-btn"
    :style="{ borderColor: getColor(category) }"
    :disabled="loading || disabled"
    @click="$emit('execute')"
  >
    <span class="cmd-name" :style="{ color: getColor(category) }">make {{ name }}</span>
    <span v-if="description" class="cmd-desc">{{ description }}</span>
    <span v-if="loading" class="cmd-loading">...</span>
  </button>
</template>

<style scoped>
.cmd-btn {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  text-align: left;
  transition: background var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast), transform var(--transition-fast);
  position: relative;
  overflow: hidden;
}

.cmd-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity var(--transition-fast);
  border-radius: inherit;
  background: radial-gradient(ellipse at 50% 100%, currentColor, transparent 70%);
  pointer-events: none;
}

.cmd-btn:hover:not(:disabled) {
  background: var(--bg-tertiary);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.cmd-btn:hover:not(:disabled)::before {
  opacity: 0.06;
}

.cmd-btn:active:not(:disabled) {
  transform: translateY(0);
}

.cmd-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.cmd-name {
  font-size: 13px;
  font-weight: 600;
  font-family: var(--font-mono);
}

.cmd-desc {
  font-size: 12px;
  color: var(--text-secondary);
}

.cmd-loading {
  font-size: 12px;
  color: var(--accent-orange);
}
</style>
