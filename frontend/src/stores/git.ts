import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProject } from '../composables/useProject'
import type { GitStatus, Commit, CommitDetail, BranchInfo, GraphData } from '../types'

export const useGitStore = defineStore('git', () => {
  const { projectApiUrl } = useProject()

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

  // New state for tabs and commit details
  const activeTab = ref<'changes' | 'log' | 'branches'>('changes')
  const commits = ref<Commit[]>([])
  const selectedCommit = ref<CommitDetail | null>(null)
  const commitMessage = ref('')
  const generatingMessage = ref(false)

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
      error.value = (e as Error).message
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
      error.value = (e as Error).message
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
      error.value = (e as Error).message
    } finally {
      loading.value.status = false
    }
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
      error.value = (e as Error).message
    } finally {
      loading.value.branches = false
    }
  }

  const LOG_PAGE_SIZE = 50
  const logOffset = ref(0)
  const logHasMore = ref(true)
  const logLoadingMore = ref(false)

  function parseCommits(data: unknown[]): Commit[] {
    return (data ?? []).map((item: unknown) => {
      const c = item as Record<string, unknown>
      return {
      hash: (c.hash ?? '') as string,
      short_hash: (c.short_hash ?? (typeof c.hash === 'string' ? (c.hash as string).slice(0, 7) : '')) as string,
      message: (c.message ?? '') as string,
      author: (c.author ?? '') as string,
      date: (c.date ?? '') as string,
      refs: Array.isArray(c.refs) ? c.refs as string[] : [],
      parents: Array.isArray(c.parents) ? c.parents as string[] : [],
      graph: typeof c.graph === 'string' ? c.graph as string : '',
      graph_only: !!c.graph_only,
      graph_data: c.graph_data as GraphData | undefined,
    }})
  }

  async function fetchLog() {
    loading.value.log = true
    logOffset.value = 0
    logHasMore.value = true
    try {
      const res = await fetch(`${projectApiUrl.value}/git/log?limit=${LOG_PAGE_SIZE}&offset=0`)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      const rawCommits = parseCommits(data)
      log.value = rawCommits
      commits.value = rawCommits
      logOffset.value = rawCommits.length
      logHasMore.value = rawCommits.length >= LOG_PAGE_SIZE
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value.log = false
    }
  }

  async function fetchMoreLog() {
    if (logLoadingMore.value || !logHasMore.value) return
    logLoadingMore.value = true
    try {
      // Запрашиваем ВСЕ коммиты с увеличенным limit, чтобы git2graph
      // посчитал граф для полного набора (иначе линии обрываются на границе страниц)
      const newLimit = logOffset.value + LOG_PAGE_SIZE
      const res = await fetch(`${projectApiUrl.value}/git/log?limit=${newLimit}&offset=0`)
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      const allCommits = parseCommits(data)
      commits.value = allCommits
      log.value = allCommits
      logHasMore.value = allCommits.length >= newLimit
      logOffset.value = allCommits.length
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      logLoadingMore.value = false
    }
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
      error.value = (e as Error).message
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
      error.value = (e as Error).message
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
      error.value = (e as Error).message
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
      error.value = (e as Error).message
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
      await Promise.all([fetchStatus(), fetchLog()])
    } catch (e) {
      error.value = (e as Error).message
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
      await Promise.all([fetchStatus(), fetchBranches(), fetchLog()])
    } catch (e) {
      error.value = (e as Error).message
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
      await Promise.all([fetchStatus(), fetchLog()])
    } catch (e) {
      error.value = (e as Error).message
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
      error.value = (e as Error).message
    } finally {
      loading.value.push = false
    }
  }

  return {
    status,
    branches,
    log,
    diff,
    selectedFile,
    activeTab,
    commits,
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
    fetchMoreLog,
    logHasMore,
    logLoadingMore,
    fetchDiff,
    fetchCommitDetail,
    fetchCommitDiff,
    generateCommitMessage,
    commit,
    checkout,
    pull,
    push,
  }
})
