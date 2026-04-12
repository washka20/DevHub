<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { GitLabIssue, GitLabMR, GitLabNote, GitLabDiscussion } from '../types'
import { useMarkdown } from '../composables/useMarkdown'
import { useGitLabStore } from '../stores/gitlab'
import { formatRelativeTime, formatDate, isOverdue, formatDueDate } from '../utils/date'
import { hexToRgb } from '../utils/color'

const props = defineProps<{
  item: GitLabIssue | GitLabMR | null
  itemType: 'issue' | 'mr'
  notes: GitLabNote[]
  discussions: GitLabDiscussion[]
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  'add-comment': [body: string]
  'toggle-checkbox': [index: number]
  'update-state': [stateEvent: 'close' | 'reopen']
  'resolve-discussion': [discussionId: string, resolved: boolean]
  'reply-to-discussion': [discussionId: string, body: string]
}>()

const { render } = useMarkdown()
const gitlabStore = useGitLabStore()

const commentText = ref('')
const submitting = ref(false)
const approvingMR = ref(false)

const isMR = computed(() => props.itemType === 'mr')
const itemAsMR = computed(() => props.item as GitLabMR | null)
const itemAsIssue = computed(() => props.item as GitLabIssue | null)

const stateLabel = computed(() => {
  if (!props.item) return ''
  switch (props.item.state) {
    case 'opened': return 'Open'
    case 'closed': return 'Closed'
    case 'merged': return 'Merged'
    default: return props.item.state
  }
})

const stateClass = computed(() => {
  if (!props.item) return ''
  switch (props.item.state) {
    case 'opened': return 'state-open'
    case 'closed': return 'state-closed'
    case 'merged': return 'state-merged'
    default: return ''
  }
})

const checkboxes = computed(() => {
  if (!props.item?.description) return []
  const lines = props.item.description.split('\n')
  const result: { text: string; checked: boolean; index: number }[] = []
  let idx = 0
  for (const line of lines) {
    const match = line.match(/^\s*[-*]\s+\[([ xX])\]\s*(.*)$/)
    if (match) {
      result.push({
        text: match[2],
        checked: match[1] !== ' ',
        index: idx,
      })
      idx++
    }
  }
  return result
})

const descriptionWithoutCheckboxes = computed(() => {
  if (!props.item?.description) return ''
  return props.item.description
    .split('\n')
    .filter(line => !/^\s*[-*]\s+\[([ xX])\]\s/.test(line))
    .join('\n')
})

const checkedCount = computed(() => checkboxes.value.filter(c => c.checked).length)
const taskProgress = computed(() => {
  if (!checkboxes.value.length) return 0
  return (checkedCount.value / checkboxes.value.length) * 100
})

// Extract project base URL from item's web_url: https://gitlab.host/group/project/-/issues/42 → https://gitlab.host/group/project
const projectWebUrl = computed(() => {
  const webUrl = props.item?.web_url
  if (!webUrl) return undefined
  const idx = webUrl.indexOf('/-/')
  return idx !== -1 ? webUrl.substring(0, idx) : undefined
})

const markdownOpts = computed(() =>
  projectWebUrl.value ? { projectWebUrl: projectWebUrl.value } : undefined
)

const userNotes = computed(() => props.notes.filter(n => !n.system))

const userDiscussions = computed(() =>
  props.discussions.filter(d => d.notes.some(n => !n.system))
)

const resolvableThreads = computed(() =>
  userDiscussions.value.filter(d => !d.individual_note && d.notes.some(n => n.resolvable))
)

const resolvedCount = computed(() =>
  resolvableThreads.value.filter(d => d.notes.every(n => !n.resolvable || n.resolved)).length
)

const replyingTo = ref<string | null>(null)
const replyText = ref('')
const replySubmitting = ref(false)

function toggleReply(discussionId: string) {
  if (replyingTo.value === discussionId) {
    replyingTo.value = null
    replyText.value = ''
  } else {
    replyingTo.value = discussionId
    replyText.value = ''
  }
}

async function submitReply(discussionId: string) {
  if (!replyText.value.trim()) return
  replySubmitting.value = true
  try {
    emit('reply-to-discussion', discussionId, replyText.value.trim())
    replyText.value = ''
    replyingTo.value = null
  } finally {
    replySubmitting.value = false
  }
}

