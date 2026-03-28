import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProject } from '../useProject'
import { useProjectsStore } from '../../stores/projects'
import type { Project } from '../../types'

const mockProjects: Project[] = [
  { name: 'alpha', path: '/srv/alpha', is_git: true, has_makefile: true, has_docker: true },
  { name: 'beta', path: '/srv/beta', is_git: false, has_makefile: false, has_docker: false },
]

describe('useProject', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    localStorage.clear()

    // Stub fetch globally so stores that call fetch on init do not break
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))
  })

  describe('projectApiUrl', () => {
    it('returns base URL when no project is selected', () => {
      const { projectApiUrl } = useProject()
      expect(projectApiUrl.value).toBe('/api/projects')
    })

    it('returns project-specific URL when project is selected', () => {
      const store = useProjectsStore()
      store.currentProject = mockProjects[0]

      const { projectApiUrl } = useProject()
      expect(projectApiUrl.value).toBe('/api/projects/alpha')
    })
  })

  describe('switchProject', () => {
    it('updates the store current project', async () => {
      const store = useProjectsStore()
      store.projects = [...mockProjects]
      store.currentProject = mockProjects[0]

      const { switchProject } = useProject()
      await switchProject('beta')

      expect(store.currentProject).toEqual(mockProjects[1])
    })
  })

  describe('initProject', () => {
    it('fetches projects and restores saved project from localStorage', async () => {
      localStorage.setItem('devhub_current_project', 'beta')

      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockProjects),
      })
      vi.stubGlobal('fetch', fetchMock)

      const { initProject } = useProject()
      await initProject()

      const store = useProjectsStore()
      expect(store.projects).toEqual(mockProjects)
      expect(store.currentProject).toEqual(mockProjects[1])
    })

    it('uses default first project when localStorage has no saved value', async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockProjects),
      })
      vi.stubGlobal('fetch', fetchMock)

      const { initProject } = useProject()
      await initProject()

      const store = useProjectsStore()
      // fetchProjects sets first project as default
      expect(store.currentProject).toEqual(mockProjects[0])
    })

    it('ignores saved project name if it does not exist in the list', async () => {
      localStorage.setItem('devhub_current_project', 'nonexistent')

      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockProjects),
      })
      vi.stubGlobal('fetch', fetchMock)

      const { initProject } = useProject()
      await initProject()

      const store = useProjectsStore()
      // Falls back to default (first project set by fetchProjects)
      expect(store.currentProject).toEqual(mockProjects[0])
    })
  })
})
