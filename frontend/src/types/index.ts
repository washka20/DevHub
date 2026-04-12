export interface Project {
  name: string
  path: string
  is_git: boolean
  has_makefile: boolean
  has_docker: boolean
  group?: string
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

export interface GraphNodeOut {
  id: string
  parents: string[]
  graph_data: GraphData
}

export interface FullGraphResponse {
  nodes: GraphNodeOut[]
  max_width: number
}

export interface CommitMeta {
  hash: string
  short_hash: string
  message: string
  author: string
  date: string
  refs: string[]
}

export interface MakeCommand {
  name: string
  description: string
  category: string
}

export interface TerminalSession {
  id: string
  label: string
  cwd: string
}

export interface TerminalPane {
  id: string
  sessionId: string | null
  cwd: string
  status: 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
  hasActivity?: boolean
  hasBell?: boolean
}

export interface TerminalTab {
  id: string
  label: string
  panes: TerminalPane[]
  splitDirection: 'horizontal' | 'vertical' | null
}

export interface PanelState {
  mode: 'pinned' | 'floating'
  visible: boolean
  height: number
  floatingPos: { x: number; y: number; w: number; h: number }
}

export interface PersistedLayout {
  tabs: Array<{
    id: string
    label: string
    panes: Array<{ id: string; cwd: string; sessionId?: string | null; label?: string }>
    direction: 'horizontal' | 'vertical' | null
  }>
  activeTabId: string | null
  panel: PanelState
}

// Settings
export interface ServerSettings {
  port: number
  projects_dir: string
  default_project: string
  terminal: {
    max_sessions: number
    shell: string
  }
}

export interface TerminalTheme {
  background: string
  foreground: string
  cursor: string
  selectionBackground: string
  black: string
  red: string
  green: string
  yellow: string
  blue: string
  magenta: string
  cyan: string
  white: string
  brightBlack: string
  brightRed: string
  brightGreen: string
  brightYellow: string
  brightBlue: string
  brightMagenta: string
  brightCyan: string
  brightWhite: string
}

export interface UISettings {
  fontSize: number
  fontFamily: string
  scrollback: number
  cursorBlink: boolean
  themeName: string
  siteThemeName: string
  editorEngine: 'codemirror' | 'monaco'
  editorKeymap: 'default' | 'vim'
  editorMinimap: boolean
  editorFontSize: number
}

export interface StashEntry {
  index: number
  message: string
  date: string
}

export interface FileNode {
  name: string
  path: string
  is_dir: boolean
  children?: FileNode[]
}

export interface OpenFile {
  path: string
  name: string
  content: string
  originalContent: string
  dirty: boolean
  language: string
}

// GitLab types
export interface GitLabAuthor {
  id: number
  username: string
  name: string
  avatar_url: string
}

export interface GitLabLabel {
  id: number
  name: string
  color: string
}

export interface GitLabMilestone {
  id: number
  title: string
  state: string
}

export interface GitLabMember {
  id: number
  username: string
  name: string
  avatar_url: string
}

export interface GitLabNote {
  id: number
  body: string
  author: GitLabAuthor
  created_at: string
  system: boolean
}

export interface GitLabProject {
  id: number
  name: string
  path_with_namespace: string
  web_url: string
  description: string
}

export interface GitLabLabelDetail {
  name: string
  color: string
}

export interface GitLabIssue {
  id: number
  iid: number
  project_id: number
  title: string
  description: string
  state: 'opened' | 'closed'
  author: GitLabAuthor
  assignees: GitLabAuthor[]
  labels: string[]
  label_details?: GitLabLabelDetail[]
  milestone: GitLabMilestone | null
  due_date: string | null
  created_at: string
  updated_at: string
  web_url: string
  project_path: string
  references: { full: string }
}

export interface GitLabMR {
  id: number
  iid: number
  project_id: number
  title: string
  description: string
  state: 'opened' | 'merged' | 'closed'
  author: GitLabAuthor
  assignees: GitLabAuthor[]
  reviewers: GitLabAuthor[]
  labels: string[]
  label_details?: GitLabLabelDetail[]
  source_branch: string
  target_branch: string
  draft: boolean
  merge_status: string
  created_at: string
  updated_at: string
  merged_at: string | null
  web_url: string
  project_path: string
  references: { full: string }
  pipeline: GitLabPipeline | null
}

export interface GitLabPipeline {
  id: number
  status: 'created' | 'waiting_for_resource' | 'preparing' | 'pending' | 'running' | 'success' | 'failed' | 'canceled' | 'skipped' | 'manual' | 'scheduled'
  ref: string
  sha: string
  web_url: string
  created_at: string
  updated_at: string
}
