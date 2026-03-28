import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjectsStore } from '../projects'
import type { Project } from '../../types'

const mockProjects: Project[] = [
  { name: 'alpha', path: '/srv/alpha', is_git: true, has_makefile: true, has_docker: false },
  { name: 'beta', path: '/srv/beta', is_git: false, has_makefile: false, has_docker: true },
]

describe('useProjectsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('fetchProjects updates the projects list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockProjects),
    }))

    const store = useProjectsStore()
    await store.fetchProjects()

    expect(store.projects).toEqual(mockProjects)
    expect(fetch).toHaveBeenCalledWith('/api/projects')
  })

  it('fetchProjects sets default currentProject to first item', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockProjects),
    }))

    const store = useProjectsStore()
    expect(store.currentProject).toBeNull()

    await store.fetchProjects()

    expect(store.currentProject).toEqual(mockProjects[0])
  })

  it('fetchProjects does not overwrite existing currentProject', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockProjects),
    }))

    const store = useProjectsStore()
    // Manually set currentProject before fetch
    store.currentProject = mockProjects[1]

    await store.fetchProjects()

    expect(store.currentProject).toEqual(mockProjects[1])
  })

  it('setCurrentProject finds project by name and saves to localStorage', () => {
    const store = useProjectsStore()
    store.projects = [...mockProjects]

    store.setCurrentProject('beta')

    expect(store.currentProject).toEqual(mockProjects[1])
    expect(localStorage.getItem('devhub_current_project')).toBe('beta')
  })

  it('setCurrentProject does not change state when project not found', () => {
    const store = useProjectsStore()
    store.projects = [...mockProjects]
    store.currentProject = mockProjects[0]

    store.setCurrentProject('nonexistent')

    expect(store.currentProject).toEqual(mockProjects[0])
    expect(localStorage.getItem('devhub_current_project')).toBeNull()
  })
})
