import { api, apiText, apiVoid, apiRaw, projectUrl, postJson, patchJson, DELETE } from './client'
import type { FileNode } from '../types'

export const filesApi = {
  tree: (project: string) =>
    api<FileNode[]>(`${projectUrl(project)}/files/tree`),

  content: (project: string, path: string) =>
    apiText(`${projectUrl(project)}/files/content/${encodeURIComponent(path)}`),

  save: (project: string, path: string, content: string) =>
    apiRaw(`${projectUrl(project)}/files/content/${encodeURIComponent(path)}`, {
      method: 'PUT',
      body: content,
    }),

  create: (project: string, path: string, isDir: boolean) =>
    apiVoid(`${projectUrl(project)}/files/create`, postJson({ path, is_dir: isDir })),

  delete: (project: string, path: string) =>
    apiVoid(`${projectUrl(project)}/files/delete/${encodeURIComponent(path)}`, DELETE),

  rename: (project: string, oldPath: string, newPath: string) =>
    apiVoid(`${projectUrl(project)}/files/rename/${encodeURIComponent(oldPath)}`, patchJson({ new_path: newPath })),

  openInFileManager: (project: string, path: string) =>
    apiVoid(`${projectUrl(project)}/open-in-fm`, postJson({ path })),
}
