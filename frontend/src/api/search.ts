import { api, projectUrl } from './client'

export interface SearchResult {
  file: string
  line: number
  column: number
  content: string
}

export const searchApi = {
  search: (project: string, query: string, glob?: string) =>
    api<SearchResult[]>(
      `${projectUrl(project)}/files/search?q=${encodeURIComponent(query)}${glob ? '&glob=' + encodeURIComponent(glob) : ''}`,
    ),
}
