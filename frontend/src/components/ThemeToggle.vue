<script setup lang="ts">
import { useTheme } from '../composables/useTheme'

const { theme, setTheme } = useTheme()
</script>

<template>
  <div class="theme-toggle" role="group" aria-label="Theme">
    <button
      type="button"
      :class="{ on: theme === 'light' }"
      aria-label="Light theme"
      @click="setTheme('light')"
    >
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="8" cy="8" r="3"/>
        <path d="M8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.5 1.5M11.5 11.5L13 13M3 13l1.5-1.5M11.5 4.5L13 3"/>
      </svg>
    </button>
    <button
      type="button"
      :class="{ on: theme === 'dark' }"
      aria-label="Dark theme"
      @click="setTheme('dark')"
    >
      <svg viewBox="0 0 16 16" fill="currentColor">
        <path d="M13 9.5A5.5 5.5 0 017.5 4a5.5 5.5 0 01.3-1.8A6 6 0 1013.8 9.2a5.5 5.5 0 01-.8.3z"/>
      </svg>
    </button>
  </div>
</template>
