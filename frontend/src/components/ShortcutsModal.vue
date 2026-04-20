<script setup lang="ts">
import { watch, toRef, onBeforeUnmount } from 'vue'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const shortcuts = [
  { action: 'Dashboard', key: 'Alt+1' },
  { action: 'Git', key: 'Alt+2' },
  { action: 'Commands', key: 'Alt+3' },
  { action: 'Docker', key: 'Alt+4' },
  { action: 'Console', key: 'Alt+5' },
  { action: 'README', key: 'Alt+6' },
  { action: 'Notes', key: 'Alt+7' },
  { action: 'Editor', key: 'Alt+8' },
  { action: 'Toggle Terminal', key: 'Ctrl+`' },
  { action: 'Save File', key: 'Ctrl+S' },
  { action: 'Go to Git', key: 'Ctrl+Shift+G' },
  { action: 'Show Shortcuts', key: '?' },
  { action: 'New Tab', key: 'Ctrl+Shift+T' },
  { action: 'Close Tab', key: 'Ctrl+Shift+W' },
  { action: 'Next Tab', key: 'Ctrl+PgDn' },
  { action: 'Prev Tab', key: 'Ctrl+PgUp' },
  { action: 'Split / Unsplit', key: 'Ctrl+Shift+D' },
  { action: 'File Search', key: 'Ctrl+Shift+F' },
]

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    document.addEventListener('keydown', handleKeydown)
  } else {
    document.removeEventListener('keydown', handleKeydown)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="modal-overlay" @click.self="emit('close')">
        <div class="modal-content">
          <div class="modal-header">
            <span class="modal-title">Keyboard Shortcuts</span>
            <span class="modal-esc" @click="emit('close')">ESC</span>
          </div>
          <div class="shortcuts-grid">
            <div
              v-for="s in shortcuts"
              :key="s.key"
              class="shortcut-row"
            >
              <span class="shortcut-action">{{ s.action }}</span>
              <kbd class="shortcut-key">{{ s.key }}</kbd>
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
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  width: 480px;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
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

.shortcuts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 24px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px solid var(--bg-tertiary);
}

.shortcut-action {
  font-size: 12px;
  color: var(--text-secondary);
}

.shortcut-key {
  font-size: 11px;
  color: var(--text-primary);
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: var(--font-mono);
  border: 1px solid var(--border);
}
</style>
