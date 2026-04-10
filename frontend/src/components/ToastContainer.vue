<script setup lang="ts">
import { useToast } from '../composables/useToast'

const { toasts, remove } = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="toast-container">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="toast"
          :class="`toast-${toast.type}`"
          @click="remove(toast.id)"
        >
          <span class="toast-icon">
            <template v-if="toast.type === 'success'">&#10003;</template>
            <template v-else-if="toast.type === 'error'">&#10007;</template>
            <template v-else>&#8505;</template>
          </span>
          <span class="toast-msg">{{ toast.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  position: fixed;
  bottom: 16px;
  right: 16px;
  display: flex;
  flex-direction: column-reverse;
  gap: 8px;
  z-index: 200;
  pointer-events: none;
}

.toast {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  pointer-events: auto;
  cursor: pointer;
  white-space: nowrap;
}

.toast-success {
  border-left: 3px solid var(--accent-green);
}
.toast-success .toast-icon {
  color: var(--accent-green);
}

.toast-info {
  border-left: 3px solid var(--accent-blue);
}
.toast-info .toast-icon {
  color: var(--accent-blue);
}

.toast-error {
  border-left: 3px solid var(--accent-red);
}
.toast-error .toast-icon {
  color: var(--accent-red);
}

.toast-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.toast-msg {
  color: var(--text-primary);
}
</style>
