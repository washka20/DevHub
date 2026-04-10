import { api, apiText, apiVoid, projectUrl, postJson, putJson, DELETE } from './client'

interface NoteCreated {
  slug: string
}

export const notesApi = {
  list: (project: string) =>
    api<Array<{ slug: string; title: string; updated_at: string }>>(`${projectUrl(project)}/notes`),

  get: (project: string, slug: string) =>
    apiText(`${projectUrl(project)}/notes/${slug}`),

  create: (project: string, title: string) =>
    api<NoteCreated>(`${projectUrl(project)}/notes`, postJson({ title })),

  save: (project: string, slug: string, content: string) =>
    apiVoid(`${projectUrl(project)}/notes/${slug}`, putJson({ content })),

  delete: (project: string, slug: string) =>
    apiVoid(`${projectUrl(project)}/notes/${slug}`, DELETE),
}
