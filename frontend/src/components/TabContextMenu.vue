<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  x: number
  y: number
  tabId: string
  canSplit: boolean
}>()

const emit = defineEmits<{
  close: []
  rename: [tabId: string]
  splitH: [tabId: string]
  splitV: [tabId: string]
  closeTab: [tabId: string]
  closeOthers: [tabId: string]
  closeAll: []
}>()

const menuEl = ref<HTMLDivElement>()

function handleClickOutside(e: MouseEvent) {
  if (menuEl.value && !menuEl.value.contains(e.target as Node)) {
    emit('close')
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => {
  document.addEventListener('mousedown', handleClickOutside)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div ref="menuEl" class="context-menu" :style="{ left: x + 'px', top: y + 'px' }">
    <div class="menu-item" @click="emit('rename', tabId); emit('close')">
      <span>Rename</span>
      <span class="hint">F2</span>
    </div>
    <div class="menu-sep"></div>
    <div class="menu-item" :class="{ disabled: !canSplit }" @click="canSplit && (emit('splitH', tabId), emit('close'))">
      Split Horizontal
    </div>
    <div class="menu-item" :class="{ disabled: !canSplit }" @click="canSplit && (emit('splitV', tabId), emit('close'))">
      Split Vertical
    </div>
    <div class="menu-sep"></div>
    <div class="menu-item" @click="emit('closeTab', tabId); emit('close')">Close</div>
    <div class="menu-item" @click="emit('closeOthers', tabId); emit('close')">Close Others</div>
    <div class="menu-item danger" @click="emit('closeAll'); emit('close')">Close All</div>
  </div>
</template>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 1000;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 4px 0;
  min-width: 180px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  font-family: var(--font-ui);
  font-size: 13px;
}

.menu-item {
  padding: 6px 12px;
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.menu-item:hover {
  background: var(--accent-blue);
  color: #fff;
}

.menu-item.danger {
  color: var(--accent-red);
}

.menu-item.danger:hover {
  background: var(--accent-red);
  color: #fff;
}

.menu-item.disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.menu-item.disabled:hover {
  background: none;
  color: var(--text-primary);
}

.hint {
  font-size: 11px;
  color: var(--text-secondary);
}

.menu-item:hover .hint {
  color: rgba(255, 255, 255, 0.7);
}

.menu-sep {
  height: 1px;
  background: var(--border);
  margin: 4px 0;
}
</style>
