import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { CommitDetail } from '../types'

export const useGitDiffStore = defineStore('gitDiff', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const selectedCommit = ref<CommitDetail | null>(null)
  const loadingCommitDetail = ref(false)
  const loadingCommitDiff = ref(false)

  async function fetchCommitDetail(hash: string) {
    loadingCommitDetail.value = true
    try {
      const data = await gitApi.commitDetail(projectApiUrl.value, hash)
      selectedCommit.value = data as CommitDetail
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingCommitDetail.value = false
    }
  }

  async function fetchCommitDiff(hash: string, file?: string) {
    loadingCommitDiff.value = true
    try {
      const { useGitStatusStore } = await import('./gitStatus')
      const statusStore = useGitStatusStore()
      const data = await gitApi.commitDiff(projectApiUrl.value, hash, file)
      statusStore.diff = data.diff ?? ''
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingCommitDiff.value = false
    }
  }

  return {
    selectedCommit,
    loadingCommitDetail,
    loadingCommitDiff,
    fetchCommitDetail,
    fetchCommitDiff,
  }
})
