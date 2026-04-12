<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref } from 'vue'
import { useGitLabStore } from '../stores/gitlab'
import { useGitStore } from '../stores/git'
import { useToast } from '../composables/useToast'
import { formatRelativeTime, isOverdue } from '../utils/date'
import { hexToRgb } from '../utils/color'
import { getErrorMessage } from '../utils/error'
import GitLabDetailModal from '../components/GitLabDetailModal.vue'
import GitLabCreateIssue from '../components/GitLabCreateIssue.vue'
import GitLabCreateMR from '../components/GitLabCreateMR.vue'
import type { GitLabIssue, GitLabMR } from '../types'

const store = useGitLabStore()
const gitStore = useGitStore()
const toast = useToast()

const collapsedGroups = ref<Set<string>>(new Set())

const gitlabProjects = computed(() => store.availableProjects)

const currentBranch = computed(() => gitStore.status.branch || 'main')

const isRefreshing = computed(() =>
  store.myIssuesLoading || store.myMRsLoading || store.reviewMRsLoading
)

const detailItem = computed(() => {
  if (!store.selectedItem) return null
  if (store.selectedItem.type === 'issue') return store.detailIssue
  return store.detailMR
})

function toggleGroup(key: string) {
  const s = new Set(collapsedGroups.value)
  if (s.has(key)) s.delete(key)
  else s.add(key)
  collapsedGroups.value = s
}

function isGroupCollapsed(key: string): boolean {
  return collapsedGroups.value.has(key)
}

const formatTimeAgo = formatRelativeTime

// Build label color map from label_details across all issues and MRs
const labelColorMap = computed(() => {
  const map = new Map<string, string>()
  for (const issue of store.myIssues) {
    for (const ld of issue.label_details || []) {
      if (ld.color) map.set(ld.name, ld.color)
    }
  }
  for (const mr of store.myMRs) {
    for (const ld of mr.label_details || []) {
      if (ld.color) map.set(ld.name, ld.color)
    }
  }
  for (const mr of store.reviewMRs) {
    for (const ld of mr.label_details || []) {
      if (ld.color) map.set(ld.name, ld.color)
    }
  }
  for (const l of store.labels) {
    if (l.color) map.set(l.name, l.color)
  }
  return map
})

function labelStyle(label: string): Record<string, string> {
  const color = labelColorMap.value.get(label)
  if (color) {
    const rgb = hexToRgb(color)
    return {
      background: `rgba(${rgb},0.1)`,
      color: color,
      borderColor: `rgba(${rgb},0.25)`,
    }
  }
  return { background: 'rgba(139,148,158,0.1)', color: '#8b949e', borderColor: 'rgba(139,148,158,0.25)' }
}

function ciStatusClass(pipeline: GitLabMR['pipeline']): string {
  if (!pipeline) return 'ci-unknown'
  switch (pipeline.status) {
    case 'success': return 'ci-passed'
    case 'failed': return 'ci-failed'
    case 'running':
    case 'pending':
    case 'created':
      return 'ci-pending'
    case 'canceled': return 'ci-canceled'
    default: return 'ci-unknown'
  }
}

function ciStatusText(pipeline: GitLabMR['pipeline']): string {
  if (!pipeline) return 'no pipeline'
  return pipeline.status
}

async function handleRefresh() {
  if (store.activeMainTab === 'tasks') {
    await store.fetchMyIssues()
  } else if (store.activeMainTab === 'mrs') {
    await store.fetchMyMRs()
  } else if (store.activeMainTab === 'reviews') {
    await store.fetchReviewMRs()
  } else if (store.activeMainTab === 'project') {
    await Promise.all([store.fetchProjectIssues(), store.fetchProjectMRs()])
  } else {
    await store.fetchProjectPipelines()
  }
}

function selectIssue(issue: GitLabIssue) {
  store.selectItem('issue', issue.project_path, issue.iid, issue.project_id)
}

function selectMR(mr: GitLabMR) {
  store.selectItem('mr', mr.project_path, mr.iid, mr.project_id)
}

