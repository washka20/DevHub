<script setup lang="ts">
defineProps<{
  lines?: number
  variant?: 'card' | 'row' | 'text'
}>()
</script>

<template>
  <div class="shimmer-block" :class="`shimmer-${variant ?? 'text'}`">
    <template v-if="variant === 'card'">
      <div class="shimmer-line" style="width:50%"></div>
      <div class="shimmer-line shimmer-lg" style="width:30%"></div>
      <div class="shimmer-line" style="width:70%;margin-top:8px"></div>
    </template>
    <template v-else-if="variant === 'row'">
      <div v-for="i in (lines ?? 3)" :key="i" class="shimmer-row">
        <div class="shimmer-circle"></div>
        <div class="shimmer-row-lines">
          <div class="shimmer-line" :style="{ width: (60 + (i * 10) % 30) + '%' }"></div>
          <div class="shimmer-line" :style="{ width: (40 + (i * 15) % 30) + '%' }"></div>
        </div>
      </div>
    </template>
    <template v-else>
      <div
        v-for="i in (lines ?? 3)"
        :key="i"
        class="shimmer-line"
        :style="{ width: (50 + (i * 17) % 40) + '%' }"
      ></div>
    </template>
  </div>
</template>

<style scoped>
.shimmer-line {
  height: 12px;
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, #2d333b 50%, var(--bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: 4px;
  margin-bottom: 8px;
}

.shimmer-line:last-child {
  margin-bottom: 0;
}

.shimmer-lg {
  height: 22px;
}

.shimmer-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.shimmer-row:last-child {
  margin-bottom: 0;
}

.shimmer-circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(90deg, var(--bg-tertiary) 25%, #2d333b 50%, var(--bg-tertiary) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  flex-shrink: 0;
}

.shimmer-row-lines {
  flex: 1;
}

.shimmer-card {
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 10px;
}
</style>
