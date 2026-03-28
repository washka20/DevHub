<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  subtext?: string
  color?: string
}>()
</script>

<template>
  <div class="status-card" :style="{ '--card-accent': color ?? 'var(--text-primary)' }">
    <div class="card-accent-line"></div>
    <div class="card-body">
      <div class="status-label">{{ label }}</div>
      <div class="status-value">{{ value }}</div>
      <div v-if="subtext" class="status-subtext">{{ subtext }}</div>
    </div>
  </div>
</template>

<style scoped>
.status-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 10px;
  display: flex;
  overflow: hidden;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.status-card:hover {
  border-color: var(--card-accent);
  box-shadow: 0 0 12px color-mix(in srgb, var(--card-accent) 20%, transparent);
}

.card-accent-line {
  width: 3px;
  background: var(--card-accent);
  flex-shrink: 0;
}

.card-body {
  padding: 16px 20px;
  flex: 1;
  min-width: 0;
}

.status-label {
  font-size: 11px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
}

.status-value {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--card-accent);
  font-family: var(--font-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-subtext {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}
</style>
