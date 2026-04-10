import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import type {
  GitLabIssue,
  GitLabMR,
  GitLabPipeline,
  GitLabProject,
  GitLabNote,
  GitLabLabel,
  GitLabMilestone,
  GitLabMember,
} from '../types'

export const useGitLabStore = defineStore('gitlab', () => {
  const projectsStore = useProjectsStore()

  function apiBase(): string {
    const project = projectsStore.currentProject
    if (!project) return '/api/projects/_'
    return `/api/projects/${project.name}`
  }

  // Tab state
  const activeMainTab = ref<'tasks' | 'mrs' | 'project' | 'pipelines'>('tasks')

  // Cross-project: My Issues
  const myIssues = ref<GitLabIssue[]>([])
  const myIssuesState = ref<'opened' | 'closed'>('opened')
  const myIssuesLoading = ref(false)

  // Cross-project: My MRs
  const myMRs = ref<GitLabMR[]>([])
  const myMRsState = ref<'opened' | 'merged' | 'closed'>('opened')
  const myMRsLoading = ref(false)

  // Detail panel
  const selectedItem = ref<{ type: 'issue' | 'mr'; projectPath: string; iid: number } | null>(null)
  const detailIssue = ref<GitLabIssue | null>(null)
  const detailMR = ref<GitLabMR | null>(null)
  const detailNotes = ref<GitLabNote[]>([])
  const detailLoading = ref(false)

  // Filters
  const labels = ref<GitLabLabel[]>([])
  const milestones = ref<GitLabMilestone[]>([])
  const members = ref<GitLabMember[]>([])
  const searchQuery = ref('')
  const filterLabels = ref<string[]>([])
  const filterMilestone = ref('')
  const filterAssignee = ref('')

  // Create modals
  const showCreateIssue = ref(false)
  const showCreateMR = ref(false)

  // Per-project state (existing)
  const enabled = ref<boolean | null>(null)
  const project = ref<GitLabProject | null>(null)
  const issues = ref<GitLabIssue[]>([])
  const mergeRequests = ref<GitLabMR[]>([])
  const pipelines = ref<GitLabPipeline[]>([])
  const projectLoading = ref(false)

  // Auto-refresh
  let refreshInterval: ReturnType<typeof setInterval> | null = null

  // Computed
  const openIssuesCount = computed(() =>
    myIssues.value.filter(i => i.state === 'opened').length
  )

  const groupedMyIssues = computed(() => {
    const groups: Record<string, GitLabIssue[]> = {}
    const filtered = filteredMyIssues.value
    for (const issue of filtered) {
      const key = issue.project_path || 'unknown'
      if (!groups[key]) groups[key] = []
      groups[key].push(issue)
    }
    return groups
  })

  const groupedMyMRs = computed(() => {
    const groups: Record<string, GitLabMR[]> = {}
    const filtered = filteredMyMRs.value
    for (const mr of filtered) {
      const key = mr.project_path || 'unknown'
      if (!groups[key]) groups[key] = []
      groups[key].push(mr)
    }
    return groups
  })

  const filteredMyIssues = computed(() => {
    let result = myIssues.value
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      result = result.filter(i =>
        i.title.toLowerCase().includes(q) || `#${i.iid}`.includes(q)
      )
    }
    if (filterLabels.value.length > 0) {
      result = result.filter(i =>
        filterLabels.value.every(l => i.labels.includes(l))
      )
    }
    if (filterMilestone.value) {
      result = result.filter(i => i.milestone?.title === filterMilestone.value)
    }
    if (filterAssignee.value) {
      result = result.filter(i =>
        i.assignees.some(a => a.username === filterAssignee.value)
      )
    }
    return result
  })

  const filteredMyMRs = computed(() => {
    let result = myMRs.value
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      result = result.filter(m =>
        m.title.toLowerCase().includes(q) || `!${m.iid}`.includes(q)
      )
    }
    if (filterLabels.value.length > 0) {
      result = result.filter(m =>
        filterLabels.value.every(l => m.labels.includes(l))
      )
    }
    return result
  })

  // Cross-project fetches
  async function fetchMyIssues(state?: string) {
    myIssuesLoading.value = true
    try {
      const s = state || myIssuesState.value
      const res = await fetch(`/api/gitlab/my/issues?state=${s}`)
      if (!res.ok) throw new Error(await res.text())
      myIssues.value = enrichProjectPath(await res.json() ?? [])
    } catch (e) {
      console.error('Failed to fetch my issues:', e)
      myIssues.value = []
    } finally {
      myIssuesLoading.value = false
    }
  }

  async function fetchMyMRs(state?: string) {
    myMRsLoading.value = true
    try {
      const s = state || myMRsState.value
      const res = await fetch(`/api/gitlab/my/merge-requests?state=${s}`)
      if (!res.ok) throw new Error(await res.text())
      myMRs.value = enrichProjectPath(await res.json() ?? [])
    } catch (e) {
      console.error('Failed to fetch my MRs:', e)
      myMRs.value = []
    } finally {
      myMRsLoading.value = false
    }
  }

  async function fetchLabels() {
    try {
      const res = await fetch('/api/gitlab/labels')
      if (res.ok) labels.value = await res.json() ?? []
    } catch { /* ignore */ }
  }

  async function fetchMilestones() {
    try {
      const res = await fetch('/api/gitlab/milestones')
      if (res.ok) milestones.value = await res.json() ?? []
    } catch { /* ignore */ }
  }

  async function fetchCurrentUser() {
    try {
      const res = await fetch('/api/gitlab/user')
      if (res.ok) {
        const user = await res.json()
        if (user) {
          members.value = [user, ...members.value.filter(m => m.id !== user.id)]
        }
      }
    } catch { /* ignore */ }
  }

  // Detail — uses GitLab project ID directly (not DevHub project)
  async function fetchIssueDetail(pid: number, iid: number) {
    detailLoading.value = true
    try {
      const res = await fetch(`/api/gitlab/projects/${pid}/issues/${iid}`)
      if (!res.ok) throw new Error(await res.text())
      detailIssue.value = await res.json()
    } catch (e) {
      console.error('Failed to fetch issue detail:', e)
    } finally {
      detailLoading.value = false
    }
  }

  async function fetchIssueNotes(pid: number, iid: number) {
    try {
      const res = await fetch(`/api/gitlab/projects/${pid}/issues/${iid}/notes`)
      if (!res.ok) throw new Error(await res.text())
      detailNotes.value = await res.json() ?? []
    } catch (e) {
      console.error('Failed to fetch issue notes:', e)
      detailNotes.value = []
    }
  }

  async function fetchMRDetail(pid: number, iid: number) {
    detailLoading.value = true
    try {
      // MR detail: use the issue detail endpoint pattern but for MRs
      // GitLab API: GET /projects/:id/merge_requests/:iid
      const res = await fetch(`/api/gitlab/projects/${pid}/merge-requests/${iid}/notes`)
      // For now just use the MR data from the list (already has all fields)
      // We don't have a dedicated MR detail endpoint yet, so skip
    } catch (e) {
      console.error('Failed to fetch MR detail:', e)
    } finally {
      detailLoading.value = false
    }
  }

  async function fetchMRNotes(pid: number, iid: number) {
    try {
      const res = await fetch(`/api/gitlab/projects/${pid}/merge-requests/${iid}/notes`)
      if (!res.ok) throw new Error(await res.text())
      detailNotes.value = await res.json() ?? []
    } catch (e) {
      console.error('Failed to fetch MR notes:', e)
      detailNotes.value = []
    }
  }

  async function selectItem(type: 'issue' | 'mr', projectPath: string, iid: number, projectId?: number) {
    selectedItem.value = { type, projectPath, iid }
    detailIssue.value = null
    detailMR.value = null
    detailNotes.value = []

    // Find project_id from the item data
    let pid = projectId
    if (!pid) {
      if (type === 'issue') {
        const issue = myIssues.value.find(i => i.iid === iid && i.project_path === projectPath)
        pid = issue?.project_id
      } else {
        const mr = myMRs.value.find(m => m.iid === iid && m.project_path === projectPath)
        pid = mr?.project_id
      }
    }
    if (!pid) return

    if (type === 'issue') {
      await Promise.all([
        fetchIssueDetail(pid, iid),
        fetchIssueNotes(pid, iid),
      ])
    } else {
      // For MRs, use the data from the list + fetch notes
      const mr = myMRs.value.find(m => m.iid === iid && m.project_path === projectPath)
      if (mr) detailMR.value = mr
      await fetchMRNotes(pid, iid)
    }
  }

  function closeDetail() {
    selectedItem.value = null
    detailIssue.value = null
    detailMR.value = null
    detailNotes.value = []
  }

  // Write operations — all use GitLab project ID directly
  async function createIssue(pid: number, body: {
    title: string
    description?: string
    labels?: string[]
    assignee_ids?: number[]
    milestone_id?: number
  }) {
    const res = await fetch(`/api/gitlab/projects/${pid}/issues`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchMyIssues()
    return res.json()
  }

  async function createMR(pid: number, body: {
    title: string
    description?: string
    source_branch: string
    target_branch: string
    assignee_id?: number
    reviewer_ids?: number[]
    draft?: boolean
    remove_source_branch?: boolean
  }) {
    const res = await fetch(`/api/gitlab/projects/${pid}/merge-requests`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchMyMRs()
    return res.json()
  }

  async function addComment(pid: number, type: 'issue' | 'mr', iid: number, body: string) {
    const endpoint = type === 'issue'
      ? `/api/gitlab/projects/${pid}/issues/${iid}/notes`
      : `/api/gitlab/projects/${pid}/merge-requests/${iid}/notes`
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body }),
    })
    if (!res.ok) throw new Error(await res.text())

    if (type === 'issue') {
      await fetchIssueNotes(pid, iid)
    } else {
      await fetchMRNotes(pid, iid)
    }
  }

  async function updateIssueDescription(pid: number, iid: number, description: string) {
    const res = await fetch(`/api/gitlab/projects/${pid}/issues/${iid}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ description }),
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchIssueDetail(pid, iid)
  }

  async function toggleCheckbox(pid: number, iid: number, checkboxIndex: number) {
    const issue = detailIssue.value
    if (!issue?.description) return

    const lines = issue.description.split('\n')
    let cbIdx = 0
    for (let i = 0; i < lines.length; i++) {
      const match = lines[i].match(/^(\s*[-*]\s+)\[([ xX])\](.*)$/)
      if (match) {
        if (cbIdx === checkboxIndex) {
          const checked = match[2] === ' ' ? 'x' : ' '
          lines[i] = `${match[1]}[${checked}]${match[3]}`
          break
        }
        cbIdx++
      }
    }

    await updateIssueDescription(pid, iid, lines.join('\n'))
  }

  async function updateIssueState(pid: number, iid: number, stateEvent: 'close' | 'reopen') {
    const res = await fetch(`/api/gitlab/projects/${pid}/issues/${iid}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ state_event: stateEvent }),
    })
    if (!res.ok) throw new Error(await res.text())
    await fetchIssueDetail(pid, iid)
    await fetchMyIssues()
  }

  async function updateMRState(pid: number, iid: number, stateEvent: 'close' | 'reopen') {
    // MR state update not yet implemented in direct endpoints
    console.warn('MR state update not yet available')
  }

  // Per-project (existing)
  async function fetchProjectGitLab() {
    projectLoading.value = true
    try {
      const res = await fetch(`${apiBase()}/gitlab/project`)
      if (res.ok) {
        project.value = await res.json()
        enabled.value = true
      } else {
        enabled.value = false
      }
    } catch {
      enabled.value = false
    } finally {
      projectLoading.value = false
    }
  }

  async function fetchProjectIssues() {
    try {
      const res = await fetch(`${apiBase()}/gitlab/issues`)
      if (res.ok) issues.value = await res.json() ?? []
    } catch { /* ignore */ }
  }

  async function fetchProjectMRs() {
    try {
      const res = await fetch(`${apiBase()}/gitlab/merge-requests`)
      if (res.ok) mergeRequests.value = await res.json() ?? []
    } catch { /* ignore */ }
  }

  async function fetchProjectPipelines() {
    try {
      const res = await fetch(`${apiBase()}/gitlab/pipelines`)
      if (res.ok) pipelines.value = await res.json() ?? []
    } catch { /* ignore */ }
  }

  // Lifecycle
  function startAutoRefresh() {
    stopAutoRefresh()
    refreshInterval = setInterval(() => {
      if (activeMainTab.value === 'tasks') fetchMyIssues()
      else if (activeMainTab.value === 'mrs') fetchMyMRs()
      else if (activeMainTab.value === 'pipelines') fetchProjectPipelines()
    }, 60_000)
  }

  function stopAutoRefresh() {
    if (refreshInterval) {
      clearInterval(refreshInterval)
      refreshInterval = null
    }
  }

  async function init() {
    await Promise.all([
      fetchMyIssues(),
      fetchMyMRs(),
      fetchLabels(),
      fetchMilestones(),
      fetchCurrentUser(),
    ])
    startAutoRefresh()
  }

  async function checkEnabled() {
    try {
      const res = await fetch('/api/gitlab/enabled')
      if (res.ok) {
        const data = await res.json()
        enabled.value = data.enabled === true
      } else {
        enabled.value = false
      }
    } catch {
      enabled.value = false
    }
  }

  // Helpers
  function extractProjectName(projectPath: string): string {
    const parts = projectPath.split('/')
    return parts[parts.length - 1]
  }

  function enrichProjectPath<T extends { web_url: string; project_path?: string }>(items: T[]): T[] {
    for (const item of items) {
      if (!item.project_path && item.web_url) {
        // https://gitlab.example.com/group/project/-/issues/42 → group/project
        const match = item.web_url.match(/^https?:\/\/[^/]+\/(.+?)\/-\//)
        item.project_path = match ? match[1] : 'unknown'
      }
    }
    return items
  }

  function setMyIssuesState(state: 'opened' | 'closed') {
    myIssuesState.value = state
    fetchMyIssues(state)
  }

  function setMyMRsState(state: 'opened' | 'merged' | 'closed') {
    myMRsState.value = state
    fetchMyMRs(state)
  }

  function reset() {
    myIssues.value = []
    myMRs.value = []
    selectedItem.value = null
    detailIssue.value = null
    detailNotes.value = []
    project.value = null
    issues.value = []
    mergeRequests.value = []
    pipelines.value = []
    clearFilters()
  }

  function clearFilters() {
    searchQuery.value = ''
    filterLabels.value = []
    filterMilestone.value = ''
    filterAssignee.value = ''
  }

  return {
    // Tab state
    activeMainTab,

    // My Issues
    myIssues,
    myIssuesState,
    myIssuesLoading,

    // My MRs
    myMRs,
    myMRsState,
    myMRsLoading,

    // Detail
    selectedItem,
    detailIssue,
    detailMR,
    detailNotes,
    detailLoading,

    // Filters
    labels,
    milestones,
    members,
    searchQuery,
    filterLabels,
    filterMilestone,
    filterAssignee,

    // Create modals
    showCreateIssue,
    showCreateMR,

    // Per-project
    enabled,
    project,
    issues,
    mergeRequests,
    pipelines,
    projectLoading,

    // Computed
    openIssuesCount,
    groupedMyIssues,
    groupedMyMRs,
    filteredMyIssues,
    filteredMyMRs,

    // Cross-project fetches
    fetchMyIssues,
    fetchMyMRs,
    fetchLabels,
    fetchMilestones,
    fetchCurrentUser,

    // Detail
    fetchIssueDetail,
    fetchIssueNotes,
    fetchMRDetail,
    fetchMRNotes,
    selectItem,
    closeDetail,

    // Write
    createIssue,
    createMR,
    addComment,
    updateIssueDescription,
    toggleCheckbox,
    updateIssueState,
    updateMRState,

    // Per-project
    fetchProjectGitLab,
    fetchProjectIssues,
    fetchProjectMRs,
    fetchProjectPipelines,

    // Lifecycle
    startAutoRefresh,
    stopAutoRefresh,
    init,

    // Helpers
    setMyIssuesState,
    setMyMRsState,
    clearFilters,
    reset,
    checkEnabled,
    extractProjectName,
  }
})
