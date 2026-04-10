import { api, apiRaw, projectUrl, putJson } from './client'

export const markdownApi = {
  listFiles: (project: string) =>
    api<string[]>(`${projectUrl(project)}/markdown`),

  getFile: (project: string, path: string) =>
    apiRaw(`${projectUrl(project)}/markdown/${path}`),

  toggleCheckbox: (project: string, path: string, line: number) =>
    apiRaw(`${projectUrl(project)}/markdown/${path}`, putJson({ line })),
}