async function handleAddComment(body: string) {
  if (!store.selectedItem) return
  try {
    await store.addComment(
      store.selectedItem.projectId,
      store.selectedItem.type,
      store.selectedItem.iid,
      body,
    )
    toast.show('success', 'Comment added')
  } catch (e) {
    toast.show('error', `Failed to add comment: ${getErrorMessage(e)}`)
  }
}

async function handleToggleCheckbox(index: number) {
  if (!store.selectedItem || store.selectedItem.type !== 'issue') return
  try {
    await store.toggleCheckbox(store.selectedItem.projectId, store.selectedItem.iid, index)
  } catch (e) {
    toast.show('error', `Failed to toggle checkbox: ${getErrorMessage(e)}`)
  }
}

async function handleUpdateState(stateEvent: 'close' | 'reopen') {
  if (!store.selectedItem) return
  try {
    if (store.selectedItem.type === 'issue') {
      await store.updateIssueState(store.selectedItem.projectId, store.selectedItem.iid, stateEvent)
    } else {
      await store.updateMRState(store.selectedItem.projectId, store.selectedItem.iid, stateEvent)
    }
    toast.show('success', `${store.selectedItem.type === 'issue' ? 'Issue' : 'MR'} ${stateEvent === 'close' ? 'closed' : 'reopened'}`)
  } catch (e) {
    toast.show('error', `Failed: ${getErrorMessage(e)}`)
  }
}

async function handleCreateIssue(data: {
  projectId: number
  title: string
  description: string
  labels: string[]
  assignee_ids: number[]
  milestone_id: number | undefined
}) {
  try {
    await store.createIssue(data.projectId, {
      title: data.title,
      description: data.description,
      labels: data.labels,
      assignee_ids: data.assignee_ids,
      milestone_id: data.milestone_id,
    })
    store.showCreateIssue = false
    toast.show('success', 'Issue created')
  } catch (e) {
    toast.show('error', `Failed to create issue: ${getErrorMessage(e)}`)
  }
}

async function handleCreateMR(data: {
  projectId: number
  title: string
  description: string
  source_branch: string
  target_branch: string
  assignee_ids: number[]
  reviewer_ids: number[]
  draft: boolean
  remove_source_branch: boolean
}) {
  try {
    await store.createMR(data.projectId, {
      title: data.title,
      description: data.description,
      source_branch: data.source_branch,
      target_branch: data.target_branch,
      assignee_ids: data.assignee_ids,
      reviewer_ids: data.reviewer_ids,
      draft: data.draft,
      remove_source_branch: data.remove_source_branch,
    })
    store.showCreateMR = false
    toast.show('success', 'Merge request created')
  } catch (e) {
    toast.show('error', `Failed to create MR: ${getErrorMessage(e)}`)
  }
}

onMounted(() => {
  store.init()
})

onUnmounted(() => {
  store.stopAutoRefresh()
})
</script>

