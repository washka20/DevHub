import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useProjectsStore } from './projects'
import type { Container } from '../types'

export const useDockerStore = defineStore('docker', () => {
  const containers = ref<Container[]>([])
  const selectedContainer = ref<string | null>(null)
  const loading = ref(false)
  const actionLoading = ref<string | null>(null)
  const composeLoading = ref<'up' | 'rebuild' | 'down' | null>(null)

  const projectsStore = useProjectsStore()

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
      if (res.ok) {
        containers.value = await res.json()
      }
    } finally {
      loading.value = false
    }
  }

  async function containerAction(name: string, action: string) {
    actionLoading.value = name
    try {
      await fetch(`${apiBase()}/docker/${name}/${action}`, {
        method: 'POST',
      })
      // Give Docker time to change state before refetching
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } finally {
      actionLoading.value = null
    }
  }

  async function composeUp() {
    composeLoading.value = 'up'
    try {
      await fetch(`${apiBase()}/docker/compose/up`, { method: 'POST' })
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } finally {
      composeLoading.value = null
    }
  }

  async function composeUpBuild() {
    composeLoading.value = 'rebuild'
    try {
      await fetch(`${apiBase()}/docker/compose/up-build`, { method: 'POST' })
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
    } finally {
      composeLoading.value = null
    }
  }

  async function composeDown() {
    composeLoading.value = 'down'
    try {
      await fetch(`${apiBase()}/docker/compose/down`, { method: 'POST' })
      await new Promise((resolve) => setTimeout(resolve, 2000))
      await fetchContainers()
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
