import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGitStore } from '../git'
import { useProjectsStore } from '../projects'

function setupStoreWithProject() {
  const projectsStore = useProjectsStore()
  projectsStore.currentProject = {
    name: 'myapp',
    path: '/srv/myapp',
    is_git: true,
    has_makefile: false,
    has_docker: false,
  }
  return useGitStore()
}

describe('useGitStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  describe('fetchStatus', () => {
    it('parses git status response', async () => {
      const statusData = {
        branch: 'main',
        modified: ['file1.ts', 'file2.ts'],
        staged: ['file3.ts'],
        untracked: ['newfile.ts'],
        ahead: 2,
        behind: 1,
      }

      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(statusData),
      }))

      const store = setupStoreWithProject()
      await store.fetchStatus()

      expect(store.status.branch).toBe('main')
      expect(store.status.modified).toEqual(['file1.ts', 'file2.ts'])
      expect(store.status.staged).toEqual(['file3.ts'])
      expect(store.status.untracked).toEqual(['newfile.ts'])
      expect(store.status.ahead).toBe(2)
      expect(store.status.behind).toBe(1)
      expect(fetch).toHaveBeenCalledWith('/api/projects/myapp/git/status', undefined)
    })

    it('handles missing fields with defaults', async () => {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ branch: 'dev' }),
      }))

      const store = setupStoreWithProject()
      await store.fetchStatus()

      expect(store.status.branch).toBe('dev')
      expect(store.status.modified).toEqual([])
      expect(store.status.staged).toEqual([])
      expect(store.status.untracked).toEqual([])
      expect(store.status.ahead).toBe(0)
      expect(store.status.behind).toBe(0)
    })
  })

  describe('fetchLog', () => {
    it('parses commits from metadata endpoint', async () => {
      const logData = [
        {
          hash: 'abc123def456',
          short_hash: 'abc123d',
          message: 'feat: add login',
          author: 'dev',
          date: '2025-01-01',
          refs: ['HEAD', 'main'],
        },
        {
          hash: 'bbb222ccc333',
          short_hash: 'bbb222c',
          message: 'fix: typo',
          author: 'dev2',
          date: '2024-12-31',
          refs: [],
        },
      ]

      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(logData),
      }))

      const store = setupStoreWithProject()
      await store.fetchLog()

      expect(store.log).toHaveLength(2)
      expect(store.log[0].hash).toBe('abc123def456')
      expect(store.log[0].refs).toEqual(['HEAD', 'main'])
      expect(store.log[0].parents).toEqual([])
      expect(store.log[1].short_hash).toBe('bbb222c')
      expect(store.log[1].refs).toEqual([])
      expect(store.log[1].parents).toEqual([])
      expect(fetch).toHaveBeenCalledWith('/api/projects/myapp/git/log/metadata?offset=0&limit=30', undefined)
    })
  })

  describe('selectedFiles management', () => {
    it('toggleSelectFile adds and removes files', () => {
      const store = setupStoreWithProject()

      store.toggleSelectFile('file1.ts')
      expect(store.selectedFiles.has('file1.ts')).toBe(true)

      store.toggleSelectFile('file1.ts')
      expect(store.selectedFiles.has('file1.ts')).toBe(false)
    })

    it('isSelected checks presence in Set', () => {
      const store = setupStoreWithProject()

      expect(store.isSelected('file1.ts')).toBe(false)
      store.toggleSelectFile('file1.ts')
      expect(store.isSelected('file1.ts')).toBe(true)
    })

    it('selectAllUnstaged selects all modified and untracked', () => {
      const store = setupStoreWithProject()
      store.status.modified = ['a.ts', 'b.ts']
      store.status.untracked = ['c.ts']

      store.selectAllUnstaged()

      expect(store.selectedFiles.has('a.ts')).toBe(true)
      expect(store.selectedFiles.has('b.ts')).toBe(true)
      expect(store.selectedFiles.has('c.ts')).toBe(true)
      expect(store.selectedFiles.size).toBe(3)
    })

    it('clearSelection empties the Set', () => {
      const store = setupStoreWithProject()
      store.toggleSelectFile('a.ts')
      store.toggleSelectFile('b.ts')
      expect(store.selectedFiles.size).toBe(2)

      store.clearSelection()

      expect(store.selectedFiles.size).toBe(0)
    })
  })

  describe('commit', () => {
    it('POSTs to /git/commit and clears commitMessage', async () => {
      const fetchMock = vi.fn()
        // commit POST
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
        // fetchStatus (called inside commit after success)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ branch: 'main', modified: [], staged: [], untracked: [], ahead: 0, behind: 0 }),
        })
        // fetchLog (called inside commit after success)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve([]),
        })

      vi.stubGlobal('fetch', fetchMock)

      const store = setupStoreWithProject()
      store.commitMessage = 'my commit message'

      await store.commit('my commit message', ['file.ts'])

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/projects/myapp/git/commit',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ message: 'my commit message', files: ['file.ts'] }),
        }),
      )
      expect(store.commitMessage).toBe('')
    })
  })
})
