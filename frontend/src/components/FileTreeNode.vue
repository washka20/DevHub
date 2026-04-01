<script setup lang="ts">
import { defineProps, defineEmits } from 'vue'

export interface FileNode {
  name: string
  path: string
  isDir: boolean
  children: FileNode[]
}

defineProps<{
  nodes: FileNode[]
  currentFile: string
  collapsed: Record<string, boolean>
}>()

const emit = defineEmits<{
  select: [path: string]
  toggle: [path: string]
}>()
</script>

<template>
  <template v-for="node in nodes" :key="node.path">
    <template v-if="node.isDir">
      <button class="tree-dir" @click="emit('toggle', node.path)">
        <svg class="tree-chevron" :class="{ collapsed: collapsed[node.path] }" width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
          <path d="M12.78 5.22a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L3.22 6.28a.75.75 0 0 1 1.06-1.06L8 8.94l3.72-3.72a.75.75 0 0 1 1.06 0z"/>
        </svg>
        <svg class="tree-folder-icon" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
          <path d="M1.75 1A1.75 1.75 0 0 0 0 2.75v10.5C0 14.216.784 15 1.75 15h12.5A1.75 1.75 0 0 0 16 13.25v-8.5A1.75 1.75 0 0 0 14.25 3H7.5a.25.25 0 0 1-.2-.1l-.9-1.2C6.07 1.26 5.55 1 5 1H1.75z"/>
        </svg>
        <span class="tree-name">{{ node.name }}</span>
      </button>
      <div v-show="!collapsed[node.path]" class="tree-children">
        <FileTreeNode
          :nodes="node.children"
          :current-file="currentFile"
          :collapsed="collapsed"
          @select="emit('select', $event)"
          @toggle="emit('toggle', $event)"
        />
      </div>
    </template>
    <button
      v-else
      :class="['tree-file', { active: node.path === currentFile }]"
      @click="emit('select', node.path)"
      :title="node.path"
    >
      <svg class="tree-file-icon" width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
        <path d="M0 1.75A.75.75 0 0 1 .75 1h4.253c1.227 0 2.317.59 3 1.501A3.743 3.743 0 0 1 11.006 1h4.245a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75h-4.507a2.25 2.25 0 0 0-1.591.659l-.622.621a.75.75 0 0 1-1.06 0l-.622-.621A2.25 2.25 0 0 0 5.258 13H.75a.75.75 0 0 1-.75-.75zm7.251 10.324l.004-5.073-.002-2.253A2.25 2.25 0 0 0 5.003 2.5H1.5v9h3.757a3.75 3.75 0 0 1 1.994.574zM8.755 4.75l-.004 7.322a3.752 3.752 0 0 1 1.992-.572H14.5v-9h-3.495a2.25 2.25 0 0 0-2.25 2.25z"/>
      </svg>
      <span class="tree-name">{{ node.name }}</span>
    </button>
  </template>
</template>