function isThreadResolved(d: GitLabDiscussion): boolean {
  return d.notes.every(n => !n.resolvable || n.resolved)
}

const canToggleState = computed(() => {
  if (!props.item) return false
  return props.item.state === 'opened' || props.item.state === 'closed'
})

function labelStyle(labelName: string): Record<string, string> {
  // First check label_details on the item itself (has colors from API)
  const fromItem = props.item?.label_details?.find(d => d.name === labelName)
  // Fallback to global labels store
  const fromStore = gitlabStore.labels.find(l => l.name === labelName)
  const color = fromItem?.color ?? fromStore?.color
  if (color) {
    const rgb = hexToRgb(color)
    return {
      background: `rgba(${rgb},0.1)`,
      color,
      borderColor: `rgba(${rgb},0.25)`,
    }
  }
  return {
    background: 'rgba(139,148,158,0.1)',
    color: '#8b949e',
    borderColor: 'rgba(139,148,158,0.25)',
  }
}

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

const approvals = computed(() => gitlabStore.mrApprovals)

const approvalStatusText = computed(() => {
  const a = approvals.value
  if (!a) return ''
  if (a.approved) return 'Approved'
  const approved = a.approvals_required - a.approvals_left
  return `${approved}/${a.approvals_required} Approved`
})

async function toggleApproval() {
  const sel = gitlabStore.selectedItem
  if (!sel || sel.type !== 'mr') return
  approvingMR.value = true
  try {
    const a = approvals.value
    const isApprovedByMe = a?.approved_by?.some(
      ab => gitlabStore.members[0] && ab.user.id === gitlabStore.members[0].id
    )
    if (isApprovedByMe) {
      await gitlabStore.unapproveMR(sel.projectId, sel.iid)
    } else {
      await gitlabStore.approveMR(sel.projectId, sel.iid)
    }
  } finally {
    approvingMR.value = false
  }
}

const isApprovedByCurrentUser = computed(() => {
  const a = approvals.value
  const currentUser = gitlabStore.members[0]
  if (!a || !currentUser) return false
  return a.approved_by.some(ab => ab.user.id === currentUser.id)
})

const formatTimeAgo = formatRelativeTime

async function submitComment() {
  if (!commentText.value.trim()) return
  submitting.value = true
  try {
    emit('add-comment', commentText.value.trim())
    commentText.value = ''
  } finally {
    submitting.value = false
  }
}

function onOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('modal-overlay')) {
    emit('close')
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div class="modal-overlay" @click="onOverlayClick">
      <div class="modal-detail">
        <!-- Header -->
        <div class="modal-header">
          <div class="modal-title-row">
            <span class="modal-title">
              {{ isMR ? '!' : '#' }}{{ item?.iid }} {{ item?.title }}
            </span>
            <button class="modal-close" @click="emit('close')" title="Close">&#x2715;</button>
          </div>
          <div class="modal-meta">
            <span class="state-badge" :class="stateClass">{{ stateLabel }}</span>
            <span
              v-for="label in item?.labels"
              :key="label"
              class="label-badge"
              :style="labelStyle(label)"
            >{{ label }}</span>
          </div>
          <div class="modal-info">
            <strong class="modal-author">@{{ item?.author?.username }}</strong>
            <template v-if="!isMR && itemAsIssue?.due_date">
              <span class="modal-info-sep">&middot;</span>
              <span :class="{ 'text-overdue': isOverdue(itemAsIssue.due_date) }">
                Due {{ formatDate(itemAsIssue.due_date) }}
                <template v-if="isOverdue(itemAsIssue.due_date)">(overdue)</template>
              </span>
            </template>
            <span class="modal-info-sep">&middot;</span>
            <span>Updated {{ formatTimeAgo(item?.updated_at ?? '') }}</span>
          </div>
          <div class="modal-actions">
            <a
              v-if="item?.web_url"
              :href="item.web_url"
              target="_blank"
              rel="noopener"
              class="action-btn"
            >Open in GitLab &#x2197;</a>
            <button
              v-if="isMR && approvals && item?.state === 'opened'"
              class="action-btn"
              :class="isApprovedByCurrentUser ? 'btn-unapprove' : 'btn-approve'"
              :disabled="approvingMR"
              @click="toggleApproval"
            >
              {{ isApprovedByCurrentUser ? 'Unapprove' : 'Approve' }}
            </button>
            <button
              v-if="canToggleState"
              class="action-btn"
              :class="item?.state === 'opened' ? 'btn-close-item' : 'btn-reopen-item'"
              @click="emit('update-state', item?.state === 'opened' ? 'close' : 'reopen')"
            >
              {{ item?.state === 'opened' ? `Close ${isMR ? 'MR' : 'Issue'}` : `Reopen ${isMR ? 'MR' : 'Issue'}` }}
            </button>
          </div>
        </div>

        <!-- Loading -->
        <div v-if="loading" class="modal-loading">
          <div class="loading-spinner"></div>
          Loading...
        </div>

        <!-- Body: two columns -->
        <div v-else class="modal-body">
          <!-- Left column -->
          <div class="modal-left">
            <!-- Description -->
            <template v-if="item?.description">
              <div class="section-title">Description</div>
              <div class="markdown-body" v-html="render(descriptionWithoutCheckboxes, markdownOpts)"></div>
            </template>

            <!-- Tasks -->
            <template v-if="checkboxes.length">
              <div class="section-title">Tasks</div>
              <div class="task-progress">
                <div class="task-bar">
                  <div class="task-fill" :style="{ width: `${taskProgress}%` }"></div>
                </div>
                <span class="task-count">{{ checkedCount }} / {{ checkboxes.length }}</span>
              </div>
              <div
                v-for="cb in checkboxes"
                :key="cb.index"
                class="task-item"
                @click="emit('toggle-checkbox', cb.index)"
              >
                <span class="task-cb" :class="{ done: cb.checked }">
                  <template v-if="cb.checked">&#x2713;</template>
                </span>
                <span class="task-text" :class="{ done: cb.checked }">{{ cb.text }}</span>
              </div>
            </template>

            <!-- MR Discussions (threaded) -->
            <template v-if="isMR">
              <div class="section-title">
                Discussions
                <template v-if="resolvableThreads.length">
                  <span class="threads-summary">({{ resolvedCount }}/{{ resolvableThreads.length }} threads resolved)</span>
                </template>
              </div>

              <div v-if="!userDiscussions.length" class="no-comments">No discussions yet</div>

              <div
                v-for="discussion in userDiscussions"
                :key="discussion.id"
                class="discussion"
                :class="{ 'discussion-thread': !discussion.individual_note && discussion.notes.some(n => n.resolvable) }"
              >
                <!-- Thread header with resolve badge -->
                <div
                  v-if="!discussion.individual_note && discussion.notes.some(n => n.resolvable)"
                  class="thread-header"
                >
                  <span
                    class="thread-badge"
                    :class="isThreadResolved(discussion) ? 'thread-resolved' : 'thread-unresolved'"
                  >{{ isThreadResolved(discussion) ? 'Resolved' : 'Unresolved' }}</span>
                  <button
                    class="thread-resolve-btn"
                    @click="emit('resolve-discussion', discussion.id, !isThreadResolved(discussion))"
                  >{{ isThreadResolved(discussion) ? 'Unresolve' : 'Resolve' }}</button>
                </div>

                <!-- Root note (first in thread, or only note for individual) -->
                <div class="comment">
                  <div class="comment-head">
                    <span class="comment-avatar">{{ getInitials(discussion.notes[0].author.name) }}</span>
                    <span class="comment-author">@{{ discussion.notes[0].author.username }}</span>
                    <span class="comment-time">&middot; {{ formatTimeAgo(discussion.notes[0].created_at) }}</span>
                  </div>
                  <div class="comment-body" v-html="render(discussion.notes[0].body, markdownOpts)"></div>
                </div>

                <!-- Thread replies -->
                <template v-if="!discussion.individual_note && discussion.notes.length > 1">
                  <div
                    v-for="note in discussion.notes.slice(1).filter(n => !n.system)"
                    :key="note.id"
                    class="comment thread-reply"
                  >
                    <div class="comment-head">
                      <span class="comment-avatar">{{ getInitials(note.author.name) }}</span>
                      <span class="comment-author">@{{ note.author.username }}</span>
                      <span class="comment-time">&middot; {{ formatTimeAgo(note.created_at) }}</span>
                    </div>
                    <div class="comment-body" v-html="render(note.body, markdownOpts)"></div>
                  </div>
                </template>

                <!-- Reply button & input for threads -->
                <div v-if="!discussion.individual_note" class="thread-actions">
                  <button class="thread-reply-btn" @click="toggleReply(discussion.id)">
                    {{ replyingTo === discussion.id ? 'Cancel' : 'Reply' }}
                  </button>
                </div>
                <div v-if="replyingTo === discussion.id" class="thread-reply-box">
                  <textarea
                    v-model="replyText"
                    class="comment-input"
                    placeholder="Reply to thread... (Markdown)"
                    @keydown.ctrl.enter="submitReply(discussion.id)"
                    @keydown.meta.enter="submitReply(discussion.id)"
                  ></textarea>
                  <div class="comment-footer">
                    <span class="comment-hint">Ctrl+Enter to send</span>
                    <button
                      class="send-btn"
                      :disabled="!replyText.trim() || replySubmitting"
                      @click="submitReply(discussion.id)"
                    >Reply</button>
                  </div>
                </div>
              </div>
            </template>

            <!-- Issue Comments (flat) -->
            <template v-else>
              <div class="section-title">Comments<template v-if="userNotes.length"> ({{ userNotes.length }})</template></div>

              <div v-if="!userNotes.length" class="no-comments">No comments yet</div>

              <div v-for="note in userNotes" :key="note.id" class="comment">
                <div class="comment-head">
                  <span class="comment-avatar">{{ getInitials(note.author.name) }}</span>
                  <span class="comment-author">@{{ note.author.username }}</span>
                  <span class="comment-time">&middot; {{ formatTimeAgo(note.created_at) }}</span>
                </div>
                <div class="comment-body" v-html="render(note.body, markdownOpts)"></div>
              </div>
            </template>

            <!-- Comment input -->
            <div class="comment-input-box">
              <textarea
                v-model="commentText"
                class="comment-input"
                placeholder="Write a comment... (Markdown)"
                @keydown.ctrl.enter="submitComment"
                @keydown.meta.enter="submitComment"
              ></textarea>
              <div class="comment-footer">
                <span class="comment-hint">Ctrl+Enter to send</span>
                <button
                  class="send-btn"
                  :disabled="!commentText.trim() || submitting"
                  @click="submitComment"
                >Send</button>
              </div>
            </div>
          </div>

          <!-- Right sidebar -->
          <div class="modal-right">
            <!-- Assignee -->
            <div class="sidebar-field">
              <div class="sidebar-label">Assignee</div>
              <div v-if="item?.assignees?.length" class="sidebar-value">
                <div
                  v-for="assignee in item.assignees"
                  :key="assignee.id"
                  class="assignee-row"
                >
                  <span class="avatar-sm">{{ getInitials(assignee.name) }}</span>
                  <span>{{ assignee.username }}</span>
                </div>
              </div>
              <div v-else class="sidebar-value sidebar-empty">None</div>
            </div>

            <!-- Milestone (issues only, but MRs could have it via project) -->
            <div v-if="!isMR && itemAsIssue?.milestone" class="sidebar-field">
              <div class="sidebar-label">Milestone</div>
              <div class="sidebar-value">{{ itemAsIssue.milestone.title }}</div>
            </div>

            <!-- Due Date (issues only) -->
            <div v-if="!isMR && itemAsIssue?.due_date" class="sidebar-field">
              <div class="sidebar-label">Due Date</div>
              <div
                class="sidebar-value"
                :class="{ 'text-overdue': isOverdue(itemAsIssue.due_date) }"
              >{{ formatDueDate(itemAsIssue.due_date) }}</div>
            </div>

            <!-- Labels -->
            <div v-if="item?.labels?.length" class="sidebar-field">
              <div class="sidebar-label">Labels</div>
              <div class="sidebar-labels">
                <span
                  v-for="label in item.labels"
                  :key="label"
                  class="label-badge label-badge-lg"
                  :style="labelStyle(label)"
                >{{ label }}</span>
              </div>
            </div>

            <!-- Created -->
            <div class="sidebar-field">
              <div class="sidebar-label">Created</div>
              <div class="sidebar-value">{{ formatDate(item?.created_at ?? '') }}</div>
            </div>

            <!-- Updated -->
            <div class="sidebar-field">
              <div class="sidebar-label">Updated</div>
              <div class="sidebar-value">{{ formatTimeAgo(item?.updated_at ?? '') }}</div>
            </div>

            <!-- Project -->
            <div class="sidebar-field">
              <div class="sidebar-label">Project</div>
              <div class="sidebar-value">
                <a v-if="item?.web_url" :href="item.web_url.split('/-/')[0]" target="_blank" rel="noopener">
                  {{ item?.project_path }}
                </a>
                <span v-else>{{ item?.project_path }}</span>
              </div>
            </div>

            <!-- MR branches -->
            <template v-if="isMR && itemAsMR">
              <div class="sidebar-field">
                <div class="sidebar-label">Branches</div>
                <div class="sidebar-value branch-flow">
                  <code>{{ itemAsMR.source_branch }}</code>
                  <span class="branch-arrow">&rarr;</span>
                  <code>{{ itemAsMR.target_branch }}</code>
                </div>
              </div>

              <!-- Pipeline -->
              <div v-if="itemAsMR.pipeline" class="sidebar-field">
                <div class="sidebar-label">Pipeline</div>
                <div class="sidebar-value">
                  <a :href="itemAsMR.pipeline.web_url" target="_blank" rel="noopener">
                    #{{ itemAsMR.pipeline.id }}
                  </a>
                  <span class="pipeline-status" :class="`pipeline-${itemAsMR.pipeline.status}`">
                    {{ itemAsMR.pipeline.status }}
                  </span>
                </div>
              </div>

              <!-- Draft -->
              <div v-if="itemAsMR.draft" class="sidebar-field">
                <div class="sidebar-label">Draft</div>
                <div class="sidebar-value">Yes</div>
              </div>

              <!-- Reviewers -->
              <div v-if="itemAsMR.reviewers?.length" class="sidebar-field">
                <div class="sidebar-label">Reviewers</div>
                <div class="sidebar-value">
                  <div
                    v-for="reviewer in itemAsMR.reviewers"
                    :key="reviewer.id"
                    class="assignee-row"
                  >
                    <span class="avatar-sm">{{ getInitials(reviewer.name) }}</span>
                    <span>{{ reviewer.username }}</span>
                  </div>
                </div>
              </div>

              <!-- Approvals -->
              <div v-if="approvals" class="sidebar-field">
                <div class="sidebar-label">Approvals</div>
                <div class="sidebar-value">
                  <span
                    class="approval-badge"
                    :class="approvals.approved ? 'approval-approved' : 'approval-pending'"
                  >{{ approvalStatusText }}</span>
                </div>
                <div v-if="approvals.approved_by.length" class="approval-list">
                  <div
                    v-for="ab in approvals.approved_by"
                    :key="ab.user.id"
                    class="assignee-row"
                  >
                    <span class="avatar-sm approval-avatar">{{ getInitials(ab.user.name) }}</span>
                    <span>{{ ab.user.username }}</span>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* === Overlay === */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn 0.15s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.97) translateY(6px); }
  to { opacity: 1; transform: none; }
}

