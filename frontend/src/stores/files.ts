import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import { filesApi } from '../api/files'
import type { FileNode, OpenFile } from '../types'

const MAX_TABS = 5

function detectLanguage(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  const map: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    vue: 'vue',
    html: 'html',
    htm: 'html',
    css: 'css',
    scss: 'scss',
    less: 'css',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    go: 'go',
    py: 'python',
    php: 'php',
    sql: 'sql',
    xml: 'xml',
    rs: 'rust',
    sh: 'shell',
    bash: 'shell',
  }
  return map[ext] ?? 'text'
}

const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'])

function isImage(filename: string): boolean {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  return IMAGE_EXTENSIONS.has(ext)
}

export const useFilesStore = defineStore('files', () => {
  const projectsStore = useProjectsStore()

  function projectName(): string {
    return projectsStore.currentProject?.name ?? '_'
  }

  const tree = ref<FileNode[]>([])
  const openFiles = ref<OpenFile[]>([])
  const activeFilePath = ref<string | null>(null)
  const loading = ref(false)
  const changedOnDisk = ref<Set<string>>(new Set())

  const activeFile = computed<OpenFile | undefined>(() =>
    openFiles.value.find((f) => f.path === activeFilePath.value)
  )

  const hasUnsaved = computed<boolean>(() => openFiles.value.some((f) => f.dirty))

  async function fetchTree() {
    loading.value = true
    try {
      tree.value = await filesApi.tree(projectName())
    } finally {
      loading.value = false
    }
  }

  async function openFile(path: string) {
    const existing = openFiles.value.find((f) => f.path === path)
    if (existing) {
      activeFilePath.value = path
      return
    }

    if (openFiles.value.length >= MAX_TABS) {
      const oldest = openFiles.value.find((f) => !f.dirty && f.path !== activeFilePath.value)
      if (oldest) {
        closeFile(oldest.path)
      }
    }

    const name = path.split('/').pop() ?? path

    if (isImage(name)) {
      openFiles.value.push({
        path, name, content: '', originalContent: '', dirty: false, language: 'image',
      })
      activeFilePath.value = path
      return
    }

    try {
      const content = await filesApi.content(projectName(), path)
      openFiles.value.push({
        path,
        name,
        content,
        originalContent: content,
        dirty: false,
        language: detectLanguage(name),
      })
      activeFilePath.value = path
    } catch { /* file not readable */ }
  }

  async function saveFile(path: string) {
    const file = openFiles.value.find((f) => f.path === path)
    if (!file) return

    try {
      await filesApi.save(projectName(), path, file.content)
      file.originalContent = file.content
      file.dirty = false
      changedOnDisk.value.delete(path)
    } catch { /* save failed */ }
  }

  function updateContent(path: string, content: string) {
    const file = openFiles.value.find((f) => f.path === path)
    if (!file) return
    file.content = content
    file.dirty = content !== file.originalContent
  }

  function closeFile(path: string) {
    const idx = openFiles.value.findIndex((f) => f.path === path)
    if (idx === -1) return

    openFiles.value.splice(idx, 1)

    if (activeFilePath.value === path) {
      if (openFiles.value.length > 0) {
        const next = openFiles.value[Math.min(idx, openFiles.value.length - 1)]
        activeFilePath.value = next.path
      } else {
        activeFilePath.value = null
      }
    }
  }

  async function createFile(path: string, isDir: boolean) {
    await filesApi.create(projectName(), path, isDir)
    await fetchTree()
  }

  async function deleteFile(path: string) {
    await filesApi.delete(projectName(), path)
    closeFile(path)
    await fetchTree()
  }

  async function openInFileManager(path: string) {
    filesApi.openInFileManager(projectName(), path)
  }

  async function renameFile(oldPath: string, newPath: string) {
    await filesApi.rename(projectName(), oldPath, newPath)

    const file = openFiles.value.find((f) => f.path === oldPath)
    if (file) {
      file.path = newPath
      file.name = newPath.split('/').pop() ?? newPath
      file.language = detectLanguage(file.name)
      if (activeFilePath.value === oldPath) {
        activeFilePath.value = newPath
      }
    }

    await fetchTree()
  }

  async function checkOpenFiles(paths: string[]) {
    for (const path of paths) {
      const file = openFiles.value.find((f) => f.path === path)
      if (!file) continue

      if (file.dirty) {
        changedOnDisk.value = new Set([...changedOnDisk.value, path])
      } else {
        await reloadFromDisk(path)
      }
    }
  }

  function dismissDiskChange(path: string) {
    changedOnDisk.value.delete(path)
  }

  async function reloadFromDisk(path: string) {
    const file = openFiles.value.find((f) => f.path === path)
    if (!file) return

    try {
      const content = await filesApi.content(projectName(), path)
      file.content = content
      file.originalContent = content
      file.dirty = false
      changedOnDisk.value.delete(path)
    } catch { /* file not readable */ }
  }

  return {
    tree,
    openFiles,
    activeFilePath,
    loading,
    changedOnDisk,
    activeFile,
    hasUnsaved,
    fetchTree,
    openFile,
    saveFile,
    updateContent,
    closeFile,
    createFile,
    deleteFile,
    renameFile,
    openInFileManager,
    checkOpenFiles,
    dismissDiskChange,
    reloadFromDisk,
  }
})
