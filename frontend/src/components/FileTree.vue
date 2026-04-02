<script setup lang="ts">
import { ref, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useFilesStore } from '../stores/files'
import { useProjectsStore } from '../stores/projects'
import { getFileIcon } from './FileIcons'
import type { FileNode } from '../types'

const filesStore = useFilesStore()
const projectsStore = useProjectsStore()

function getFullPath(node: FileNode): string {
  const basePath = projectsStore.currentProject?.path || ''
  return `${basePath}/${node.path}`
}

function copyPath(node: FileNode) {
  navigator.clipboard.writeText(getFullPath(node))
  closeContextMenu()
}

function openInFileManager(node: FileNode) {
  closeContextMenu()
  filesStore.openInFileManager(node.path)
}

const expandedDirs = ref<Set<string>>(new Set())
const contextMenu = ref<{ x: number; y: number; node: FileNode } | null>(null)
const renaming = ref<{ path: string; value: string } | null>(null)
const creating = ref<{ parentPath: string; isDir: boolean; value: string } | null>(null)

const renameInput = ref<HTMLInputElement | null>(null)
const createInput = ref<HTMLInputElement | null>(null)
const contextMenuEl = ref<HTMLDivElement | null>(null)

function toggleDir(path: string) {
  const dirs = new Set(expandedDirs.value)
  if (dirs.has(path)) {
    dirs.delete(path)
  } else {
    dirs.add(path)
  }
  expandedDirs.value = dirs
}

function handleClick(node: FileNode) {
  if (node.is_dir) {
    toggleDir(node.path)
  } else {
    filesStore.openFile(node.path)
  }
}

function handleContextMenu(e: MouseEvent, node: FileNode) {
  e.preventDefault()
  contextMenu.value = { x: e.clientX, y: e.clientY, node }
}

function closeContextMenu() {
  contextMenu.value = null
}

function startRename(node: FileNode) {
  renaming.value = { path: node.path, value: node.name }
  closeContextMenu()
  nextTick(() => {
    renameInput.value?.focus()
    renameInput.value?.select()
  })
}

async function finishRename() {
  if (!renaming.value) return
  const oldPath = renaming.value.path
  const dir = oldPath.substring(0, oldPath.lastIndexOf('/'))
  const newPath = dir ? `${dir}/${renaming.value.value}` : renaming.value.value
  if (newPath !== oldPath && renaming.value.value.trim()) {
    await filesStore.renameFile(oldPath, newPath)
  }
  renaming.value = null
}

function handleRenameKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    finishRename()
  } else if (e.key === 'Escape') {
    renaming.value = null
  }
}

function startCreate(parentPath: string, isDir: boolean) {
  creating.value = { parentPath, isDir, value: '' }
  closeContextMenu()
  if (!expandedDirs.value.has(parentPath)) {
    const dirs = new Set(expandedDirs.value)
    dirs.add(parentPath)
    expandedDirs.value = dirs
  }
  nextTick(() => {
    createInput.value?.focus()
  })
}

async function finishCreate() {
  if (!creating.value || !creating.value.value.trim()) {
    creating.value = null
    return
  }
  const path = creating.value.parentPath
    ? `${creating.value.parentPath}/${creating.value.value.trim()}`
    : creating.value.value.trim()
  const isDir = creating.value.isDir
  await filesStore.createFile(path, isDir)
  if (!isDir) {
    await filesStore.openFile(path)
  }
  creating.value = null
}

function handleCreateKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    finishCreate()
  } else if (e.key === 'Escape') {
    creating.value = null
  }
}

async function handleDelete(node: FileNode) {
  if (!confirm(`Delete ${node.name}?`)) return
  await filesStore.deleteFile(node.path)
  closeContextMenu()
}

function collapseAll() {
  expandedDirs.value = new Set()
}

function handleGlobalClick(e: MouseEvent) {
  if (contextMenu.value && contextMenuEl.value && !contextMenuEl.value.contains(e.target as Node)) {
    closeContextMenu()
  }
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    closeContextMenu()
  }
}