/* === Modal box === */
.modal-detail {
  width: 90vw;
  max-width: 1200px;
  height: 85vh;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
  animation: modalIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

/* === Header === */
.modal-header {
  padding: 16px 24px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.modal-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.modal-title {
  font-size: 18px;
  font-weight: 600;
  line-height: 1.4;
  flex: 1;
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 22px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 6px;
  line-height: 1;
  flex-shrink: 0;
}

.modal-close:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.modal-meta {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 6px;
}

.state-badge {
  font-size: 11px;
  padding: 2px 10px;
  border-radius: 10px;
  font-weight: 600;
}

.state-open {
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

.label-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
  white-space: nowrap;
  border: 1px solid;
  line-height: 1.4;
}

.modal-info {
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.modal-author {
  color: var(--accent-blue);
}

.modal-info-sep {
  margin: 0 2px;
}

.text-overdue {
  color: var(--accent-red);
}

.modal-actions {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}

.action-btn {
  padding: 4px 10px;
  border-radius: 5px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 11px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  text-decoration: none;
}

.action-btn:hover {
  border-color: var(--accent-blue);
}

.btn-close-item {
  color: var(--accent-red);
  border-color: rgba(248, 81, 73, 0.25);
}

.btn-close-item:hover {
  background: rgba(248, 81, 73, 0.1);
}

.btn-reopen-item {
  color: var(--accent-green);
  border-color: rgba(63, 185, 80, 0.25);
}

.btn-reopen-item:hover {
  background: rgba(63, 185, 80, 0.1);
}

/* === Loading === */
.modal-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
}

.loading-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* === Body (two columns) === */
.modal-body {
  flex: 1;
  overflow: hidden;
  display: flex;
}

.modal-left {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 24px;
}

.modal-right {
  width: 300px;
  flex-shrink: 0;
  overflow-y: auto;
  padding: 20px 24px;
  border-left: 1px solid var(--border);
  background: var(--bg-primary);
}

/* === Section titles === */
.section-title {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--border);
  margin-top: 20px;
}

.section-title:first-child {
  margin-top: 0;
}

/* === Markdown body === */
.markdown-body {
  font-size: 14px;
  line-height: 1.7;
}

.markdown-body :deep(h1) {
  font-size: 20px;
  font-weight: 600;
  margin: 18px 0 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--border);
}

