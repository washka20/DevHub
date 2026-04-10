import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import type { Container } from '../types'

export const useDockerStore = defineStore('docker', () => {
  const containers = ref<Container[]>([])
  const selectedContainer = ref<string | null>(null)
  const loading = ref(false)
  const actionLoading = ref<string | null>(null)
  const composeLoading = ref<'up' | 'rebuild' | 'down' | null>(null)

  const projectsStore = useProjectsStore()
  const toast = useToast()

  function apiBase(): string {
    const project = projectsStore.currentProject
    if (!project) return '/api/projects/_'
    return `/api/projects/${project.name}`
  }

  const runningCount = computed(() =>
    containers.value.filter((c) => c.state === 'running').length
  )

  const totalCount = computed(() => containers.value.length)

  async function fetchContainers() {
    loading.value = true
    try {
      const res = await fetch(`${apiBase()}/docker/containers`)
      if (!res.ok) throw new Error(`Failed to fetch containers: ${res.statusText}`)
      containers.value = await res.json()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value = false
    }
  }

  async function containerAction(name: string, action: string) {
    actionLoading.value = name
    try {
      const res = await fetch(`${apiBase()}/docker/${name}/${action}`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error(`${action} failed: ${await res.text()}`)
      // Give Docker time to change state before refetching
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
      const res = await fetch(`${apiBase()}/docker/compose/up`, { method: 'POST' })
      if (!res.ok) throw new Error(`compose up failed: ${await res.text()}`)
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
      const res = await fetch(`${apiBase()}/docker/compose/up-build`, { method: 'POST' })
      if (!res.ok) throw new Error(`compose rebuild failed: ${await res.text()}`)
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
      const res = await fetch(`${apiBase()}/docker/compose/down`, { method: 'POST' })
      if (!res.ok) throw new Error(`compose down failed: ${await res.text()}`)
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
    return `${apiBase()}/docker/${name}/logs`
  }

  return {
    containers,
    selectedContainer,
    loading,
    actionLoading,
    composeLoading,
    runningCount,
    totalCount,
    fetchContainers,
    containerAction,
    composeUp,
    composeUpBuild,
    composeDown,
    selectContainer,
    logsUrl,
  }
})
