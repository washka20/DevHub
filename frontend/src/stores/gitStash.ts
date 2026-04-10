import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { StashEntry } from '../types'

export const useGitStashStore = defineStore('gitStash', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const stashEntries = ref<StashEntry[]>([])
  const stashLoading = ref(false)

  async function fetchStash() {
    try {
      stashEntries.value = await gitApi.stashList(projectApiUrl.value)
    } catch { /* best-effort, non-critical */ }
  }

  async function stashPush(message: string) {
    stashLoading.value = true
    try {
      await gitApi.stashPush(projectApiUrl.value, message)
      await fetchStash()
      const { useGitStatusStore } = await import('./gitStatus')
      await useGitStatusStore().fetchStatus()
    } catch (e) {
      toast.show('error', `Stash push failed: ${getErrorMessage(e)}`)
    } finally {
      stashLoading.value = false
    }
  }

  async function stashApply(index: number) {
    try {
      await gitApi.stashApply(projectApiUrl.value, index)
      const { useGitStatusStore } = await import('./gitStatus')
      await useGitStatusStore().fetchStatus()
    } catch (e) {
      toast.show('error', `Stash apply failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashPop(index: number) {
    try {
      await gitApi.stashPop(projectApiUrl.value, index)
      await fetchStash()
      const { useGitStatusStore } = await import('./gitStatus')
      await useGitStatusStore().fetchStatus()
    } catch (e) {
      toast.show('error', `Stash pop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDrop(index: number) {
    try {
      await gitApi.stashDrop(projectApiUrl.value, index)
      await fetchStash()
    } catch (e) {
      toast.show('error', `Stash drop failed: ${getErrorMessage(e)}`)
    }
  }

  async function stashDiff(index: number): Promise<string> {
    try {
      const data = await gitApi.stashDiff(projectApiUrl.value, index)
      return data.diff || ''
    } catch {
      return ''
    }
  }

  return {
    stashEntries,
    stashLoading,
    fetchStash,
    stashPush,
    stashApply,
    stashPop,
    stashDrop,
    stashDiff,
  }
})
