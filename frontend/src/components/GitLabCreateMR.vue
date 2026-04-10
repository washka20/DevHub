<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { GitLabMember } from '../types'

const props = defineProps<{
  visible: boolean
  members: GitLabMember[]
  currentBranch: string
  projectNames: string[]
}>()

const emit = defineEmits<{
  close: []
  create: [data: {
    projectName: string
    title: string
    description: string
    source_branch: string
    target_branch: string
    assignee_ids: number[]
    reviewer_ids: number[]
    draft: boolean
    remove_source_branch: boolean
  }]
}>()

const form = ref({
  projectName: '',
  title: '',
  description: '',
  sourceBranch: '',
  targetBranch: 'main',
  assigneeId: null as number | null,
  reviewerIds: [] as number[],
  draft: false,
  removeSourceBranch: false,
})

const submitting = ref(false)

const suggestedTitle = computed(() => {
  const branch = form.value.sourceBranch
  if (!branch) return ''
  return branch
    .replace(/^(feature|fix|hotfix|bugfix|chore|refactor|docs)[/\-_]/, '')
    .replace(/[_-]/g, ' ')
    .replace(/^\w/, c => c.toUpperCase())
})

watch(() => props.visible, (val) => {
  if (val) {
    form.value = {
      projectName: props.projectNames[0] ?? '',
      title: '',
      description: '',
      sourceBranch: props.currentBranch,
      targetBranch: 'main',
      assigneeId: null,
      reviewerIds: [],
      draft: false,
      removeSourceBranch: false,
    }
  }
})

watch(() => form.value.sourceBranch, () => {
  if (!form.value.title) {
    form.value.title = suggestedTitle.value
  }
})

function toggleReviewer(id: number) {
  const idx = form.value.reviewerIds.indexOf(id)
  if (idx >= 0) {
    form.value.reviewerIds.splice(idx, 1)
  } else {
    form.value.reviewerIds.push(id)
  }
}

async function submit() {
  if (!form.value.title.trim() || !form.value.sourceBranch || !form.value.projectName) return
  submitting.value = true
  try {
    emit('create', {
      projectName: form.value.projectName,
      title: (form.value.draft ? 'Draft: ' : '') + form.value.title.trim(),
      description: form.value.description,
      source_branch: form.value.sourceBranch,
      target_branch: form.value.targetBranch,
      assignee_ids: form.value.assigneeId ? [form.value.assigneeId] : [],
      reviewer_ids: form.value.reviewerIds,
      draft: form.value.draft,
      remove_source_branch: form.value.removeSourceBranch,
    })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="modal-overlay" @click.self="emit('close')">
        <div class="modal-content">
          <div class="modal-header">
            <h3>New Merge Request</h3>
            <button class="modal-close" @click="emit('close')">&times;</button>
          </div>

          <form class="modal-body" @submit.prevent="submit">
            <div class="form-group">
              <label class="form-label">Project</label>
              <select v-model="form.projectName" class="form-select">
                <option v-for="name in projectNames" :key="name" :value="name">
                  {{ name }}
                </option>
              </select>
            </div>

            <div class="branch-row">
              <div class="form-group branch-field">
                <label class="form-label">Source branch <span class="required">*</span></label>
                <input
                  v-model="form.sourceBranch"
                  class="form-input branch-input"
                  type="text"
                  placeholder="feature/my-branch"
                />
              </div>
              <div class="branch-arrow">
                <svg viewBox="0 0 16 16" fill="currentColor" width="16" height="16">
                  <path d="M8.22 2.97a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06l2.97-2.97H3.75a.75.75 0 010-1.5h7.44L8.22 4.03a.75.75 0 010-1.06z"/>
                </svg>
              </div>
              <div class="form-group branch-field">
                <label class="form-label">Target branch</label>
                <input
                  v-model="form.targetBranch"
                  class="form-input branch-input"
                  type="text"
                  placeholder="main"
                />
              </div>
            </div>

            <div class="form-group">
              <label class="form-label">Title <span class="required">*</span></label>
              <input
                v-model="form.title"
                class="form-input"
                type="text"
                :placeholder="suggestedTitle || 'MR title'"
              />
            </div>

            <div class="form-group">
              <label class="form-label">Description</label>
              <textarea
                v-model="form.description"
                class="form-textarea"
                placeholder="Describe the changes..."
                rows="4"
              ></textarea>
            </div>

            <div class="form-row">
              <div class="form-group form-half">
                <label class="form-label">Assignee</label>
                <select v-model="form.assigneeId" class="form-select">
                  <option :value="null">None</option>
                  <option v-for="m in members" :key="m.id" :value="m.id">
                    {{ m.name }} (@{{ m.username }})
                  </option>
                </select>
              </div>

              <div class="form-group form-half">
                <label class="form-label">Reviewers</label>
                <div class="reviewers-list">
                  <button
                    v-for="m in members"
                    :key="m.id"
                    type="button"
                    class="reviewer-chip"
                    :class="{ active: form.reviewerIds.includes(m.id) }"
                    @click="toggleReviewer(m.id)"
                  >
                    {{ m.name }}
                  </button>
                  <span v-if="!members.length" class="empty-hint">No members</span>
                </div>
              </div>
            </div>

            <div class="checkboxes-row">
              <label class="checkbox-label">
                <input type="checkbox" v-model="form.draft" />
                <span>Mark as Draft</span>
              </label>
              <label class="checkbox-label">
                <input type="checkbox" v-model="form.removeSourceBranch" />
                <span>Delete source branch after merge</span>
              </label>
            </div>

            <div class="modal-footer">
              <button type="button" class="btn-cancel" @click="emit('close')">Cancel</button>
              <button
                type="submit"
                class="btn-submit"
                :disabled="!form.title.trim() || !form.sourceBranch || !form.projectName || submitting"
              >
                <svg v-if="submitting" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                </svg>
                Create MR
              </button>
            </div>
          </form>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  width: 600px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 24px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.modal-close:hover {
  color: var(--accent-red);
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.required {
  color: var(--accent-red);
}

.form-input,
.form-select,
.form-textarea {
  width: 100%;
  background: var(--bg-primary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 14px;
  color: var(--text-primary);
  font-family: inherit;
}

.form-input:focus,
.form-select:focus,
.form-textarea:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.2);
  outline: none;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
  line-height: 1.5;
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-half {
  flex: 1;
}

.branch-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.branch-field {
  flex: 1;
}

.branch-input {
  font-family: var(--font-mono);
  font-size: 13px;
}

.branch-arrow {
  color: var(--text-secondary);
  padding-bottom: 24px;
}

.reviewers-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.reviewer-chip {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.reviewer-chip:hover {
  border-color: var(--text-secondary);
}

.reviewer-chip.active {
  background: rgba(88, 166, 255, 0.15);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.empty-hint {
  font-size: 13px;
  color: var(--text-secondary);
}

.checkboxes-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--accent-blue);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
  margin-top: 8px;
}

.btn-cancel {
  padding: 8px 16px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
}

.btn-cancel:hover {
  background: var(--border);
}

.btn-submit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 20px;
  border-radius: 6px;
  border: 1px solid rgba(88, 166, 255, 0.4);
  background: rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-submit:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.25);
  border-color: var(--accent-blue);
}

.btn-submit:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.spin-icon {
  width: 16px;
  height: 16px;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.modal-enter-active {
  transition: opacity 0.2s ease;
}

.modal-leave-active {
  transition: opacity 0.15s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .modal-content {
  transition: transform 0.2s ease;
}

.modal-enter-from .modal-content {
  transform: scale(0.95) translateY(10px);
}
</style>
