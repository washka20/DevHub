import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitlabApi } from '../api/gitlab'
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
  const toast = useToast()

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
  const selectedItem = ref<{ type: 'issue' | 'mr'; projectPath: string; iid: number; projectId: number } | null>(null)
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
      const data = await gitlabApi.myIssues(s)
      myIssues.value = enrichProjectPath(data ?? [])
    } catch (e) {
      toast.show('error', `Failed to fetch issues: ${getErrorMessage(e)}`)
      myIssues.value = []
    } finally {
      myIssuesLoading.value = false
    }
  }

  async function fetchMyMRs(state?: string) {
    myMRsLoading.value = true
    try {
      const s = state || myMRsState.value
      const data = await gitlabApi.myMergeRequests(s)
      myMRs.value = enrichProjectPath(data ?? [])
    } catch (e) {
      toast.show('error', `Failed to fetch MRs: ${getErrorMessage(e)}`)
      myMRs.value = []
    } finally {
      myMRsLoading.value = false
    }
  }

  async function fetchLabels() {
    try {
      labels.value = await gitlabApi.labels() ?? []
    } catch { /* ignore */ }
  }

  async function fetchMilestones() {
    try {
      milestones.value = await gitlabApi.milestones() ?? []
    } catch { /* ignore */ }
  }

  async function fetchCurrentUser() {
    try {
      const user = await gitlabApi.currentUser()
      if (user) {
        members.value = [user, ...members.value.filter(m => m.id !== user.id)]
      }
    } catch { /* ignore */ }
  }

  // Detail
  async function fetchIssueDetail(pid: number, iid: number) {
    detailLoading.value = true
    try {
      detailIssue.value = await gitlabApi.issueDetail(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to fetch issue: ${getErrorMessage(e)}`)
    } finally {
      detailLoading.value = false
    }
  }

  async function fetchIssueNotes(pid: number, iid: number) {
    try {
      detailNotes.value = await gitlabApi.issueNotes(pid, iid) ?? []
    } catch (e) {
      toast.show('error', `Failed to fetch issue notes: ${getErrorMessage(e)}`)
      detailNotes.value = []
    }
  }

  async function fetchMRDetail(pid: number, iid: number) {
    detailLoading.value = true
    try {
      await gitlabApi.mrNotes(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to fetch MR: ${getErrorMessage(e)}`)
    } finally {
      detailLoading.value = false
    }
  }

  async function fetchMRNotes(pid: number, iid: number) {
    try {
      detailNotes.value = await gitlabApi.mrNotes(pid, iid) ?? []
    } catch (e) {
      toast.show('error', `Failed to fetch MR notes: ${getErrorMessage(e)}`)
      detailNotes.value = []
    }
  }

  // Unique GitLab projects derived from user's issues and MRs.
  const availableProjects = computed(() => {
    const map = new Map<number, string>()
    for (const issue of myIssues.value) {
      if (issue.project_id && issue.project_path) {
        map.set(issue.project_id, issue.project_path)
      }
    }
    for (const mr of myMRs.value) {
      if (mr.project_id && mr.project_path) {
        map.set(mr.project_id, mr.project_path)
      }
    }
    return Array.from(map, ([id, path]) => ({ id, path }))
  })

  function findProjectId(type: 'issue' | 'mr', projectPath: string, iid: number): number | undefined {
    if (type === 'issue') {
      return myIssues.value.find(i => i.iid === iid && i.project_path === projectPath)?.project_id
    }
    return myMRs.value.find(m => m.iid === iid && m.project_path === projectPath)?.project_id
  }

  async function selectItem(type: 'issue' | 'mr', projectPath: string, iid: number, projectId?: number) {
    const pid = projectId ?? findProjectId(type, projectPath, iid)
    if (!pid) return

    selectedItem.value = { type, projectPath, iid, projectId: pid }
    detailIssue.value = null
    detailMR.value = null
    detailNotes.value = []

    if (type === 'issue') {
      await Promise.all([
        fetchIssueDetail(pid, iid),
        fetchIssueNotes(pid, iid),
      ])
    } else {
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

  // Write operations
  async function createIssue(pid: number, body: {
    title: string
    description?: string
    labels?: string[]
    assignee_ids?: number[]
    milestone_id?: number
  }) {
    const result = await gitlabApi.createIssue(pid, body)
    await fetchMyIssues()
    return result
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
    const result = await gitlabApi.createMR(pid, body)
    await fetchMyMRs()
    return result
  }

  async function addComment(pid: number, type: 'issue' | 'mr', iid: number, body: string) {
    if (type === 'issue') {
      await gitlabApi.addIssueNote(pid, iid, body)
      await fetchIssueNotes(pid, iid)
    } else {
      await gitlabApi.addMRNote(pid, iid, body)
      await fetchMRNotes(pid, iid)
    }
  }

  async function updateIssueDescription(pid: number, iid: number, description: string) {
    await gitlabApi.updateIssue(pid, iid, { description })
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
    await gitlabApi.updateIssue(pid, iid, { state_event: stateEvent })
    await fetchIssueDetail(pid, iid)
    await fetchMyIssues()
  }

  async function updateMRState(pid: number, iid: number, stateEvent: 'close' | 'reopen') {
    toast.show('error', 'MR state update not yet available')
  }

  // Per-project (existing)
  async function fetchProjectGitLab() {
    projectLoading.value = true
    try {
      project.value = await gitlabApi.project(apiBase())
      enabled.value = true
    } catch {
      enabled.value = false
    } finally {
      projectLoading.value = false
    }
  }

  async function fetchProjectIssues() {
    try {
      issues.value = await gitlabApi.projectIssues(apiBase()) ?? []
    } catch { /* ignore */ }
  }

  async function fetchProjectMRs() {
    try {
      mergeRequests.value = await gitlabApi.projectMRs(apiBase()) ?? []
    } catch { /* ignore */ }
  }

  async function fetchProjectPipelines() {
    try {
      pipelines.value = await gitlabApi.projectPipelines(apiBase()) ?? []
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
      const data = await gitlabApi.checkEnabled()
      enabled.value = data.enabled === true
    } catch {
      enabled.value = false
    }
  }

  // Helpers
  function enrichProjectPath<T extends { web_url: string; project_path?: string }>(items: T[]): T[] {
    for (const item of items) {
      if (!item.project_path && item.web_url) {
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
    availableProjects,

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
  }
})
