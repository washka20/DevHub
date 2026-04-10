import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
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

  const error = ref<string | null>(null)

  // Local selection (checkboxes) — no git calls until user clicks Stage/Unstage button
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

  // Real git add — called by button
  async function stageSelected() {
    const files = Array.from(selectedFiles.value)
    if (files.length === 0) return
    try {
      await fetch(`${projectApiUrl.value}/git/stage`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ files }),
      })
      selectedFiles.value = new Set()
      await fetchStatus()
    } catch (e) {
      error.value = getErrorMessage(e)
    }
  }

  // Real git reset — called by button
  async function unstageAll() {
    const files = status.value.staged || []
    if (files.length === 0) return
    try {
      await fetch(`${projectApiUrl.value}/git/unstage`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ files }),
      })
      await fetchStatus()
    } catch (e) {
      error.value = getErrorMessage(e)
    }
  }

  function isLocallyStaged(file: string): boolean {
    return (status.value.staged || []).includes(file)
  }

  async function fetchStatus() {
    loading.value.status = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/status`)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      status.value = {
        branch: data.branch ?? '',
        modified: data.modified ?? [],
        staged: data.staged ?? [],
        untracked: data.untracked ?? [],
        ahead: data.ahead ?? 0,
        behind: data.behind ?? 0,
      }
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.status = false
    }
    // Also refresh stash list
    fetchStash()
  }

  async function fetchBranches() {
    loading.value.branches = true
    try {
      const res = await fetch(`${projectApiUrl.value}/git/branches`)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      // Support both string[] and BranchInfo[] responses from API
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
      error.value = getErrorMessage(e)
    } finally {
      loading.value.branches = false
    }
  }

  async function fetchLog() {
    loading.value.log = true
    try {
      const res = await fetch(`${projectApiUrl.value}/git/log/metadata?offset=0&limit=5`)
      if (!res.ok) throw new Error(await res.text())
      const data: CommitMeta[] = await res.json()
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
      error.value = getErrorMessage(e)
    } finally {
      loading.value.log = false
    }
  }

  const LOG_PAGE_SIZE = 50

  // Topology — загружается один раз
  const topoNodes = ref<TopoNode[]>([])

  // Простой вид для рендеринга
  const graphNodes = computed(() => topoNodes.value)

  // Метаданные — загружаются порциями
  const metadataMap = ref<Map<string, CommitMeta>>(new Map())
  const metadataLoaded = ref(0)
  const metadataLoading = ref(false)

  const totalCommits = computed(() => topoNodes.value.length)

  async function fetchGraph() {
    loading.value.log = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/graph`)
      if (!res.ok) throw new Error(await res.text())
      const data: TopoNode[] = await res.json()
      // Нормализуем parents: null → []
      topoNodes.value = data.map(n => ({ id: n.id, parents: n.parents ?? [] }))
      metadataMap.value = new Map()
      metadataLoaded.value = 0
      // Сразу подгружаем первую порцию
      await fetchMetadata(0, LOG_PAGE_SIZE)
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.log = false
    }
  }

  async function fetchMetadata(offset: number, limit: number) {
    if (metadataLoading.value) return
    metadataLoading.value = true
    try {
      let url = `${projectApiUrl.value}/git/log/metadata?offset=${offset}&limit=${limit}`
      if (viewingBranch.value) {
        url += `&branch=${encodeURIComponent(viewingBranch.value)}`
      }
      const res = await fetch(url)
      if (!res.ok) throw new Error(await res.text())
      const data: CommitMeta[] = await res.json()
      const map = new Map(metadataMap.value)
      for (const m of data) {
        map.set(m.hash, m)
      }
      metadataMap.value = map
      metadataLoaded.value = offset + data.length
    } catch (e) {
      error.value = getErrorMessage(e)
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
      const url = file
        ? `${projectApiUrl.value}/git/diff?file=${encodeURIComponent(file)}`
        : `${projectApiUrl.value}/git/diff`
      const res = await fetch(url)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      diff.value = data.diff ?? ''
      if (file) {
        selectedFile.value = file
      }
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.diff = false
    }
  }

  async function fetchCommitDetail(hash: string) {
    loading.value.commitDetail = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/commits/${hash}`)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      selectedCommit.value = data as CommitDetail
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.commitDetail = false
    }
  }

  async function fetchCommitDiff(hash: string, file?: string) {
    loading.value.commitDiff = true
    error.value = null
    try {
      const url = file
        ? `${projectApiUrl.value}/git/commits/${hash}/diff?file=${encodeURIComponent(file)}`
        : `${projectApiUrl.value}/git/commits/${hash}/diff`
      const res = await fetch(url)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      diff.value = data.diff ?? ''
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.commitDiff = false
    }
  }

  async function generateCommitMessage() {
    generatingMessage.value = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/generate-commit`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      commitMessage.value = data.message ?? ''
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      generatingMessage.value = false
    }
  }

  async function commit(message: string, files: string[]) {
    loading.value.commit = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/commit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, files }),
      })
      if (!res.ok) throw new Error(await res.text())
      commitMessage.value = ''
      await Promise.all([fetchStatus(), fetchGraph()])
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.commit = false
    }
  }

  async function checkout(branch: string) {
    loading.value.checkout = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/checkout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ branch }),
      })
      if (!res.ok) throw new Error(await res.text())
      await Promise.all([fetchStatus(), fetchBranches(), fetchGraph()])
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.checkout = false
    }
  }

  async function pull() {
    loading.value.pull = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/pull`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error(await res.text())
      await Promise.all([fetchStatus(), fetchGraph()])
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.pull = false
    }
  }

  async function push() {
    loading.value.push = true
    error.value = null
    try {
      const res = await fetch(`${projectApiUrl.value}/git/push`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error(await res.text())
      await fetchStatus()
    } catch (e) {
      error.value = getErrorMessage(e)
    } finally {
      loading.value.push = false
    }
  }

  // Set viewing branch and refetch metadata
  function setViewingBranch(branch: string) {
    viewingBranch.value = branch
    metadataMap.value = new Map()
    metadataLoaded.value = 0
    fetchMetadata(0, 50)
  }

  // Fetch recent commits for a specific branch (for card expansion)
  async function fetchBranchCommits(branch: string) {
    try {
      const res = await fetch(`${projectApiUrl.value}/git/branches/${encodeURIComponent(branch)}/commits?limit=5`)
      if (!res.ok) throw new Error(await res.text())
      const data: CommitMeta[] = await res.json()
      const map = new Map(branchCommits.value)
      map.set(branch, data)
      branchCommits.value = map
    } catch (e) {
      error.value = getErrorMessage(e)
    }
  }

  // Stash actions
  async function fetchStash() {
    try {
      const res = await fetch(`${projectApiUrl.value}/git/stash`)
      if (res.ok) stashEntries.value = await res.json()
    } catch { /* best-effort, non-critical */ }
  }

  async function stashPush(message: string) {
    stashLoading.value = true
    try {
      const res = await fetch(`${projectApiUrl.value}/git/stash`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message }),
      })
      if (!res.ok) throw new Error(await res.text())
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
      const res = await fetch(`${projectApiUrl.value}/git/stash/${index}/apply`, { method: 'POST' })
      if (!res.ok) throw new Error(await res.text())
      await fetchStatus()
    } catch (e) {
      toast.show('error', `Stash apply failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashPop(index: number) {
    try {
      const res = await fetch(`${projectApiUrl.value}/git/stash/${index}/pop`, { method: 'POST' })
      if (!res.ok) throw new Error(await res.text())
      await fetchStash()
      await fetchStatus()
    } catch (e) {
      toast.show('error', `Stash pop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDrop(index: number) {
    try {
      const res = await fetch(`${projectApiUrl.value}/git/stash/${index}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(await res.text())
      await fetchStash()
    } catch (e) {
      toast.show('error', `Stash drop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDiff(index: number): Promise<string> {
    const res = await fetch(`${projectApiUrl.value}/git/stash/${index}/diff`)
    if (!res.ok) return ''
    const data = await res.json()
    return data.diff || ''
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
    error,
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
