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

const emptyComposeInfo = { files: [], default_files: [] }

/**
 * The docker store installs a watch that calls `fetchComposeInfo()` immediately
 * when the project is set. Tests below stub that very first fetch so it
 * resolves harmlessly, then assert the subsequent calls we actually care about.
 */
function stubFetchInOrder(responses: unknown[]): ReturnType<typeof vi.fn> {
  const fn = vi.fn()
  for (const r of responses) {
    fn.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(r) })
  }
  vi.stubGlobal('fetch', fn)
  return fn
}

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
      // 1st fetch = watcher-triggered fetchComposeInfo, 2nd = fetchContainers
      const fetchMock = stubFetchInOrder([emptyComposeInfo, mockContainers])

      const store = useDockerStore()
      await store.fetchContainers()

      expect(store.containers).toEqual(mockContainers)
      expect(store.containers).toHaveLength(3)
      expect(fetchMock).toHaveBeenCalledWith('/api/projects/myapp/docker/containers', undefined)
    })
  })

  describe('containerAction', () => {
    it('POSTs action and refetches containers after delay', async () => {
      vi.useFakeTimers()

      const fetchMock = vi.fn()
        // watcher: fetchComposeInfo
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(emptyComposeInfo) })
        // POST action
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(null) })
        // refetch containers after delay
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(mockContainers) })

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
      expect(fetchMock).toHaveBeenCalledWith('/api/projects/myapp/docker/containers', undefined)
      // 3 calls: compose info + POST + refetch
      expect(fetchMock).toHaveBeenCalledTimes(3)

      vi.useRealTimers()
    })
  })

  describe('computed properties', () => {
    beforeEach(() => {
      stubFetchInOrder([emptyComposeInfo])
    })

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