.markdown-body :deep(h2) {
  font-size: 16px;
  font-weight: 600;
  margin: 16px 0 6px;
}

.markdown-body :deep(h3) {
  font-size: 14px;
  font-weight: 600;
  margin: 12px 0 4px;
}

.markdown-body :deep(p) {
  margin-bottom: 8px;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
  margin-bottom: 8px;
}

.markdown-body :deep(li) {
  margin-bottom: 2px;
}

.markdown-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--bg-tertiary);
  padding: 2px 5px;
  border-radius: 3px;
}

.markdown-body :deep(pre) {
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 12px 14px;
  margin-bottom: 10px;
  overflow-x: auto;
}

.markdown-body :deep(pre code) {
  background: none;
  padding: 0;
  font-size: 12px;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 10px;
  font-size: 13px;
}

.markdown-body :deep(th) {
  text-align: left;
  padding: 6px 10px;
  border-bottom: 2px solid var(--border);
  color: var(--text-secondary);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.markdown-body :deep(td) {
  padding: 5px 10px;
  border-bottom: 1px solid var(--border);
}

.markdown-body :deep(tr:hover td) {
  background: rgba(88, 166, 255, 0.03);
}

.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--accent-blue);
  padding-left: 12px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.markdown-body :deep(a) {
  color: var(--accent-blue);
}

