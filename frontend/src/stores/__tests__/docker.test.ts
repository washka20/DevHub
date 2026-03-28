import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDockerStore } from '../docker'
import { useProjectsStore } from '../projects'
import type { Container } from '../../types'

const mockContainers: Container[] = [
  { name: 'app', image: 'node:20', status: 'Up 3 hours', ports: '3000:3000', state: 'running' },
  { name: 'db', image: 'postgres:16', status: 'Up 3 hours', ports: '5432:5432', state: 'running' },
  { name: 'redis', image: 'redis:7', status: 'Exited (0) 1 hour ago', ports: '', state: 'exited' },
]

describe('useDockerStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()

    // Set current project so apiBase() returns the right URL
    const projectsStore = useProjectsStore()
    projectsStore.currentProject = {
      name: 'myapp',
      path: '/srv/myapp',
      is_git: true,
      has_makefile: true,
      has_docker: true,
    }
  })

  describe('fetchContainers', () => {
    it('parses Container[] response', async () => {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockContainers),
      }))

      const store = useDockerStore()
      await store.fetchContainers()

      expect(store.containers).toEqual(mockContainers)
      expect(store.containers).toHaveLength(3)
      expect(fetch).toHaveBeenCalledWith('/api/projects/myapp/docker/containers')
    })
  })

  describe('containerAction', () => {
    it('POSTs action and refetches containers after delay', async () => {
      vi.useFakeTimers()

      const fetchMock = vi.fn()
        // POST action
        .mockResolvedValueOnce({ ok: true })
        // refetch containers after delay
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve(mockContainers),
        })

      vi.stubGlobal('fetch', fetchMock)

      const store = useDockerStore()
      const actionPromise = store.containerAction('app', 'restart')

      // Advance past the 2s setTimeout
      await vi.advanceTimersByTimeAsync(2000)
      await actionPromise

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/projects/myapp/docker/app/restart',
        expect.objectContaining({ method: 'POST' }),
      )
      // Second call is the refetch
      expect(fetchMock).toHaveBeenCalledWith('/api/projects/myapp/docker/containers')
      expect(fetchMock).toHaveBeenCalledTimes(2)

      vi.useRealTimers()
    })
  })

  describe('computed properties', () => {
    it('runningCount counts only running containers', () => {
      const store = useDockerStore()
      store.containers = [...mockContainers]

      expect(store.runningCount).toBe(2)
    })

    it('totalCount counts all containers', () => {
      const store = useDockerStore()
      store.containers = [...mockContainers]

      expect(store.totalCount).toBe(3)
    })

    it('runningCount is 0 when no containers', () => {
      const store = useDockerStore()
      store.containers = []

      expect(store.runningCount).toBe(0)
      expect(store.totalCount).toBe(0)
    })
  })
})
