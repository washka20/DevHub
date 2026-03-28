export interface Project {
  name: string
  path: string
  is_git: boolean
  has_makefile: boolean
  has_docker: boolean
}

export interface Container {
  name: string
  image: string
  status: string
  ports: string
  state: string
}

export interface GitStatus {
  branch: string
  modified: string[]
  staged: string[]
  untracked: string[]
  ahead: number
  behind: number
}

export interface GraphLine {
  x1: number
  x2: number
  type: number  // 0=Bottom, 1=Top, 2=Full, 3=Fork, 4=MergeBack
  color: string
}

export interface GraphData {
  column: number
  color: string
  lines: GraphLine[]
}

export interface Commit {
  hash: string
  short_hash: string
  message: string
  author: string
  date: string
  refs: string[]
  parents: string[]
  graph?: string
  graph_only?: boolean
  graph_data?: GraphData
}

export interface CommitDetail {
  hash: string
  message: string
  author: string
  email: string
  date: string
  body: string
  files: FileChange[]
  stats: string
}

export interface FileChange {
  status: string
  path: string
}

export interface BranchInfo {
  name: string
  short_hash: string
  message: string
  author: string
  date: string
  is_current: boolean
  ahead: number
  behind: number
  is_merged: boolean
}

export interface DiffLine {
  type: 'add' | 'remove' | 'context' | 'header'
  content: string
  oldLineNo: number | null
  newLineNo: number | null
}

export interface DiffHunk {
  header: string
  lines: DiffLine[]
}

export interface MakeCommand {
  name: string
  description: string
  category: string
}