.markdown-body :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
  border: 1px solid var(--border);
  margin: 4px 0;
}

.markdown-body :deep(.task-list) {
  list-style: none;
  padding-left: 0;
}

.markdown-body :deep(.task-list-item) {
  display: flex;
  align-items: flex-start;
  gap: 6px;
}

.markdown-body :deep(.md-checkbox) {
  margin-top: 3px;
  accent-color: var(--accent-green);
}

/* === Tasks section === */
.task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.task-bar {
  flex: 1;
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.task-fill {
  height: 100%;
  background: var(--accent-green);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.task-count {
  font-size: 11px;
  color: var(--text-secondary);
}

.task-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 6px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 13px;
}

.task-item:hover {
  background: rgba(88, 166, 255, 0.05);
}

.task-cb {
  width: 16px;
  height: 16px;
  border: 1.5px solid var(--border);
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 1px;
  font-size: 10px;
  transition: all 0.15s;
}

.task-cb.done {
  background: var(--accent-green);
  border-color: var(--accent-green);
  color: #fff;
}

.task-text.done {
  text-decoration: line-through;
  color: var(--text-secondary);
}

/* === Comments === */
.no-comments {
  font-size: 13px;
  color: var(--text-secondary);
  text-align: center;
  padding: 12px 0;
}

.comment {
  padding: 12px 0;
  border-bottom: 1px solid rgba(48, 54, 61, 0.4);
}

.comment:last-of-type {
  border-bottom: none;
}

.comment-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.comment-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: var(--text-secondary);
  font-weight: 700;
  flex-shrink: 0;
}

.comment-author {
  font-size: 12px;
  font-weight: 600;
  color: var(--accent-blue);
}

.comment-time {
  font-size: 10px;
  color: var(--text-secondary);
}

.comment-body {
  font-size: 13px;
  line-height: 1.6;
  padding-left: 28px;
}

.comment-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--bg-tertiary);
  padding: 1px 4px;
  border-radius: 3px;
}

.comment-body :deep(p) {
  margin-bottom: 4px;
}

