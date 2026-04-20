import { api, apiVoid, projectUrl, POST } from './client'
import type {
  Container,
  ContainerStats,
  ContainerInspect,
  ComposeInfo,
  DockerAllResponse,
} from '../types'

/** Files + profiles the caller wants `docker compose -f ... --profile ...` to use. */
export interface StackParams {
  files?: string[]
  profiles?: string[]
}

function stackQuery(s?: StackParams): string {
  if (!s) return ''
  const params = new URLSearchParams()
  for (const f of s.files ?? []) if (f) params.append('files', f)
  for (const p of s.profiles ?? []) if (p) params.append('profiles', p)
  const q = params.toString()
  return q ? `?${q}` : ''
}

export const dockerApi = {
  composeInfo: (project: string) =>
    api<ComposeInfo>(`${projectUrl(project)}/docker/compose`),

  containers: (project: string, stack?: StackParams) =>
    api<Container[]>(`${projectUrl(project)}/docker/containers${stackQuery(stack)}`),

  stats: (project: string, stack?: StackParams) =>
    api<ContainerStats[]>(`${projectUrl(project)}/docker/stats${stackQuery(stack)}`),

  composeUp: (project: string, stack?: StackParams) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up${stackQuery(stack)}`, POST),

  composeUpBuild: (project: string, stack?: StackParams) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up-build${stackQuery(stack)}`, POST),

  composeDown: (project: string, stack?: StackParams) =>
    apiVoid(`${projectUrl(project)}/docker/compose/down${stackQuery(stack)}`, POST),

  action: (project: string, name: string, action: string, stack?: StackParams) =>
    apiVoid(`${projectUrl(project)}/docker/${name}/${action}${stackQuery(stack)}`, POST),

  inspect: (project: string, name: string, stack?: StackParams) =>
    api<ContainerInspect>(`${projectUrl(project)}/docker/${name}/inspect${stackQuery(stack)}`),

  logsUrl: (project: string, name: string, stack?: StackParams) =>
    `${projectUrl(project)}/docker/${name}/logs${stackQuery(stack)}`,

  // --- Global scope (not tied to a project) ---

  allContainers: () =>
    api<DockerAllResponse>(`/api/docker/all`),

  globalAction: (id: string, action: 'start' | 'stop' | 'restart' | 'kill' | 'remove') =>
    apiVoid(`/api/docker/containers/${id}/${action}`, POST),

  globalLogsUrl: (id: string) =>
    `/api/docker/containers/${id}/logs`,
}
