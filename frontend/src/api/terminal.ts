import { api, apiVoid, postJson, DELETE } from './client'

interface CreateSessionResponse {
  session_id: string
  shell: string
}

interface LiveSession {
  id: string
}

export const terminalApi = {
  createSession: (cwd: string, cols: number, rows: number) =>
    api<CreateSessionResponse>('/api/terminal/sessions', postJson({ cols, rows, cwd })),

  destroySession: (id: string) =>
    apiVoid(`/api/terminal/sessions/${id}`, DELETE),

  getSession: (id: string) =>
    api<unknown>(`/api/terminal/sessions/${id}`),

  listSessions: () =>
    api<LiveSession[]>('/api/terminal/sessions'),
}
