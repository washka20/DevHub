export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

export async function api<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new ApiError(res.status, text)
  }
  return res.json()
}

export async function apiText(url: string, options?: RequestInit): Promise<string> {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new ApiError(res.status, text)
  }
  return res.text()
}

export async function apiVoid(url: string, options?: RequestInit): Promise<void> {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new ApiError(res.status, text)
  }
}

/**
 * Like apiVoid, but returns the raw Response for callers that need
 * to inspect status/headers (e.g. 202 Accepted).
 */
export async function apiRaw(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, options)
  if (!res.ok) {
    const text = await res.text()
    throw new ApiError(res.status, text)
  }
  return res
}

export function projectUrl(project: string): string {
  return `/api/projects/${project}`
}

export function postJson(body: unknown): RequestInit {
  return {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }
}

export function putJson(body: unknown): RequestInit {
  return {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }
}

export function patchJson(body: unknown): RequestInit {
  return {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }
}

export const DELETE: RequestInit = { method: 'DELETE' }
export const POST: RequestInit = { method: 'POST' }
