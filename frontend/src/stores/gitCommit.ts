import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'

export const useGitCommitStore = defineStore('gitCommit', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const commitMessage = ref('')
  const generatingMessage = ref(false)
  const loadingCommit = ref(false)
  const loadingPull = ref(false)
  const loadingPush = ref(false)

  async function generateCommitMessage() {
    generatingMessage.value = true
    try {
      const data = await gitApi.generateCommit(projectApiUrl.value)
      commitMessage.value = data.message ?? ''
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      generatingMessage.value = false
    }
  }

  async function commit(message: string, files: string[]) {
    loadingCommit.value = true
    try {
      await gitApi.commit(projectApiUrl.value, message, files)
      commitMessage.value = ''
      const { useGitStatusStore } = await import('./gitStatus')
      const { useGitLogStore } = await import('./gitLog')
      await Promise.all([
        useGitStatusStore().fetchStatus(),
        useGitLogStore().fetchGraph(),
      ])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingCommit.value = false
    }
  }

  async function pull() {
    loadingPull.value = true
    try {
      await gitApi.pull(projectApiUrl.value)
      const { useGitStatusStore } = await import('./gitStatus')
      const { useGitLogStore } = await import('./gitLog')
      await Promise.all([
        useGitStatusStore().fetchStatus(),
        useGitLogStore().fetchGraph(),
      ])
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingPull.value = false
    }
  }

  async function push() {
    loadingPush.value = true
    try {
      await gitApi.push(projectApiUrl.value)
      const { useGitStatusStore } = await import('./gitStatus')
      await useGitStatusStore().fetchStatus()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingPush.value = false
    }
  }

  return {
    commitMessage,
    generatingMessage,
    loadingCommit,
    loadingPull,
    loadingPush,
    generateCommitMessage,
    commit,
    pull,
    push,
  }
})
