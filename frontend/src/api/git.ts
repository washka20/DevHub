import { api, apiVoid, postJson, POST, DELETE } from './client'
import type { GitStatus, CommitDetail, BranchInfo, CommitMeta, StashEntry, BlameEntry } from '../types'

interface TopoNode {
  id: string
  parents: string[]
}

interface DiffResponse {
  diff: string
}

interface GenerateCommitResponse {
  message: string
}

export const gitApi = {
  status: (base: string) =>
    api<GitStatus>(`${base}/git/status`),

  branches: (base: string) =>
    api<BranchInfo[] | string[]>(`${base}/git/branches`),

  graph: (base: string) =>
    api<TopoNode[]>(`${base}/git/graph`),

  logMetadata: (base: string, offset: number, limit: number, branch?: string) => {
    let url = `${base}/git/log/metadata?offset=${offset}&limit=${limit}`
    if (branch) url += `&branch=${encodeURIComponent(branch)}`
    return api<CommitMeta[]>(url)
  },

  diff: (base: string, file?: string) => {
    const url = file
      ? `${base}/git/diff?file=${encodeURIComponent(file)}`
      : `${base}/git/diff`
    return api<DiffResponse>(url)
  },

  commitDetail: (base: string, hash: string) =>
    api<CommitDetail>(`${base}/git/commits/${hash}`),

  commitDiff: (base: string, hash: string, file?: string) => {
    const url = file
      ? `${base}/git/commits/${hash}/diff?file=${encodeURIComponent(file)}`
      : `${base}/git/commits/${hash}/diff`
    return api<DiffResponse>(url)
  },

  branchCommits: (base: string, branch: string, limit: number) =>
    api<CommitMeta[]>(`${base}/git/branches/${encodeURIComponent(branch)}/commits?limit=${limit}`),

  generateCommit: (base: string) =>
    api<GenerateCommitResponse>(`${base}/git/generate-commit`, POST),

  commit: (base: string, message: string, files: string[]) =>
    apiVoid(`${base}/git/commit`, postJson({ message, files })),

  checkout: (base: string, branch: string) =>
    apiVoid(`${base}/git/checkout`, postJson({ branch })),

  pull: (base: string) =>
    apiVoid(`${base}/git/pull`, POST),

  push: (base: string) =>
    apiVoid(`${base}/git/push`, POST),

  stage: (base: string, files: string[]) =>
    apiVoid(`${base}/git/stage`, postJson({ files })),

  unstage: (base: string, files: string[]) =>
    apiVoid(`${base}/git/unstage`, postJson({ files })),

  // Stash
  stashList: (base: string) =>
    api<StashEntry[]>(`${base}/git/stash`),

  stashPush: (base: string, message: string) =>
    apiVoid(`${base}/git/stash`, postJson({ message })),

  stashApply: (base: string, index: number) =>
    apiVoid(`${base}/git/stash/${index}/apply`, POST),

  stashPop: (base: string, index: number) =>
    apiVoid(`${base}/git/stash/${index}/pop`, POST),

  stashDrop: (base: string, index: number) =>
    apiVoid(`${base}/git/stash/${index}`, DELETE),

  stashDiff: (base: string, index: number) =>
    api<DiffResponse>(`${base}/git/stash/${index}/diff`),

  blame: (base: string, filePath: string) =>
    api<BlameEntry[]>(`${base}/git/blame?file=${encodeURIComponent(filePath)}`),
}
