import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { GitStatus, CommitDetail, BranchInfo, CommitMeta, Commit, StashEntry } from '../types'

interface TopoNode {
  id: string
  parents: string[]
}

export const useGitStore = defineStore('git', () => {
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
  const branches = ref<BranchInfo[]>([])
  const log = ref<Commit[]>([])
  const diff = ref('')
  const selectedFile = ref<string | null>(null)

  // Branch browsing state
  const viewingBranch = ref<string>('') // empty = all branches
  const branchCommits = ref<Map<string, CommitMeta[]>>(new Map())

  // New state for tabs and commit details
  const activeTab = ref<'changes' | 'log' | 'branches'>('changes')
  const selectedCommit = ref<CommitDetail | null>(null)
  const commitMessage = ref('')
  const generatingMessage = ref(false)

  // Stash state
  const stashEntries = ref<StashEntry[]>([])
  const stashLoading = ref(false)

  const loading = ref({
    status: false,
    branches: false,
    log: false,
    diff: false,
    commit: false,
    checkout: false,
    pull: false,
    push: false,
    commitDetail: false,
    commitDiff: false,
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
    fetchStash()
  }

  async function fetchBranches() {
    loading.value.branches = true
    try {
      const data = await gitApi.branches(projectApiUrl.value)
      if (Array.isArray(data) && data.length > 0) {
        if (typeof data[0] === 'string') {
          branches.value = (data as string[]).map((name: string) => ({
            name,
            short_hash: '',
            message: '',
            author: '',
            date: '',
            is_current: name === status.value.branch,
            ahead: 0,
            behind: 0,
            is_merged: false,
          }))
        } else {
          branches.value = data as BranchInfo[]
        }
      } else {
        branches.value = []
      }
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.branches = false
    }
  }

  async function fetchLog() {
    loading.value.log = true
    try {
      const data = await gitApi.logMetadata(projectApiUrl.value, 0, 5)
      log.value = data.map(m => ({
        hash: m.hash,
        short_hash: m.short_hash,
        message: m.message,
        author: m.author,
        date: m.date,
        refs: m.refs,
        parents: [],
      }))
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.log = false
    }
  }

  const LOG_PAGE_SIZE = 50

  const topoNodes = ref<TopoNode[]>([])
  const graphNodes = computed(() => topoNodes.value)
  const metadataMap = ref<Map<string, CommitMeta>>(new Map())
  const metadataLoaded = ref(0)
  const metadataLoading = ref(false)
  const totalCommits = computed(() => topoNodes.value.length)

  async function fetchGraph() {
    loading.value.log = true
    try {
      const data = await gitApi.graph(projectApiUrl.value)
      topoNodes.value = data.map(n => ({ id: n.id, parents: n.parents ?? [] }))
      metadataMap.value = new Map()
      metadataLoaded.value = 0
      await fetchMetadata(0, LOG_PAGE_SIZE)
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.log = false
    }
  }

  async function fetchMetadata(offset: number, limit: number) {
    if (metadataLoading.value) return
    metadataLoading.value = true
    try {
      const data = await gitApi.logMetadata(projectApiUrl.value, offset, limit, viewingBranch.value || undefined)
      const map = new Map(metadataMap.value)
      for (const m of data) {
        map.set(m.hash, m)
      }
      metadataMap.value = map
      metadataLoaded.value = offset + data.length
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      metadataLoading.value = false
    }
  }

  function getMetadata(hash: string): CommitMeta | undefined {
    return metadataMap.value.get(hash)
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

  async function fetchCommitDetail(hash: string) {
    loading.value.commitDetail = true
    try {
      const data = await gitApi.commitDetail(projectApiUrl.value, hash)
      selectedCommit.value = data as CommitDetail
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.commitDetail = false
    }
  }

  async function fetchCommitDiff(hash: string, file?: string) {
    loading.value.commitDiff = true
    try {
      const data = await gitApi.commitDiff(projectApiUrl.value, hash, file)
      diff.value = data.diff ?? ''
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.commitDiff = false
    }
  }

  async function generateCommitMessage() {
    generatingMessage.value = true
    try {
      const data = await gitApi.generateCommit(projectApiUrl.value)
      commitMessage.value = data.message ?? ''
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      generatingMessage.value = false
    }
  }

  async function commit(message: string, files: string[]) {
    loading.value.commit = true
    try {
      await gitApi.commit(projectApiUrl.value, message, files)
      commitMessage.value = ''
      await Promise.all([fetchStatus(), fetchGraph()])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.commit = false
    }
  }

  async function checkout(branch: string) {
    loading.value.checkout = true
    try {
      await gitApi.checkout(projectApiUrl.value, branch)
      await Promise.all([fetchStatus(), fetchBranches(), fetchGraph()])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.checkout = false
    }
  }

  async function pull() {
    loading.value.pull = true
    try {
      await gitApi.pull(projectApiUrl.value)
      await Promise.all([fetchStatus(), fetchGraph()])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.pull = false
    }
  }

  async function push() {
    loading.value.push = true
    try {
      await gitApi.push(projectApiUrl.value)
      await fetchStatus()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value.push = false
    }
  }

  function setViewingBranch(branch: string) {
    viewingBranch.value = branch
    metadataMap.value = new Map()
    metadataLoaded.value = 0
    fetchMetadata(0, 50)
  }

  async function fetchBranchCommits(branch: string) {
    try {
      const data = await gitApi.branchCommits(projectApiUrl.value, branch, 5)
      const map = new Map(branchCommits.value)
      map.set(branch, data)
      branchCommits.value = map
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    }
  }

  // Stash actions
  async function fetchStash() {
    try {
      stashEntries.value = await gitApi.stashList(projectApiUrl.value)
    } catch { /* best-effort, non-critical */ }
  }

  async function stashPush(message: string) {
    stashLoading.value = true
    try {
      await gitApi.stashPush(projectApiUrl.value, message)
      await fetchStash()
      await fetchStatus()
    } catch (e) {
      toast.show('error', `Stash push failed: ${getErrorMessage(e)}`)
    } finally {
      stashLoading.value = false
    }
  }

  async function stashApply(index: number) {
    try {
      await gitApi.stashApply(projectApiUrl.value, index)
      await fetchStatus()
    } catch (e) {
      toast.show('error', `Stash apply failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashPop(index: number) {
    try {
      await gitApi.stashPop(projectApiUrl.value, index)
      await fetchStash()
      await fetchStatus()
    } catch (e) {
      toast.show('error', `Stash pop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDrop(index: number) {
    try {
      await gitApi.stashDrop(projectApiUrl.value, index)
      await fetchStash()
    } catch (e) {
      toast.show('error', `Stash drop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDiff(index: number): Promise<string> {
    try {
      const data = await gitApi.stashDiff(projectApiUrl.value, index)
      return data.diff || ''
    } catch {
      return ''
    }
  }

  return {
    status,
    branches,
    log,
    diff,
    selectedFile,
    viewingBranch,
    branchCommits,
    activeTab,
    selectedCommit,
    commitMessage,
    generatingMessage,
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
    fetchBranches,
    fetchLog,
    graphNodes,
    metadataMap,
    metadataLoaded,
    metadataLoading,
    totalCommits,
    fetchGraph,
    fetchMetadata,
    getMetadata,
    fetchDiff,
    fetchCommitDetail,
    fetchCommitDiff,
    generateCommitMessage,
    commit,
    checkout,
    pull,
    push,
    setViewingBranch,
    fetchBranchCommits,
    stashEntries,
    stashLoading,
    fetchStash,
    stashPush,
    stashApply,
    stashPop,
    stashDrop,
    stashDiff,
  }
})