onMounted(() => {
  document.addEventListener('mousedown', handleGlobalClick)
  document.addEventListener('keydown', handleGlobalKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', handleGlobalClick)
  document.removeEventListener('keydown', handleGlobalKeydown)
})

interface FlatNode {
  node: FileNode
  depth: number
}

function flattenTree(nodes: FileNode[], depth: number = 0): FlatNode[] {
  const result: FlatNode[] = []
  for (const node of nodes) {
    result.push({ node, depth })
    if (node.is_dir && expandedDirs.value.has(node.path) && node.children) {
      result.push(...flattenTree(node.children, depth + 1))
    }
  }
  return result
}
</script>

<template>
  <div class="file-tree">
    <div class="tree-header">
      <span class="tree-header-title">Explorer</span>
      <div class="tree-header-actions">
        <button
          class="tree-header-btn"
          title="New File"
          @click="startCreate('', false)"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M1.5 1h11l.5.5v4.127A4.954 4.954 0 0 0 11.5 5H11V2H2v11h3.05c.07.38.19.74.36 1.08L5 14.5l-.5.5h-3l-.5-.5v-13z"/>
            <path d="M11.5 7a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7zm.5 3h1.5v1H12v1.5h-1V11H9.5v-1H11V8.5h1z"/>
          </svg>
        </button>
        <button
          class="tree-header-btn"
          title="New Folder"
          @click="startCreate('', true)"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M14 4H9.618l-1-2H2a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1zm-4.5 7H8v1.5H7V11H5.5v-1H7V8.5h1V10h1.5z"/>
          </svg>
        </button>
        <button
          class="tree-header-btn"
          title="Collapse All"
          @click="collapseAll"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
            <path d="M9 9H4v1h5z"/>
            <path fill-rule="evenodd" clip-rule="evenodd" d="m5 3 1-1h7l1 1v7l-1 1h-2v2l-1 1H3l-1-1V6l1-1h2zm1 2h4l1 1v4h2V3H6zm4 1H3v7h7z"/>
          </svg>
        </button>
      </div>
    </div>

    <div class="tree-content">
      <!-- Creating at root level -->
      <div v-if="creating && !creating.parentPath" class="tree-item" :style="{ paddingLeft: '8px' }">
        <span class="tree-chevron-space"></span>
        <span class="tree-icon" v-html="creating.isDir
          ? getFileIcon('new-folder', true, false)
          : getFileIcon('untitled', false, false)
        "></span>
        <input
          ref="createInput"
          v-model="creating.value"
          class="tree-inline-input"
          :placeholder="creating.isDir ? 'folder name' : 'file name'"
          @keydown="handleCreateKeydown"
          @blur="finishCreate"
        />
      </div>

      <template v-for="item in flattenTree(filesStore.tree)" :key="item.node.path">
        <div
          :class="[
            'tree-item',
            {
              active: !item.node.is_dir && item.node.path === filesStore.activeFilePath,
            },
          ]"
          :style="{ paddingLeft: (item.depth * 16 + 8) + 'px' }"
          @click="handleClick(item.node)"
          @contextmenu="handleContextMenu($event, item.node)"
        >
          <!-- Chevron for directories, space for files -->
          <span v-if="item.node.is_dir" class="tree-chevron" :class="{ expanded: expandedDirs.has(item.node.path) }">
            <svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor">
              <path d="M6 4l4 4-4 4z"/>
            </svg>
          </span>
          <span v-else class="tree-chevron-space"></span>

          <!-- Icon -->
          <span
            class="tree-icon"
            v-html="getFileIcon(item.node.name, item.node.is_dir, expandedDirs.has(item.node.path))"
          ></span>

          <!-- Name or rename input -->
          <input
            v-if="renaming && renaming.path === item.node.path"
            ref="renameInput"
            v-model="renaming.value"
            class="tree-inline-input"
            @keydown="handleRenameKeydown"
            @blur="finishRename"
            @click.stop
          />
          <span v-else class="tree-name" :title="item.node.path">{{ item.node.name }}</span>
        </div>

        <!-- Creating inside directory -->
        <div
          v-if="creating && creating.parentPath === item.node.path && item.node.is_dir"
          class="tree-item"
          :style="{ paddingLeft: ((item.depth + 1) * 16 + 8) + 'px' }"
        >
          <span class="tree-chevron-space"></span>
          <span class="tree-icon" v-html="creating.isDir
            ? getFileIcon('new-folder', true, false)
            : getFileIcon('untitled', false, false)
          "></span>
          <input
            ref="createInput"
            v-model="creating.value"
            class="tree-inline-input"
            :placeholder="creating.isDir ? 'folder name' : 'file name'"
            @keydown="handleCreateKeydown"
            @blur="finishCreate"
          />
        </div>
      </template>

      <div v-if="!filesStore.tree.length && !filesStore.loading" class="tree-empty">
        No files
      </div>

      <div v-if="filesStore.loading" class="tree-empty">
        Loading...
      </div>
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu"
        ref="contextMenuEl"
        class="context-menu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      >
        <template v-if="contextMenu.node.is_dir">
          <div class="menu-item" @click="startCreate(contextMenu.node.path, false)">New File</div>
          <div class="menu-item" @click="startCreate(contextMenu.node.path, true)">New Folder</div>
          <div class="menu-sep"></div>
        </template>
        <div class="menu-item" @click="copyPath(contextMenu!.node)">Copy Path</div>
        <div class="menu-item" @click="openInFileManager(contextMenu!.node)">Open in File Manager</div>
        <div class="menu-sep"></div>
        <div class="menu-item" @click="startRename(contextMenu!.node)">Rename</div>
        <div class="menu-item danger" @click="handleDelete(contextMenu!.node)">Delete</div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.file-tree {
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  height: 100%;
  user-select: none;
}

.tree-header {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.tree-header-title {
  white-space: nowrap;
}

.tree-header-actions {
  display: flex;
  gap: 2px;
}

.tree-header-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  padding: 2px 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  opacity: 0.7;
}

.tree-header-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
  opacity: 1;
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 4px 0;
}

.tree-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  height: 24px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
}

.tree-item:hover {
  background: rgba(255, 255, 255, 0.04);
}

.tree-item.active {
  background: rgba(88, 166, 255, 0.12);
  color: var(--text-primary);
}

.tree-chevron {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.15s ease;
  color: var(--text-secondary);
}

.tree-chevron.expanded {
  transform: rotate(90deg);
}

.tree-chevron-space {
  width: 16px;
  flex-shrink: 0;
}

.tree-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tree-icon :deep(svg) {
  width: 18px;
  height: 18px;
}

.tree-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.tree-inline-input {
  flex: 1;
  min-width: 0;
  height: 20px;
  padding: 0 4px;
  font-size: 13px;
  font-family: var(--font-ui);
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--accent-blue);
  border-radius: 3px;
  outline: none;
}

.tree-empty {
  padding: 12px;
  font-size: 12px;
  color: var(--text-secondary);
  text-align: center;
}

/* Context menu */
.context-menu {
  position: fixed;
  z-index: 1000;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 4px 0;
  min-width: 160px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  font-family: var(--font-ui);
  font-size: 13px;
}

.menu-item {
  padding: 6px 12px;
  color: var(--text-primary);
  cursor: pointer;
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

.menu-sep {
  height: 1px;
  background: var(--border);
  margin: 4px 0;
}
</style>
