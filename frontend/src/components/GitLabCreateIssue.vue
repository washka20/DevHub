<script setup lang="ts">
import { ref, watch } from 'vue'
import type { GitLabLabel, GitLabMilestone, GitLabMember } from '../types'

interface GitLabProjectRef {
  id: number
  path: string
}

const props = defineProps<{
  visible: boolean
  members: GitLabMember[]
  labels: GitLabLabel[]
  milestones: GitLabMilestone[]
  projects: GitLabProjectRef[]
}>()

const emit = defineEmits<{
  close: []
  create: [data: {
    projectId: number
    title: string
    description: string
    labels: string[]
    assignee_ids: number[]
    milestone_id: number | undefined
  }]
}>()

const form = ref({
  projectId: null as number | null,
  title: '',
  description: '',
  selectedLabels: [] as string[],
  assigneeId: null as number | null,
  milestoneId: null as number | null,
})

const submitting = ref(false)

watch(() => props.visible, (val) => {
  if (val) {
    form.value = {
      projectId: props.projects[0]?.id ?? null,
      title: '',
      description: '',
      selectedLabels: [],
      assigneeId: null,
      milestoneId: null,
    }
  }
})

function toggleLabel(name: string) {
  const idx = form.value.selectedLabels.indexOf(name)
  if (idx >= 0) {
    form.value.selectedLabels.splice(idx, 1)
  } else {
    form.value.selectedLabels.push(name)
  }
}

async function submit() {
  if (!form.value.title.trim() || !form.value.projectId) return
  submitting.value = true
  try {
    emit('create', {
      projectId: form.value.projectId,
      title: form.value.title.trim(),
      description: form.value.description,
      labels: form.value.selectedLabels,
      assignee_ids: form.value.assigneeId ? [form.value.assigneeId] : [],
      milestone_id: form.value.milestoneId ?? undefined,
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
            <h3>New Issue</h3>
            <button class="modal-close" @click="emit('close')">&times;</button>
          </div>

          <form class="modal-body" @submit.prevent="submit">
            <div class="form-group">
              <label class="form-label">Project</label>
              <select v-model="form.projectId" class="form-select">
                <option v-for="p in projects" :key="p.id" :value="p.id">
                  {{ p.path }}
                </option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">Title <span class="required">*</span></label>
              <input
                v-model="form.title"
                class="form-input"
                type="text"
                placeholder="Issue title"
                autofocus
              />
            </div>

            <div class="form-group">
              <label class="form-label">Description</label>
              <textarea
                v-model="form.description"
                class="form-textarea"
                placeholder="Describe the issue..."
                rows="5"
              ></textarea>
            </div>

            <div class="form-group">
              <label class="form-label">Labels</label>
              <div class="labels-picker">
                <button
                  v-for="label in labels"
                  :key="label.id"
                  type="button"
                  class="label-chip"
                  :class="{ active: form.selectedLabels.includes(label.name) }"
                  :style="{ '--label-color': label.color }"
                  @click="toggleLabel(label.name)"
                >
                  {{ label.name }}
                </button>
                <span v-if="!labels.length" class="empty-hint">No labels available</span>
              </div>
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
                <label class="form-label">Milestone</label>
                <select v-model="form.milestoneId" class="form-select">
                  <option :value="null">None</option>
                  <option v-for="ms in milestones" :key="ms.id" :value="ms.id">
                    {{ ms.title }}
                  </option>
                </select>
              </div>
            </div>

            <div class="modal-footer">
              <button type="button" class="btn-cancel" @click="emit('close')">Cancel</button>
              <button
                type="submit"
                class="btn-submit"
                :disabled="!form.title.trim() || !form.projectId || submitting"
              >
                <svg v-if="submitting" class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                </svg>
                Create Issue
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
  background: var(--overlay-strong);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  width: 560px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  box-shadow: var(--shadow-3);
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
  box-shadow: 0 0 0 2px color-mix(in oklab, var(--accent) 25%, transparent);
  outline: none;
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
  line-height: 1.5;
}

.form-select {
  cursor: pointer;
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-half {
  flex: 1;
}

.labels-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.label-chip {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.label-chip:hover {
  border-color: var(--text-secondary);
}

.label-chip.active {
  background: color-mix(in srgb, var(--label-color, var(--accent-blue)) 20%, transparent);
  border-color: var(--label-color, var(--accent-blue));
  color: var(--text-primary);
}

.empty-hint {
  font-size: 13px;
  color: var(--text-secondary);
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
  border: 1px solid color-mix(in oklab, var(--ok) 50%, transparent);
  background: var(--ok-2);
  color: var(--accent-green);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-submit:hover:not(:disabled) {
  background: color-mix(in oklab, var(--ok) 30%, transparent);
  border-color: var(--accent-green);
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

/* Transition */
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
