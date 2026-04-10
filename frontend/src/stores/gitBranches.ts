import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { BranchInfo, CommitMeta } from '../types'

export const useGitBranchesStore = defineStore('gitBranches', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const branches = ref<BranchInfo[]>([])
  const branchCommits = ref<Map<string, CommitMeta[]>>(new Map())
  const loadingBranches = ref(false)
  const loadingCheckout = ref(false)

  async function fetchBranches() {
    loadingBranches.value = true
    try {
      const data = await gitApi.branches(projectApiUrl.value)
      if (Array.isArray(data) && data.length > 0) {
        if (typeof data[0] === 'string') {
          const { useGitStatusStore } = await import('./gitStatus')
          const statusStore = useGitStatusStore()
          branches.value = (data as string[]).map((name: string) => ({
            name,
            short_hash: '',
            message: '',
            author: '',
            date: '',
            is_current: name === statusStore.status.branch,
            ahead: 0,
            behind: 0,
            is_merged: false,
          }))
        } else {
          branches.value = data as BranchInfo[]
        }
      } else {
        branches.value = []
      }
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingBranches.value = false
    }
  }

  async function checkout(branch: string) {
    loadingCheckout.value = true
    try {
      await gitApi.checkout(projectApiUrl.value, branch)
      const { useGitStatusStore } = await import('./gitStatus')
      const { useGitLogStore } = await import('./gitLog')
      await Promise.all([
        useGitStatusStore().fetchStatus(),
        fetchBranches(),
        useGitLogStore().fetchGraph(),
      ])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingCheckout.value = false
    }
  }

  async function fetchBranchCommits(branch: string) {
    try {
      const data = await gitApi.branchCommits(projectApiUrl.value, branch, 5)
      const map = new Map(branchCommits.value)
      map.set(branch, data)
      branchCommits.value = map
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    }
  }

  return {
    branches,
    branchCommits,
    loadingBranches,
    loadingCheckout,
    fetchBranches,
    checkout,
    fetchBranchCommits,
  }
})
