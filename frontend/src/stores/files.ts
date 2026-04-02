import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
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

  function apiBase(): string {
    const project = projectsStore.currentProject
    if (!project) return '/api/projects/_'
    return `/api/projects/${project.name}`
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
      const res = await fetch(`${apiBase()}/files/tree`)
      if (res.ok) {
        tree.value = await res.json()
      }
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

    // Enforce max tabs: close oldest non-dirty if overflow
    if (openFiles.value.length >= MAX_TABS) {
      const oldest = openFiles.value.find((f) => !f.dirty && f.path !== activeFilePath.value)
      if (oldest) {
        closeFile(oldest.path)
      }
    }

    const name = path.split('/').pop() ?? path

    // Image files: no content fetch
    if (isImage(name)) {
      openFiles.value.push({
        path, name, content: '', originalContent: '', dirty: false, language: 'image',
      })
      activeFilePath.value = path
      return
    }

    // Text files: fetch content (existing logic)
    const res = await fetch(`${apiBase()}/files/content/${encodeURIComponent(path)}`)
    if (!res.ok) return

    const content = await res.text()
    openFiles.value.push({
      path,
      name,
      content,
      originalContent: content,
      dirty: false,
      language: detectLanguage(name),
    })
    activeFilePath.value = path
  }

  async function saveFile(path: string) {
    const file = openFiles.value.find((f) => f.path === path)
    if (!file) return

    const res = await fetch(`${apiBase()}/files/content/${encodeURIComponent(path)}`, {
      method: 'PUT',
      body: file.content,
    })
    if (res.ok) {
      file.originalContent = file.content
      file.dirty = false
      changedOnDisk.value.delete(path)
    }
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
    await fetch(`${apiBase()}/files/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, is_dir: isDir }),
    })
    await fetchTree()
  }

  async function deleteFile(path: string) {
    await fetch(`${apiBase()}/files/delete/${encodeURIComponent(path)}`, {
      method: 'DELETE',
    })
    closeFile(path)
    await fetchTree()
  }

  async function renameFile(oldPath: string, newPath: string) {
    await fetch(`${apiBase()}/files/rename/${encodeURIComponent(oldPath)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ new_path: newPath }),
    })

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

    const res = await fetch(`${apiBase()}/files/content/${encodeURIComponent(path)}`)
    if (!res.ok) return

    const content = await res.text()
    file.content = content
    file.originalContent = content
    file.dirty = false
    changedOnDisk.value.delete(path)
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
    checkOpenFiles,
    dismissDiskChange,
    reloadFromDisk,
  }
})