.comment-body :deep(p:last-child) {
  margin-bottom: 0;
}

/* === Comment input === */
.comment-input-box {
  margin-top: 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.comment-input {
  width: 100%;
  min-height: 64px;
  padding: 10px 12px;
  font-size: 13px;
  background: var(--bg-primary);
  border: none;
  color: var(--text-primary);
  font-family: var(--font-mono);
  resize: vertical;
}

.comment-input:focus {
  outline: none;
}

.comment-input::placeholder {
  color: var(--text-secondary);
}

.comment-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  background: var(--bg-tertiary);
}

.comment-hint {
  font-size: 10px;
  color: var(--text-secondary);
}

.send-btn {
  padding: 3px 10px;
  font-size: 11px;
  border-radius: 5px;
  border: 1px solid rgba(88, 166, 255, 0.25);
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  cursor: pointer;
  font-weight: 500;
}

.send-btn:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.2);
  border-color: var(--accent-blue);
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* === Right sidebar === */
.sidebar-field {
  margin-bottom: 18px;
}

.sidebar-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.sidebar-value {
  font-size: 14px;
  line-height: 1.5;
}

.sidebar-value a {
  color: var(--accent-blue);
  text-decoration: none;
}

.sidebar-value a:hover {
  text-decoration: underline;
}

.sidebar-empty {
  color: var(--text-secondary);
  font-size: 13px;
}

.sidebar-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
  margin-top: 3px;
}

.label-badge-lg {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: 10px;
}

.assignee-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.assignee-row:last-child {
  margin-bottom: 0;
}

.avatar-sm {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  color: var(--text-secondary);
  font-weight: 700;
  flex-shrink: 0;
}

.branch-flow {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.branch-flow code {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--bg-tertiary);
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--accent-blue);
}

.branch-arrow {
  color: var(--text-secondary);
  font-size: 12px;
}

.pipeline-status {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 8px;
  margin-left: 4px;
  font-weight: 500;
}

.pipeline-success {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.pipeline-failed {
  background: rgba(248, 81, 73, 0.15);
  color: var(--accent-red);
}

.pipeline-running {
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
}

.pipeline-pending,
.pipeline-created,
.pipeline-waiting_for_resource,
.pipeline-preparing {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.pipeline-canceled,
.pipeline-skipped {
  background: rgba(139, 148, 158, 0.15);
  color: var(--text-secondary);
}

/* === Approvals === */
.approval-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 600;
}

.approval-approved {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.approval-pending {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.approval-list {
  margin-top: 6px;
}

.approval-avatar {
  background: rgba(63, 185, 80, 0.2);
  color: var(--accent-green);
}

.btn-approve {
  color: var(--accent-green);
  border-color: rgba(63, 185, 80, 0.25);
}

.btn-approve:hover:not(:disabled) {
  background: rgba(63, 185, 80, 0.1);
}

.btn-unapprove {
  color: var(--accent-orange);
  border-color: rgba(210, 153, 34, 0.25);
}

.btn-unapprove:hover:not(:disabled) {
  background: rgba(210, 153, 34, 0.1);
}

/* === Discussions === */
.threads-summary {
  font-weight: 400;
  font-size: 10px;
  color: var(--text-secondary);
  margin-left: 4px;
  text-transform: none;
  letter-spacing: 0;
}

.discussion {
  margin-bottom: 4px;
}

.discussion-thread {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 10px;
  background: rgba(0, 0, 0, 0.1);
}

.thread-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.thread-badge {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 8px;
  font-weight: 600;
}

.thread-resolved {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
}

.thread-unresolved {
  background: rgba(210, 153, 34, 0.15);
  color: var(--accent-orange);
}

.thread-resolve-btn {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 5px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
}

.thread-resolve-btn:hover {
  color: var(--text-primary);
  border-color: var(--accent-blue);
}

.thread-reply {
  margin-left: 24px;
  padding-left: 12px;
  border-left: 2px solid var(--border);
}

.thread-actions {
  margin-top: 4px;
  margin-left: 28px;
}

.thread-reply-btn {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 5px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
}

.thread-reply-btn:hover {
  color: var(--text-primary);
  border-color: var(--accent-blue);
}

.thread-reply-box {
  margin-top: 6px;
  margin-left: 28px;
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}

.thread-reply-box .comment-input {
  min-height: 48px;
}
</style>
