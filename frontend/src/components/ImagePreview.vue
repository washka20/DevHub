<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  src: string
  filename: string
}>()

const naturalWidth = ref(0)
const naturalHeight = ref(0)
const zoom = ref(1)

function onLoad(e: Event) {
  const img = e.target as HTMLImageElement
  naturalWidth.value = img.naturalWidth
  naturalHeight.value = img.naturalHeight
}

function handleWheel(e: WheelEvent) {
  e.preventDefault()
  const delta = e.deltaY > 0 ? -0.1 : 0.1
  zoom.value = Math.max(0.1, Math.min(5, zoom.value + delta))
}

function resetZoom() { zoom.value = 1 }
</script>

<template>
  <div class="image-preview" @wheel.prevent="handleWheel">
    <div class="preview-toolbar">
      <span class="preview-filename">{{ filename }}</span>
      <span v-if="naturalWidth" class="preview-size">{{ naturalWidth }} × {{ naturalHeight }}</span>
      <span class="preview-zoom">{{ Math.round(zoom * 100) }}%</span>
      <button class="preview-btn" @click="resetZoom" title="Reset zoom">1:1</button>
    </div>
    <div class="preview-body">
      <img
        :src="src"
        :style="{ transform: `scale(${zoom})` }"
        @load="onLoad"
        draggable="false"
      />
    </div>
  </div>
</template>

<style scoped>
.image-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary);
}
.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  flex-shrink: 0;
}
.preview-filename { color: var(--text-primary); font-weight: 500; }
.preview-zoom { color: var(--accent-blue); }
.preview-btn {
  padding: 2px 8px; border: 1px solid var(--border); border-radius: 4px;
  background: none; color: var(--text-secondary); cursor: pointer; font-size: 11px;
}
.preview-btn:hover { color: var(--text-primary); border-color: var(--text-secondary); }
.preview-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  padding: 24px;
  /* Checkerboard pattern for transparent PNGs */
  background-image: linear-gradient(45deg, var(--bg-tertiary) 25%, transparent 25%),
    linear-gradient(-45deg, var(--bg-tertiary) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, var(--bg-tertiary) 75%),
    linear-gradient(-45deg, transparent 75%, var(--bg-tertiary) 75%);
  background-size: 20px 20px;
  background-position: 0 0, 0 10px, 10px -10px, -10px 0;
}
.preview-body img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  transform-origin: center;
  transition: transform 0.1s ease;
  image-rendering: auto;
}
</style>