<template>
  <div class="gitlab-view">
    <!-- Header — always visible -->
    <header class="page-header">
      <div class="header-row">
        <div class="header-title">
          <h1>GitLab</h1>
          <span v-if="store.enabled && isRefreshing" class="refresh-indicator">
            <svg class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
            </svg>
          </span>
        </div>
        <div v-if="store.enabled" class="header-actions">
          <button class="btn btn-green" @click="store.showCreateIssue = true">
            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
              <path d="M8 2a.75.75 0 01.75.75v4.5h4.5a.75.75 0 010 1.5h-4.5v4.5a.75.75 0 01-1.5 0v-4.5h-4.5a.75.75 0 010-1.5h4.5v-4.5A.75.75 0 018 2z"/>
            </svg>
            Issue
          </button>
          <button class="btn btn-blue" @click="store.showCreateMR = true">
            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
              <path d="M8 2a.75.75 0 01.75.75v4.5h4.5a.75.75 0 010 1.5h-4.5v4.5a.75.75 0 01-1.5 0v-4.5h-4.5a.75.75 0 010-1.5h4.5v-4.5A.75.75 0 018 2z"/>
            </svg>
            MR
          </button>
          <button class="btn" @click="handleRefresh" :disabled="isRefreshing">
            Refresh
          </button>
        </div>
      </div>
    </header>

    <!-- Not configured -->
    <div v-if="store.enabled === false" class="not-configured">
      <div class="not-configured-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 4.82 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0 1 18.6 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.51L23 13.45a.84.84 0 0 1-.35.94z"/>
        </svg>
      </div>
      <h2>GitLab not connected</h2>
      <p>Set <code>DEVHUB_GITLAB_URL</code> and <code>DEVHUB_GITLAB_TOKEN</code> in your <code>.env</code> file to enable GitLab integration.</p>
      <p class="not-configured-hint">Token needs <code>api</code> scope for full functionality.</p>
    </div>

    <!-- Loading -->
    <div v-else-if="store.enabled === null" class="loading-check">
      <svg class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
      </svg>
      Checking GitLab connection...
    </div>

    <template v-else>
    <!-- Main tabs -->
    <nav class="main-tabs">
      <button
        class="tab-btn"
        :class="{ active: store.activeMainTab === 'tasks' }"
        @click="store.activeMainTab = 'tasks'"
      >
        My Tasks
        <span v-if="store.openIssuesCount > 0" class="tab-badge">{{ store.openIssuesCount }}</span>
      </button>
      <button
        class="tab-btn"
        :class="{ active: store.activeMainTab === 'mrs' }"
        @click="store.activeMainTab = 'mrs'"
      >
        My MRs
        <span v-if="store.myMRs.length > 0" class="tab-badge">{{ store.myMRs.length }}</span>
      </button>
      <button
        class="tab-btn"
        :class="{ active: store.activeMainTab === 'reviews' }"
        @click="store.activeMainTab = 'reviews'"
      >
        My Reviews
        <span v-if="store.reviewMRs.length > 0" class="tab-badge">{{ store.reviewMRs.length }}</span>
      </button>
      <button
        class="tab-btn"
        :class="{ active: store.activeMainTab === 'project' }"
        @click="store.activeMainTab = 'project'; store.fetchProjectIssues(); store.fetchProjectMRs()"
      >
        Project
      </button>
      <button
        class="tab-btn"
        :class="{ active: store.activeMainTab === 'pipelines' }"
        @click="store.activeMainTab = 'pipelines'; store.fetchProjectPipelines()"
      >
        Pipelines
      </button>
    </nav>

    <!-- Content area -->
    <div class="content-area">
      <div class="main-content">

        <!-- MY TASKS TAB -->
        <template v-if="store.activeMainTab === 'tasks'">
          <!-- Sub-tabs -->
          <div class="sub-tabs">
            <button
              class="sub-tab"
              :class="{ active: store.myIssuesState === 'opened' }"
              @click="store.setMyIssuesState('opened')"
            >Active</button>
            <button
              class="sub-tab"
              :class="{ active: store.myIssuesState === 'closed' }"
              @click="store.setMyIssuesState('closed')"
            >Closed</button>
          </div>

          <!-- Filter bar -->
          <div class="filter-bar">
            <input
              v-model="store.searchQuery"
              class="filter-search"
              type="text"
              placeholder="Search issues..."
            />
            <select v-model="store.filterMilestone" class="filter-select">
              <option value="">All milestones</option>
              <option v-for="ms in store.milestones" :key="ms.id" :value="ms.title">
                {{ ms.title }}
              </option>
            </select>
            <select v-model="store.filterAssignee" class="filter-select">
              <option value="">All assignees</option>
              <option v-for="m in store.members" :key="m.id" :value="m.username">
                {{ m.name }}
              </option>
            </select>
            <button
              v-if="store.searchQuery || store.filterMilestone || store.filterAssignee || store.filterLabels.length"
              class="filter-clear"
              @click="store.clearFilters()"
            >Clear</button>
          </div>

          <!-- Loading -->
          <div v-if="store.myIssuesLoading && !store.myIssues.length" class="empty">
            Loading issues...
          </div>

          <!-- Empty -->
          <div v-else-if="Object.keys(store.groupedMyIssues).length === 0" class="empty">
            No issues found
          </div>

          <!-- Grouped issue list -->
          <div v-else class="grouped-list">
            <div
              v-for="(issues, projectPath) in store.groupedMyIssues"
              :key="projectPath"
              class="group"
            >
              <div class="group-header" @click="toggleGroup(projectPath)">
                <svg
                  class="group-chevron"
                  :class="{ collapsed: isGroupCollapsed(projectPath) }"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M4.427 7.427l3.396 3.396a.25.25 0 00.354 0l3.396-3.396A.25.25 0 0011.396 7H4.604a.25.25 0 00-.177.427z"/>
                </svg>
                <span class="group-name">{{ projectPath }}</span>
                <span class="group-count">{{ issues.length }}</span>
              </div>

              <div v-if="!isGroupCollapsed(projectPath)" class="group-items">
                <div
                  v-for="issue in issues"
                  :key="issue.id"
                  class="item-row"
                  :class="{ 'row-selected': store.selectedItem?.type === 'issue' && store.selectedItem?.iid === issue.iid }"
                  @click="selectIssue(issue)"
                >
                  <span class="item-iid">#{{ issue.iid }}</span>
                  <span class="item-title">{{ issue.title }}</span>
                  <span class="item-labels">
                    <span
                      v-for="label in (issue.labels || []).slice(0, 3)"
                      :key="label"
                      class="label-badge"
                      :style="labelStyle(label)"
                    >{{ label }}</span>
                  </span>
                  <span
                    v-if="issue.due_date"
                    class="item-due"
                    :class="{ overdue: isOverdue(issue.due_date) }"
                  >{{ issue.due_date }}</span>
                  <span v-if="issue.assignees?.length" class="item-assignee">
                    {{ issue.assignees[0].username }}
                  </span>
                  <span class="item-time">{{ formatTimeAgo(issue.updated_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- MY MRs TAB -->
        <template v-if="store.activeMainTab === 'mrs'">
          <div class="sub-tabs">
            <button
              class="sub-tab"
              :class="{ active: store.myMRsState === 'opened' }"
              @click="store.setMyMRsState('opened')"
            >Active</button>
            <button
              class="sub-tab"
              :class="{ active: store.myMRsState === 'merged' }"
              @click="store.setMyMRsState('merged')"
            >Merged</button>
            <button
              class="sub-tab"
              :class="{ active: store.myMRsState === 'closed' }"
              @click="store.setMyMRsState('closed')"
            >Closed</button>
          </div>

          <!-- Filter bar -->
          <div class="filter-bar">
            <input
              v-model="store.searchQuery"
              class="filter-search"
              type="text"
              placeholder="Search merge requests..."
            />
            <button
              v-if="store.searchQuery || store.filterLabels.length"
              class="filter-clear"
              @click="store.clearFilters()"
            >Clear</button>
          </div>

          <div v-if="store.myMRsLoading && !store.myMRs.length" class="empty">
            Loading merge requests...
          </div>

          <div v-else-if="Object.keys(store.groupedMyMRs).length === 0" class="empty">
            No merge requests found
          </div>

          <div v-else class="grouped-list">
            <div
              v-for="(mrs, projectPath) in store.groupedMyMRs"
              :key="projectPath"
              class="group"
            >
              <div class="group-header" @click="toggleGroup(projectPath)">
                <svg
                  class="group-chevron"
                  :class="{ collapsed: isGroupCollapsed(projectPath) }"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M4.427 7.427l3.396 3.396a.25.25 0 00.354 0l3.396-3.396A.25.25 0 0011.396 7H4.604a.25.25 0 00-.177.427z"/>
                </svg>
                <span class="group-name">{{ projectPath }}</span>
                <span class="group-count">{{ mrs.length }}</span>
              </div>

              <div v-if="!isGroupCollapsed(projectPath)" class="group-items">
                <div
                  v-for="mr in mrs"
                  :key="mr.id"
                  class="item-row mr-row"
                  :class="{ 'row-selected': store.selectedItem?.type === 'mr' && store.selectedItem?.iid === mr.iid }"
                  @click="selectMR(mr)"
                >
                  <span class="item-iid">!{{ mr.iid }}</span>
                  <span class="item-title">
                    <span v-if="mr.draft" class="draft-badge">Draft</span>
                    {{ mr.title }}
                  </span>
                  <span class="mr-branches">
                    <code>{{ mr.source_branch }}</code>
                    <svg viewBox="0 0 16 16" fill="currentColor" width="10" height="10">
                      <path d="M8.22 2.97a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06l2.97-2.97H3.75a.75.75 0 010-1.5h7.44L8.22 4.03a.75.75 0 010-1.06z"/>
                    </svg>
                    <code>{{ mr.target_branch }}</code>
                  </span>
                  <span class="ci-dot" :class="ciStatusClass(mr.pipeline)" :title="ciStatusText(mr.pipeline)"></span>
                  <span class="item-labels">
                    <span
                      v-for="label in (mr.labels || []).slice(0, 2)"
                      :key="label"
                      class="label-badge"
                      :style="labelStyle(label)"
                    >{{ label }}</span>
                  </span>
                  <span class="item-time">{{ formatTimeAgo(mr.updated_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- MY REVIEWS TAB -->
        <template v-if="store.activeMainTab === 'reviews'">
          <div class="sub-tabs">
            <button
              class="sub-tab"
              :class="{ active: store.reviewMRsState === 'opened' }"
              @click="store.setReviewMRsState('opened')"
            >Active</button>
            <button
              class="sub-tab"
              :class="{ active: store.reviewMRsState === 'merged' }"
              @click="store.setReviewMRsState('merged')"
            >Merged</button>
            <button
              class="sub-tab"
              :class="{ active: store.reviewMRsState === 'closed' }"
              @click="store.setReviewMRsState('closed')"
            >Closed</button>
          </div>

          <!-- Filter bar -->
          <div class="filter-bar">
            <input
              v-model="store.searchQuery"
              class="filter-search"
              type="text"
              placeholder="Search review requests..."
            />
            <button
              v-if="store.searchQuery || store.filterLabels.length"
              class="filter-clear"
              @click="store.clearFilters()"
            >Clear</button>
          </div>

          <div v-if="store.reviewMRsLoading && !store.reviewMRs.length" class="empty">
            Loading review requests...
          </div>

          <div v-else-if="Object.keys(store.groupedReviewMRs).length === 0" class="empty">
            No review requests found
          </div>

          <div v-else class="grouped-list">
            <div
              v-for="(mrs, projectPath) in store.groupedReviewMRs"
              :key="projectPath"
              class="group"
            >
              <div class="group-header" @click="toggleGroup(projectPath)">
                <svg
                  class="group-chevron"
                  :class="{ collapsed: isGroupCollapsed(projectPath) }"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M4.427 7.427l3.396 3.396a.25.25 0 00.354 0l3.396-3.396A.25.25 0 0011.396 7H4.604a.25.25 0 00-.177.427z"/>
                </svg>
                <span class="group-name">{{ projectPath }}</span>
                <span class="group-count">{{ mrs.length }}</span>
              </div>

              <div v-if="!isGroupCollapsed(projectPath)" class="group-items">
                <div
                  v-for="mr in mrs"
                  :key="mr.id"
                  class="item-row mr-row"
                  :class="{ 'row-selected': store.selectedItem?.type === 'mr' && store.selectedItem?.iid === mr.iid }"
                  @click="selectMR(mr)"
                >
                  <span class="item-iid">!{{ mr.iid }}</span>
                  <span class="item-title">
                    <span v-if="mr.draft" class="draft-badge">Draft</span>
                    {{ mr.title }}
                  </span>
                  <span class="mr-branches">
                    <code>{{ mr.source_branch }}</code>
                    <svg viewBox="0 0 16 16" fill="currentColor" width="10" height="10">
                      <path d="M8.22 2.97a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06l2.97-2.97H3.75a.75.75 0 010-1.5h7.44L8.22 4.03a.75.75 0 010-1.06z"/>
                    </svg>
                    <code>{{ mr.target_branch }}</code>
                  </span>
                  <span class="ci-dot" :class="ciStatusClass(mr.pipeline)" :title="ciStatusText(mr.pipeline)"></span>
                  <span class="item-labels">
                    <span
                      v-for="label in (mr.labels || []).slice(0, 2)"
                      :key="label"
                      class="label-badge"
                      :style="labelStyle(label)"
                    >{{ label }}</span>
                  </span>
                  <span class="item-time">{{ formatTimeAgo(mr.updated_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- PROJECT TAB -->
        <template v-if="store.activeMainTab === 'project'">
          <div v-if="store.projectLoading" class="empty">Loading project...</div>
          <div v-else-if="store.enabled === false" class="empty">
            GitLab is not configured for this project.
          </div>
          <div v-else>
            <div v-if="store.project" class="project-info">
              <h3>{{ store.project.path_with_namespace }}</h3>
              <p v-if="store.project.description" class="project-desc">{{ store.project.description }}</p>
              <a v-if="store.project.web_url" :href="store.project.web_url" target="_blank" class="project-link">
                Open in GitLab
                <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                  <path d="M3.75 2h3.5a.75.75 0 010 1.5h-2.19l5.72 5.72a.75.75 0 11-1.06 1.06L4 4.56v2.19a.75.75 0 01-1.5 0v-3.5A.75.75 0 013.25 2.5h.5zm5.5 0h3.5a.75.75 0 01.75.75v10.5a.75.75 0 01-.75.75h-10.5a.75.75 0 01-.75-.75V9.75a.75.75 0 011.5 0v2.75h9V3.5h-2.75a.75.75 0 010-1.5z"/>
                </svg>
              </a>
            </div>

            <!-- Project issues -->
            <section class="project-section">
              <h4>Issues <span class="section-count" v-if="store.issues.length">{{ store.issues.length }}</span></h4>
              <div v-if="!store.issues.length" class="empty-small">No issues</div>
              <div v-else class="simple-list">
                <div
                  v-for="issue in store.issues"
                  :key="issue.id"
                  class="item-row"
                  @click="selectIssue(issue)"
                >
                  <span class="item-iid">#{{ issue.iid }}</span>
                  <span class="item-title">{{ issue.title }}</span>
                  <span class="state-badge" :class="'state-' + issue.state">{{ issue.state }}</span>
                  <span class="item-time">{{ formatTimeAgo(issue.updated_at) }}</span>
                </div>
              </div>
            </section>

            <!-- Project MRs -->
            <section class="project-section">
              <h4>Merge Requests <span class="section-count" v-if="store.mergeRequests.length">{{ store.mergeRequests.length }}</span></h4>
              <div v-if="!store.mergeRequests.length" class="empty-small">No merge requests</div>
              <div v-else class="simple-list">
                <div
                  v-for="mr in store.mergeRequests"
                  :key="mr.id"
                  class="item-row"
                  @click="selectMR(mr)"
                >
                  <span class="item-iid">!{{ mr.iid }}</span>
                  <span class="item-title">{{ mr.title }}</span>
                  <span class="state-badge" :class="'state-' + mr.state">{{ mr.state }}</span>
                  <span class="item-time">{{ formatTimeAgo(mr.updated_at) }}</span>
                </div>
              </div>
            </section>
          </div>
        </template>

        <!-- PIPELINES TAB -->
        <template v-if="store.activeMainTab === 'pipelines'">
          <div v-if="!store.pipelines.length" class="empty">No pipelines found</div>
          <table v-else class="pipelines-table">
            <thead>
              <tr>
                <th>Status</th>
                <th>Pipeline</th>
                <th>Ref</th>
                <th>SHA</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in store.pipelines" :key="p.id">
                <td>
                  <span class="ci-dot-large" :class="ciStatusClass(p)"></span>
                  <span class="ci-status-text">{{ p.status }}</span>
                </td>
                <td>
                  <a v-if="p.web_url" :href="p.web_url" target="_blank" class="pipeline-link">#{{ p.id }}</a>
                  <span v-else>#{{ p.id }}</span>
                </td>
                <td><code class="ref-badge">{{ p.ref }}</code></td>
                <td><code class="sha-text">{{ p.sha?.slice(0, 8) }}</code></td>
                <td class="time-cell">{{ formatTimeAgo(p.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </template>
      </div>

      <!-- Detail modal -->
      <GitLabDetailModal
        v-if="store.selectedItem"
        :item="detailItem"
        :item-type="store.selectedItem.type"
        :notes="store.detailNotes"
        :loading="store.detailLoading"
        @close="store.closeDetail()"
        @add-comment="handleAddComment"
        @toggle-checkbox="handleToggleCheckbox"
        @update-state="handleUpdateState"
      />
    </div>

    <!-- Create modals -->
    <GitLabCreateIssue
      :visible="store.showCreateIssue"
      :members="store.members"
      :labels="store.labels"
      :milestones="store.milestones"
      :projects="gitlabProjects"
      @close="store.showCreateIssue = false"
      @create="handleCreateIssue"
    />

    <GitLabCreateMR
      :visible="store.showCreateMR"
      :members="store.members"
      :current-branch="currentBranch"
      :projects="gitlabProjects"
      @close="store.showCreateMR = false"
      @create="handleCreateMR"
    />
    </template>
  </div>
</template>

<style scoped>
/* Not configured / loading */
.not-configured {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 24px;
  text-align: center;
  color: var(--text-secondary);
}

.not-configured-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 20px;
  opacity: 0.3;
}

.not-configured-icon svg {
  width: 100%;
  height: 100%;
}

.not-configured h2 {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.not-configured p {
  font-size: 14px;
  margin-bottom: 4px;
  max-width: 480px;
}

.not-configured code {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--bg-tertiary);
  padding: 2px 6px;
  border-radius: 4px;
}

.not-configured-hint {
  margin-top: 12px;
  font-size: 13px;
  opacity: 0.7;
}

.loading-check {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 80px 24px;
  color: var(--text-secondary);
  font-size: 14px;
}

.loading-check .spin-icon {
  width: 18px;
  height: 18px;
}

/* Header */
.page-header h1 {
  font-size: 28px;
  font-weight: 700;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.refresh-indicator {
  display: flex;
  align-items: center;
  color: var(--accent-blue);
}

.header-actions {
  display: flex;
  gap: 8px;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  transition: background 0.15s;
}

.btn:hover:not(:disabled) {
  background: var(--border);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn svg {
  flex-shrink: 0;
}

.btn-green {
  background: rgba(63, 185, 80, 0.15);
  border-color: rgba(63, 185, 80, 0.3);
  color: var(--accent-green);
}

.btn-green:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.25);
}

.btn-blue {
  background: rgba(88, 166, 255, 0.15);
  border-color: rgba(88, 166, 255, 0.3);
  color: var(--accent-blue);
}

.btn-blue:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.25);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.spin-icon {
  width: 18px;
  height: 18px;
  animation: spin 0.8s linear infinite;
}

/* Main tabs */
.main-tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 16px;
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 18px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  color: var(--accent-blue);
  border-bottom-color: var(--accent-blue);
}

.tab-badge {
  font-size: 11px;
  font-weight: 600;
  background: var(--accent-blue);
  color: #fff;
  padding: 0 6px;
  border-radius: 8px;
  min-width: 18px;
  text-align: center;
}

/* Content area */
.content-area {
  display: flex;
  gap: 0;
  min-height: 0;
  flex: 1;
}

.main-content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
}

/* Sub-tabs */
.sub-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
}

.sub-tab {
  padding: 5px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 16px;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.sub-tab:hover {
  color: var(--text-primary);
  background: var(--border);
}

.sub-tab.active {
  background: rgba(88, 166, 255, 0.15);
  border-color: rgba(88, 166, 255, 0.3);
  color: var(--accent-blue);
}

/* Filter bar */
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  align-items: center;
}

.filter-search {
  flex: 1;
  max-width: 300px;
  padding: 6px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-primary);
}

.filter-search:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.2);
  outline: none;
}

.filter-search::placeholder {
  color: var(--text-secondary);
}

.filter-select {
  padding: 6px 10px;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
}

.filter-select:focus {
  border-color: var(--accent-blue);
  outline: none;
}

.filter-clear {
  padding: 5px 12px;
  background: none;
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--accent-red);
  font-size: 12px;
  cursor: pointer;
}

.filter-clear:hover {
  background: rgba(248, 81, 73, 0.1);
}

/* Empty states */
.empty {
  color: var(--text-secondary);
  font-size: 14px;
  padding: 32px;
  background: var(--bg-secondary);
  border-radius: 8px;
  text-align: center;
}

.empty-small {
  color: var(--text-secondary);
  font-size: 13px;
  padding: 16px;
  text-align: center;
}

/* Grouped list */
.grouped-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.group {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 8px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  cursor: pointer;
  user-select: none;
  transition: background 0.1s;
}

.group-header:hover {
  background: var(--bg-tertiary);
}

.group-chevron {
  width: 16px;
  height: 16px;
  color: var(--text-secondary);
  transition: transform 0.15s;
  flex-shrink: 0;
}

.group-chevron.collapsed {
  transform: rotate(-90deg);
}

.group-name {
  font-size: 13px;
  font-weight: 600;
  font-family: var(--font-mono);
  color: var(--text-primary);
}

.group-count {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 0 8px;
  border-radius: 8px;
}

.group-items {
  border-top: 1px solid var(--border);
}

/* Item rows */
.item-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 8px 28px;
  cursor: pointer;
  transition: background 0.1s;
  border-bottom: 1px solid var(--border);
}

.item-row:last-child {
  border-bottom: none;
}

.item-row:hover {
  background: rgba(88, 166, 255, 0.04);
}

.item-row.row-selected {
  background: rgba(88, 166, 255, 0.08);
  border-left: 2px solid var(--accent-blue);
  padding-left: 26px;
}

.item-iid {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 44px;
  flex-shrink: 0;
}

.item-title {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.item-labels {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.label-badge {
  font-size: 11px;
  padding: 1px 8px;
  border-radius: 10px;
  font-weight: 500;
  white-space: nowrap;
  border: 1px solid transparent;
}

.item-due {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.item-due.overdue {
  color: var(--accent-red);
  font-weight: 600;
}

.item-assignee {
  font-size: 12px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.item-time {
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 50px;
  text-align: right;
  flex-shrink: 0;
}

/* MR specific */
.mr-branches {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.mr-branches code {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--bg-tertiary);
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--accent-blue);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mr-branches svg {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.draft-badge {
  font-size: 11px;
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 500;
  margin-right: 4px;
}

/* CI dots */
.ci-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ci-dot-large {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  vertical-align: middle;
  margin-right: 6px;
}

.ci-passed {
  background: var(--accent-green);
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
}

.ci-failed {
  background: var(--accent-red);
  box-shadow: 0 0 6px rgba(248, 81, 73, 0.5);
}

.ci-pending {
  background: var(--accent-orange);
  animation: pulse-ci 1.5s ease-in-out infinite;
}

@keyframes pulse-ci {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.ci-canceled {
  background: var(--text-secondary);
}

.ci-unknown {
  background: var(--border);
}

.ci-status-text {
  font-size: 13px;
  text-transform: capitalize;
}

/* State badges */
.state-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
  flex-shrink: 0;
}

.state-opened {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.state-closed {
  background: rgba(188, 140, 255, 0.15);
  color: var(--accent-purple);
}

.state-merged {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
}

/* Project tab */
.project-info {
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  margin-bottom: 20px;
}

.project-info h3 {
  font-size: 16px;
  font-weight: 600;
  font-family: var(--font-mono);
  margin-bottom: 4px;
}

.project-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.project-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--accent-blue);
}

.project-link:hover {
  text-decoration: underline;
}

.project-section {
  margin-bottom: 20px;
}

.project-section h4 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.section-count {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 400;
  background: var(--bg-tertiary);
  padding: 0 6px;
  border-radius: 8px;
}

.simple-list {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.simple-list .item-row {
  padding-left: 12px;
}

/* Pipelines table */
.pipelines-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.pipelines-table th {
  text-align: left;
  padding: 10px 12px;
  color: var(--text-secondary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
}

.pipelines-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

.pipelines-table tbody tr {
  transition: background 0.1s;
}

.pipelines-table tbody tr:hover {
  background: var(--bg-secondary);
}

.pipeline-link {
  color: var(--accent-blue);
  font-family: var(--font-mono);
  font-size: 13px;
}

.ref-badge {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: 4px;
  color: var(--accent-blue);
}

.sha-text {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-secondary);
}

.time-cell {
  color: var(--text-secondary);
  font-size: 13px;
}


</style>
