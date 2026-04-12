import { api, apiVoid, projectUrl, POST } from './client'
import type { Container, ContainerStats, ContainerInspect } from '../types'

export const dockerApi = {
  containers: (project: string) =>
    api<Container[]>(`${projectUrl(project)}/docker/containers`),

  stats: (project: string) =>
    api<ContainerStats[]>(`${projectUrl(project)}/docker/stats`),

  composeUp: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up`, POST),

  composeUpBuild: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/up-build`, POST),

  composeDown: (project: string) =>
    apiVoid(`${projectUrl(project)}/docker/compose/down`, POST),

  action: (project: string, name: string, action: string) =>
    apiVoid(`${projectUrl(project)}/docker/${name}/${action}`, POST),

  inspect: (project: string, name: string) =>
    api<ContainerInspect>(`${projectUrl(project)}/docker/${name}/inspect`),

  logsUrl: (project: string, name: string) =>
    `${projectUrl(project)}/docker/${name}/logs`,
}
