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
  GitLabDiscussion,
  GitLabLabel,
  GitLabMilestone,
  GitLabMember,
  GitLabTodo,
  GitLabMRApproval,
  GitLabJob,
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
  const activeMainTab = ref<'inbox' | 'tasks' | 'mrs' | 'reviews' | 'project' | 'pipelines'>('inbox')

  // Cross-project: Todos
  const todos = ref<GitLabTodo[]>([])
  const todosLoading = ref(false)
  const todosCount = computed(() => todos.value.filter(t => t.state === 'pending').length)

  // Cross-project: My Issues
  const myIssues = ref<GitLabIssue[]>([])
  const myIssuesState = ref<'opened' | 'closed'>('opened')
  const myIssuesLoading = ref(false)

  // Cross-project: My MRs
  const myMRs = ref<GitLabMR[]>([])
  const myMRsState = ref<'opened' | 'merged' | 'closed'>('opened')
  const myMRsLoading = ref(false)

  // Cross-project: Review MRs
  const reviewMRs = ref<GitLabMR[]>([])
  const reviewMRsState = ref<'opened' | 'merged' | 'closed'>('opened')
  const reviewMRsLoading = ref(false)

  // Detail panel
  const selectedItem = ref<{ type: 'issue' | 'mr'; projectPath: string; iid: number; projectId: number } | null>(null)
  const detailIssue = ref<GitLabIssue | null>(null)
  const detailMR = ref<GitLabMR | null>(null)
  const detailNotes = ref<GitLabNote[]>([])
  const detailLoading = ref(false)
  const mrApprovals = ref<GitLabMRApproval | null>(null)
  const detailDiscussions = ref<GitLabDiscussion[]>([])

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

  // Pipeline jobs
  const pipelineJobs = ref<Record<number, GitLabJob[]>>({})
  const selectedJobTrace = ref<string | null>(null)
  const selectedJobId = ref<number | null>(null)
  const expandedPipelineId = ref<number | null>(null)

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

  const filteredReviewMRs = computed(() => {
    let result = reviewMRs.value
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

  const groupedReviewMRs = computed(() => {
    const groups: Record<string, GitLabMR[]> = {}
    const filtered = filteredReviewMRs.value
    for (const mr of filtered) {
      const key = mr.project_path || 'unknown'
      if (!groups[key]) groups[key] = []
      groups[key].push(mr)
    }
    return groups
  })

  // Cross-project fetches
  async function fetchTodos() {
    todosLoading.value = true
    try {
      todos.value = await gitlabApi.myTodos() ?? []
    } catch (e) {
      toast.show('error', `Failed to fetch todos: ${getErrorMessage(e)}`)
      todos.value = []
    } finally {
      todosLoading.value = false
    }
  }

  async function markTodoDone(id: number) {
    todos.value = todos.value.filter(t => t.id !== id)
    try {
      await gitlabApi.markTodoDone(id)
    } catch (e) {
      toast.show('error', `Failed to mark todo: ${getErrorMessage(e)}`)
      await fetchTodos()
    }
  }

  async function markAllTodosDone() {
    todos.value = []
    try {
      await gitlabApi.markAllTodosDone()
    } catch (e) {
      toast.show('error', `Failed to mark all todos: ${getErrorMessage(e)}`)
      await fetchTodos()
    }
  }

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

  async function fetchReviewMRs(state?: string) {
    reviewMRsLoading.value = true
    try {
      const s = state || reviewMRsState.value
      const data = await gitlabApi.myReviewMRs(s)
      reviewMRs.value = enrichProjectPath(data ?? [])
    } catch (e) {
      toast.show('error', `Failed to fetch review MRs: ${getErrorMessage(e)}`)
      reviewMRs.value = []
    } finally {
      reviewMRsLoading.value = false
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

  // Detail — merge list item (has label_details) with detail (has description)
  async function fetchIssueDetail(pid: number, iid: number) {
    detailLoading.value = true
    try {
      const detail = await gitlabApi.issueDetail(pid, iid)
      const listItem = myIssues.value.find(i => i.project_id === pid && i.iid === iid)
      detailIssue.value = { ...listItem, ...detail, label_details: listItem?.label_details ?? detail.label_details }
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

  async function fetchMRApprovals(pid: number, iid: number) {
    try {
      mrApprovals.value = await gitlabApi.mrApprovals(pid, iid)
    } catch {
      mrApprovals.value = null
    }
  }

  async function approveMR(pid: number, iid: number) {
    try {
      await gitlabApi.approveMR(pid, iid)
      await fetchMRApprovals(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to approve MR: ${getErrorMessage(e)}`)
    }
  }

  async function unapproveMR(pid: number, iid: number) {
    try {
      await gitlabApi.unapproveMR(pid, iid)
      await fetchMRApprovals(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to unapprove MR: ${getErrorMessage(e)}`)
    }
  }

  // Discussions
  const resolvedThreadsCount = computed(() =>
    detailDiscussions.value.filter(d =>
      !d.individual_note && d.notes.some(n => n.resolvable) && d.notes.every(n => !n.resolvable || n.resolved)
    ).length
  )

  const totalThreadsCount = computed(() =>
    detailDiscussions.value.filter(d =>
      !d.individual_note && d.notes.some(n => n.resolvable)
    ).length
  )

  async function fetchMRDiscussions(pid: number, iid: number) {
    try {
      detailDiscussions.value = await gitlabApi.mrDiscussions(pid, iid) ?? []
    } catch (e) {
      toast.show('error', `Failed to fetch discussions: ${getErrorMessage(e)}`)
      detailDiscussions.value = []
    }
  }

  async function resolveDiscussion(pid: number, iid: number, discussionId: string, resolved: boolean) {
    try {
      await gitlabApi.resolveDiscussion(pid, iid, discussionId, resolved)
      await fetchMRDiscussions(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to resolve discussion: ${getErrorMessage(e)}`)
    }
  }

  async function replyToDiscussion(pid: number, iid: number, discussionId: string, body: string) {
    try {
      await gitlabApi.replyToDiscussion(pid, iid, discussionId, body)
      await fetchMRDiscussions(pid, iid)
    } catch (e) {
      toast.show('error', `Failed to reply to discussion: ${getErrorMessage(e)}`)
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
    for (const mr of reviewMRs.value) {
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
      ?? reviewMRs.value.find(m => m.iid === iid && m.project_path === projectPath)?.project_id
  }

  async function selectItem(type: 'issue' | 'mr', projectPath: string, iid: number, projectId?: number) {
    const pid = projectId ?? findProjectId(type, projectPath, iid)
    if (!pid) return

    selectedItem.value = { type, projectPath, iid, projectId: pid }
    detailIssue.value = null
    detailMR.value = null
    detailNotes.value = []
    detailDiscussions.value = []
    mrApprovals.value = null

    if (type === 'issue') {
      await Promise.all([
        fetchIssueDetail(pid, iid),
        fetchIssueNotes(pid, iid),
      ])
    } else {
      const mr = myMRs.value.find(m => m.iid === iid && m.project_path === projectPath)
        ?? reviewMRs.value.find(m => m.iid === iid && m.project_path === projectPath)
      if (mr) detailMR.value = mr
      await Promise.all([
        fetchMRDiscussions(pid, iid),
        fetchMRApprovals(pid, iid),
      ])
    }
  }

  function closeDetail() {
    selectedItem.value = null
    detailIssue.value = null
    detailMR.value = null
    detailNotes.value = []
    detailDiscussions.value = []
    mrApprovals.value = null
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

  // Pipeline jobs
  async function fetchPipelineJobs(pipelineId: number) {
    if (!project.value) return
    try {
      const jobs = await gitlabApi.pipelineJobs(project.value.id, pipelineId)
      pipelineJobs.value = { ...pipelineJobs.value, [pipelineId]: jobs ?? [] }
    } catch (e) {
      toast.show('error', `Failed to fetch jobs: ${getErrorMessage(e)}`)
    }
  }

  async function fetchJobTrace(jobId: number) {
    if (!project.value) return
    selectedJobId.value = jobId
    selectedJobTrace.value = null
    try {
      selectedJobTrace.value = await gitlabApi.jobTrace(project.value.id, jobId)
    } catch (e) {
      toast.show('error', `Failed to fetch job trace: ${getErrorMessage(e)}`)
      selectedJobTrace.value = null
    }
  }

  async function retryJob(jobId: number, pipelineId: number) {
    if (!project.value) return
    try {
      await gitlabApi.retryJob(project.value.id, jobId)
      toast.show('success', 'Job retry started')
      await fetchPipelineJobs(pipelineId)
    } catch (e) {
      toast.show('error', `Failed to retry job: ${getErrorMessage(e)}`)
    }
  }

  async function cancelJob(jobId: number, pipelineId: number) {
    if (!project.value) return
    try {
      await gitlabApi.cancelJob(project.value.id, jobId)
      toast.show('success', 'Job canceled')
      await fetchPipelineJobs(pipelineId)
    } catch (e) {
      toast.show('error', `Failed to cancel job: ${getErrorMessage(e)}`)
    }
  }

  function togglePipeline(pipelineId: number) {
    if (expandedPipelineId.value === pipelineId) {
      expandedPipelineId.value = null
      selectedJobId.value = null
      selectedJobTrace.value = null
    } else {
      expandedPipelineId.value = pipelineId
      selectedJobId.value = null
      selectedJobTrace.value = null
      if (!pipelineJobs.value[pipelineId]) {
        fetchPipelineJobs(pipelineId)
      }
    }
  }

  function closeJobTrace() {
    selectedJobId.value = null
    selectedJobTrace.value = null
  }

  // Lifecycle
  function startAutoRefresh() {
    stopAutoRefresh()
    refreshInterval = setInterval(() => {
      if (activeMainTab.value === 'inbox') fetchTodos()
      else if (activeMainTab.value === 'tasks') fetchMyIssues()
      else if (activeMainTab.value === 'mrs') fetchMyMRs()
      else if (activeMainTab.value === 'reviews') fetchReviewMRs()
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
    await checkEnabled()
    if (!enabled.value) return

    await Promise.all([
      fetchTodos(),
      fetchMyIssues(),
      fetchMyMRs(),
      fetchReviewMRs(),
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

  function setReviewMRsState(state: 'opened' | 'merged' | 'closed') {
    reviewMRsState.value = state
    fetchReviewMRs(state)
  }

  function reset() {
    todos.value = []
    myIssues.value = []
    myMRs.value = []
    reviewMRs.value = []
    selectedItem.value = null
    detailIssue.value = null
    detailNotes.value = []
    detailDiscussions.value = []
    project.value = null
    issues.value = []
    mergeRequests.value = []
    pipelines.value = []
    pipelineJobs.value = {}
    selectedJobTrace.value = null
    selectedJobId.value = null
    expandedPipelineId.value = null
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

    // Todos
    todos,
    todosLoading,
    todosCount,

    // My Issues
    myIssues,
    myIssuesState,
    myIssuesLoading,

    // My MRs
    myMRs,
    myMRsState,
    myMRsLoading,

    // Review MRs
    reviewMRs,
    reviewMRsState,
    reviewMRsLoading,

    // Detail
    selectedItem,
    detailIssue,
    detailMR,
    detailNotes,
    detailDiscussions,
    detailLoading,

    // Discussions
    resolvedThreadsCount,
    totalThreadsCount,
    fetchMRDiscussions,
    resolveDiscussion,
    replyToDiscussion,

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
    groupedReviewMRs,
    filteredMyIssues,
    filteredMyMRs,
    filteredReviewMRs,
    availableProjects,

    // Todos
    fetchTodos,
    markTodoDone,
    markAllTodosDone,

    // Cross-project fetches
    fetchMyIssues,
    fetchMyMRs,
    fetchReviewMRs,
    fetchLabels,
    fetchMilestones,
    fetchCurrentUser,

    // MR Approvals
    mrApprovals,
    fetchMRApprovals,
    approveMR,
    unapproveMR,

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

    // Pipeline jobs
    pipelineJobs,
    selectedJobTrace,
    selectedJobId,
    expandedPipelineId,
    fetchPipelineJobs,
    fetchJobTrace,
    retryJob,
    cancelJob,
    togglePipeline,
    closeJobTrace,

    // Lifecycle
    startAutoRefresh,
    stopAutoRefresh,
    init,

    // Helpers
    setMyIssuesState,
    setMyMRsState,
    setReviewMRsState,
    clearFilters,
    reset,
    checkEnabled,
  }
})
