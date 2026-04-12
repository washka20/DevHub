import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { dockerApi } from '../api/docker'
import type { Container, ContainerStats } from '../types'

export const useDockerStore = defineStore('docker', () => {
  const containers = ref<Container[]>([])
  const stats = ref<ContainerStats[]>([])
  const selectedContainer = ref<string | null>(null)
  const loading = ref(false)
  const actionLoading = ref<string | null>(null)
  const composeLoading = ref<'up' | 'rebuild' | 'down' | null>(null)
  let statsInterval: ReturnType<typeof setInterval> | null = null

  const projectsStore = useProjectsStore()
  const toast = useToast()

  function projectName(): string {
    return projectsStore.currentProject?.name ?? '_'
  }

  const runningCount = computed(() =>
    (containers.value ?? []).filter((c) => c.state === 'running').length
  )

  const totalCount = computed(() => (containers.value ?? []).length)

  async function fetchContainers() {
    loading.value = true
    try {
      containers.value = await dockerApi.containers(projectName()) ?? []
    } catch (e) {
      containers.value = []
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value = false
    }
  }

  async function fetchStats() {
    try {
      stats.value = await dockerApi.stats(projectName()) ?? []
    } catch {
      stats.value = []
    }
  }

  function startStatsPolling() {
    stopStatsPolling()
    fetchStats()
    statsInterval = setInterval(fetchStats, 5000)
  }

  function stopStatsPolling() {
    if (statsInterval) {
      clearInterval(statsInterval)
      statsInterval = null
    }
    stats.value = []
  }

  function statsForContainer(name: string): ContainerStats | undefined {
    return stats.value.find((s) => s.name.includes(name))
  }

  async function containerAction(name: string, action: string) {
    actionLoading.value = name
    try {
      await dockerApi.action(projectName(), name, action)
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      actionLoading.value = null
    }
  }

  async function composeUp() {
    composeLoading.value = 'up'
    try {
      await dockerApi.composeUp(projectName())
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      composeLoading.value = null
    }
  }

  async function composeUpBuild() {
    composeLoading.value = 'rebuild'
    try {
      await dockerApi.composeUpBuild(projectName())
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      composeLoading.value = null
    }
  }

  async function composeDown() {
    composeLoading.value = 'down'
    try {
      await dockerApi.composeDown(projectName())
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      composeLoading.value = null
    }
  }

  function selectContainer(name: string | null) {
    selectedContainer.value = name
  }

  function logsUrl(name: string): string {
    return dockerApi.logsUrl(projectName(), name)
  }

  return {
    containers,
    stats,
    selectedContainer,
    loading,
    actionLoading,
    composeLoading,
    runningCount,
    totalCount,
    fetchContainers,
    fetchStats,
    startStatsPolling,
    stopStatsPolling,
    statsForContainer,
    containerAction,
    composeUp,
    composeUpBuild,
    composeDown,
    selectContainer,
    logsUrl,
  }
})
