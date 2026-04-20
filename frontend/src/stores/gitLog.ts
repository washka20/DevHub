import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProject } from '../composables/useProject'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { gitApi } from '../api/git'
import type { Commit, CommitMeta } from '../types'

interface TopoNode {
  id: string
  parents: string[]
}

const LOG_PAGE_SIZE = 50

export const useGitLogStore = defineStore('gitLog', () => {
  const { projectApiUrl } = useProject()
  const toast = useToast()

  const log = ref<Commit[]>([])
  const loadingLog = ref(false)

  // Branch browsing state
  const viewingBranch = ref<string>('')

  // Graph + metadata
  const topoNodes = ref<TopoNode[]>([])
  const graphNodes = computed(() => topoNodes.value)
  const metadataMap = ref<Map<string, CommitMeta>>(new Map())
  const metadataLoaded = ref(0)
  const metadataLoading = ref(false)
  const totalCommits = computed(() => topoNodes.value.length)

  async function fetchLog(limit = 30) {
    loadingLog.value = true
    try {
      const data = await gitApi.logMetadata(projectApiUrl.value, 0, limit)
      log.value = (data ?? []).map(m => ({
        hash: m.hash,
        short_hash: m.short_hash,
        message: m.message,
        author: m.author,
        date: m.date,
        refs: m.refs,
        parents: [],
      }))
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingLog.value = false
    }
  }

  async function fetchGraph() {
    loadingLog.value = true
    try {
      const data = await gitApi.graph(projectApiUrl.value)
      topoNodes.value = (data ?? []).map(n => ({ id: n.id, parents: n.parents ?? [] }))
      metadataMap.value = new Map()
      metadataLoaded.value = 0
      await fetchMetadata(0, LOG_PAGE_SIZE)
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loadingLog.value = false
    }
  }

  async function fetchMetadata(offset: number, limit: number) {
    if (metadataLoading.value) return
    metadataLoading.value = true
    try {
      const data = await gitApi.logMetadata(projectApiUrl.value, offset, limit, viewingBranch.value || undefined)
      const map = new Map(metadataMap.value)
      for (const m of data) {
        map.set(m.hash, m)
      }
      metadataMap.value = map
      metadataLoaded.value = offset + data.length
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      metadataLoading.value = false
    }
  }

  function getMetadata(hash: string): CommitMeta | undefined {
    return metadataMap.value.get(hash)
  }

  function setViewingBranch(branch: string) {
    viewingBranch.value = branch
    metadataMap.value = new Map()
    metadataLoaded.value = 0
    fetchMetadata(0, 50)
  }

  return {
    log,
    loadingLog,
    viewingBranch,
    graphNodes,
    metadataMap,
    metadataLoaded,
    metadataLoading,
    totalCommits,
    fetchLog,
    fetchGraph,
    fetchMetadata,
    getMetadata,
    setViewingBranch,
  }
})
