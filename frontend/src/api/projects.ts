import { api, apiRaw, apiVoid, projectUrl, postJson, putJson } from './client'
import type { Project, MakeCommand, ServerSettings } from '../types'

export const projectsApi = {
  list: () =>
    api<Project[]>('/api/projects'),

  commands: (project: string) =>
    api<MakeCommand[]>(`${projectUrl(project)}/commands`),

  exec: (project: string, cmd: string) =>
    apiRaw(`${projectUrl(project)}/exec`, postJson({ cmd })),
}

export const settingsApi = {
  fetch: () =>
    api<ServerSettings>('/api/settings'),

  save: (updates: Partial<ServerSettings>) =>
    apiVoid('/api/settings', putJson(updates)),

  shells: () =>
    api<string[]>('/api/settings/shells'),
}
