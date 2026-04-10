import { api, apiVoid, projectUrl, POST } from './client'
import type { Container } from '../types'

export const dockerApi = {
  containers: (project: string) =>
    api<Container[]>(`${projectUrl(project)}/docker/containers`),

  composeUp: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up`, POST),

  composeUpBuild: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up-build`, POST),

  composeDown: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/down`, POST),

  action: (project: string, name: string, action: string) =>
    apiVoid(`${projectUrl(project)}/docker/${name}/${action}`, POST),

  logsUrl: (project: string, name: string) =>
    `${projectUrl(project)}/docker/${name}/logs`,
}
