import { api, postJson, putJson } from './client'
import type {
  GitLabIssue,
  GitLabMR,
  GitLabMRApproval,
  GitLabNote,
  GitLabLabel,
  GitLabMilestone,
  GitLabMember,
  GitLabProject,
  GitLabPipeline,
  GitLabTodo,
} from '../types'

export const gitlabApi = {
  checkEnabled: () =>
    api<{ enabled: boolean }>('/api/gitlab/enabled'),

  currentUser: () =>
    api<GitLabMember>('/api/gitlab/user'),

  labels: () =>
    api<GitLabLabel[]>('/api/gitlab/labels'),

  milestones: () =>
    api<GitLabMilestone[]>('/api/gitlab/milestones'),

  myIssues: (state: string) =>
    api<GitLabIssue[]>(`/api/gitlab/my/issues?state=${state}`),

  myMergeRequests: (state: string) =>
    api<GitLabMR[]>(`/api/gitlab/my/merge-requests?state=${state}`),

  myReviewMRs: (state: string) =>
    api<GitLabMR[]>(`/api/gitlab/my/review-merge-requests?state=${state}`),

  myTodos: () =>
    api<GitLabTodo[]>('/api/gitlab/my/todos'),
  markTodoDone: (todoId: number) =>
    api<{ ok: boolean }>(`/api/gitlab/my/todos/${todoId}/done`, postJson({})),
  markAllTodosDone: () =>
    api<{ ok: boolean }>('/api/gitlab/my/todos/mark-all-done', postJson({})),

  // By GitLab project ID
  issueDetail: (pid: number, iid: number) =>
    api<GitLabIssue>(`/api/gitlab/projects/${pid}/issues/${iid}`),

  issueNotes: (pid: number, iid: number) =>
    api<GitLabNote[]>(`/api/gitlab/projects/${pid}/issues/${iid}/notes`),

  mrNotes: (pid: number, iid: number) =>
    api<GitLabNote[]>(`/api/gitlab/projects/${pid}/merge-requests/${iid}/notes`),

  addIssueNote: (pid: number, iid: number, body: string) =>
    api<GitLabNote>(`/api/gitlab/projects/${pid}/issues/${iid}/notes`, postJson({ body })),

  addMRNote: (pid: number, iid: number, body: string) =>
    api<GitLabNote>(`/api/gitlab/projects/${pid}/merge-requests/${iid}/notes`, postJson({ body })),

  createIssue: (pid: number, data: Record<string, unknown>) =>
    api<GitLabIssue>(`/api/gitlab/projects/${pid}/issues`, postJson(data)),

  createMR: (pid: number, data: Record<string, unknown>) =>
    api<GitLabMR>(`/api/gitlab/projects/${pid}/merge-requests`, postJson(data)),

  updateIssue: (pid: number, iid: number, data: Record<string, unknown>) =>
    api<GitLabIssue>(`/api/gitlab/projects/${pid}/issues/${iid}`, putJson(data)),

  mrApprovals: (pid: number, iid: number) =>
    api<GitLabMRApproval>(`/api/gitlab/projects/${pid}/merge-requests/${iid}/approvals`),

  approveMR: (pid: number, iid: number) =>
    api<{ ok: boolean }>(`/api/gitlab/projects/${pid}/merge-requests/${iid}/approve`, postJson({})),

  unapproveMR: (pid: number, iid: number) =>
    api<{ ok: boolean }>(`/api/gitlab/projects/${pid}/merge-requests/${iid}/unapprove`, postJson({})),

  projectMembers: (pid: number) =>
    api<GitLabMember[]>(`/api/gitlab/projects/${pid}/members`),

  // Per-DevHub-project endpoints
  project: (projectBase: string) =>
    api<GitLabProject>(`${projectBase}/gitlab/project`),

  projectIssues: (projectBase: string) =>
    api<GitLabIssue[]>(`${projectBase}/gitlab/issues`),

  projectMRs: (projectBase: string) =>
    api<GitLabMR[]>(`${projectBase}/gitlab/merge-requests`),

  projectPipelines: (projectBase: string) =>
    api<GitLabPipeline[]>(`${projectBase}/gitlab/pipelines`),
}
