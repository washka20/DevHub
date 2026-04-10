import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { GitStatus } from '../types'

export const useGitStatusStore = defineStore('gitStatus', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const status = ref<GitStatus>({
    branch: '',
    modified: [],
    staged: [],
    untracked: [],
    ahead: 0,
    behind: 0,
  })

  const selectedFile = ref<string | null>(null)
  const diff = ref('')

  const loading = ref({
    status: false,
    diff: false,
  })

  // Local selection (checkboxes)
  const selectedFiles = ref<Set<string>>(new Set())

  const stagedFiles = computed(() => status.value.staged || [])

  const totalModified = computed(
    () => (status.value.modified?.length || 0) + (status.value.untracked?.length || 0),
  )

  const totalStaged = computed(
    () => status.value.staged?.length || 0,
  )

  function toggleSelectFile(file: string) {
    const s = new Set(selectedFiles.value)
    if (s.has(file)) s.delete(file); else s.add(file)
    selectedFiles.value = s
  }

  function selectAllUnstaged() {
    const s = new Set(selectedFiles.value)
    for (const f of (status.value.modified || [])) s.add(f)
    for (const f of (status.value.untracked || [])) s.add(f)
    selectedFiles.value = s
  }

  function clearSelection() {
    selectedFiles.value = new Set()
  }

  function isSelected(file: string): boolean {
    return selectedFiles.value.has(file)
  }

  async function stageSelected() {
    const files = Array.from(selectedFiles.value)
    if (files.length === 0) return
    try {
      await gitApi.stage(projectApiUrl.value, files)
      selectedFiles.value = new Set()
      await fetchStatus()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    }
  }

  async function unstageAll() {
    const files = status.value.staged || []
    if (files.length === 0) return
    try {
      await gitApi.unstage(projectApiUrl.value, files)
      await fetchStatus()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    }
  }

  function isLocallyStaged(file: string): boolean {
    return (status.value.staged || []).includes(file)
  }

  async function fetchStatus() {
    loading.value.status = true
    try {
      const data = await gitApi.status(projectApiUrl.value)
      status.value = {
        branch: data.branch ?? '',
        modified: data.modified ?? [],
        staged: data.staged ?? [],
        untracked: data.untracked ?? [],
        ahead: data.ahead ?? 0,
        behind: data.behind ?? 0,
      }
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.status = false
    }
    // Trigger stash refresh from stash store
    const { useGitStashStore } = await import('./gitStash')
    useGitStashStore().fetchStash()
  }

  async function fetchDiff(file?: string) {
    loading.value.diff = true
    try {
      const data = await gitApi.diff(projectApiUrl.value, file)
      diff.value = data.diff ?? ''
      if (file) {
        selectedFile.value = file
      }
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.diff = false
    }
  }

  return {
    status,
    selectedFile,
    diff,
    loading,
    selectedFiles,
    stagedFiles,
    totalModified,
    totalStaged,
    toggleSelectFile,
    selectAllUnstaged,
    clearSelection,
    isSelected,
    stageSelected,
    unstageAll,
    isLocallyStaged,
    fetchStatus,
    fetchDiff,
  }
})
