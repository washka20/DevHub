import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { useProjectsStore } from './projects'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import { dockerApi, type StackParams } from '../api/docker'
import type {
  Container,
  ContainerStats,
  ContainerInspect,
  ComposeInfo,
  DockerAllGroup,
} from '../types'

type ScopeTab = 'project' | 'all'

/**
 * Per-project persisted stack selection.
 * Stored under `devhub.docker.<projectName>.stack` so switching projects
 * doesn't overwrite each other's preferences.
 */
interface PersistedStack {
  files: string[]
  profiles: string[]
}

function stackStorageKey(projectName: string): string {
  return `devhub.docker.${projectName}.stack`
}

function loadStack(projectName: string): PersistedStack | null {
  try {
    const raw = localStorage.getItem(stackStorageKey(projectName))
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && Array.isArray(parsed.files) && Array.isArray(parsed.profiles)) {
      return { files: parsed.files, profiles: parsed.profiles }
    }
  } catch { /* ignore */ }
  return null
}

function saveStack(projectName: string, stack: PersistedStack) {
  try {
    localStorage.setItem(stackStorageKey(projectName), JSON.stringify(stack))
  } catch { /* ignore */ }
}

export const useDockerStore = defineStore('docker', () => {
  const containers = ref<Container[]>([])
  const stats = ref<ContainerStats[]>([])
  const selectedContainer = ref<string | null>(null)
  const loading = ref(false)
  const actionLoading = ref<string | null>(null)
  const composeLoading = ref<'up' | 'rebuild' | 'down' | null>(null)
  const expandedContainer = ref<string | null>(null)
  const inspectData = ref<ContainerInspect | null>(null)
  const inspectLoading = ref(false)
  let statsInterval: ReturnType<typeof setInterval> | null = null

  // --- Compose stack (Stage 15) ---
  const composeInfo = ref<ComposeInfo | null>(null)
  const selectedFiles = ref<string[]>([])
  const selectedProfiles = ref<string[]>([])

  // --- Scope tab (project vs all) ---
  const SCOPE_KEY = 'devhub.docker.scope'
  const scopeTab = ref<ScopeTab>(
    (typeof localStorage !== 'undefined' && (localStorage.getItem(SCOPE_KEY) as ScopeTab)) || 'project',
  )
  function setScopeTab(tab: ScopeTab) {
    scopeTab.value = tab
    try { localStorage.setItem(SCOPE_KEY, tab) } catch {}
  }

  // --- Global scope data ---
  const allGroups = ref<DockerAllGroup[]>([])
  const allLoading = ref(false)
  const allActionLoading = ref<string | null>(null)
  let allInterval: ReturnType<typeof setInterval> | null = null

  const projectsStore = useProjectsStore()
  const toast = useToast()

  function projectName(): string {
    return projectsStore.currentProject?.name ?? '_'
  }

  /** Helper: build the stack params we send with every docker request. */
  function stackParams(): StackParams | undefined {
    if (!selectedFiles.value.length && !selectedProfiles.value.length) return undefined
    return { files: selectedFiles.value, profiles: selectedProfiles.value }
  }

  const runningCount = computed(() =>
    (containers.value ?? []).filter((c) => c.state === 'running').length,
  )

  const totalCount = computed(() => (containers.value ?? []).length)

  const globalRunningCount = computed(() => {
    let n = 0
    for (const g of allGroups.value) {
      for (const c of g.containers) if (c.state === 'running') n++
    }
    return n
  })

  const globalTotalCount = computed(() => {
    let n = 0
    for (const g of allGroups.value) n += g.containers.length
    return n
  })

  /** Fetches the compose-info for the current project and restores persisted stack. */
  async function fetchComposeInfo() {
    const name = projectsStore.currentProject?.name
    if (!name) {
      composeInfo.value = null
      selectedFiles.value = []
      selectedProfiles.value = []
      return
    }
    try {
      const info = await dockerApi.composeInfo(name)
      composeInfo.value = info
      const persisted = loadStack(name)
      if (persisted) {
        // Keep only files still present — a removed file would otherwise error.
        const known = new Set(info.files.map((f) => f.path))
        selectedFiles.value = persisted.files.filter((f) => known.has(f))
        // Keep only profiles still declared somewhere.
        const knownProfiles = new Set<string>()
        for (const f of info.files) for (const p of f.profiles) knownProfiles.add(p)
        selectedProfiles.value = persisted.profiles.filter((p) => knownProfiles.has(p))
      } else {
        selectedFiles.value = [...(info.default_files ?? [])]
        selectedProfiles.value = []
      }
      // If nothing selected after restore, fall back to defaults.
      if (selectedFiles.value.length === 0) {
        selectedFiles.value = [...(info.default_files ?? [])]
      }
    } catch {
      composeInfo.value = null
    }
  }

  function setSelectedFiles(files: string[]) {
    selectedFiles.value = files
    saveStack(projectName(), { files, profiles: selectedProfiles.value })
  }

  function setSelectedProfiles(profiles: string[]) {
    selectedProfiles.value = profiles
    saveStack(projectName(), { files: selectedFiles.value, profiles })
  }

  function toggleFile(file: string) {
    const set = new Set(selectedFiles.value)
    if (set.has(file)) set.delete(file); else set.add(file)
    setSelectedFiles(Array.from(set))
  }

  function toggleProfile(profile: string) {
    const set = new Set(selectedProfiles.value)
    if (set.has(profile)) set.delete(profile); else set.add(profile)
    setSelectedProfiles(Array.from(set))
  }

  async function fetchContainers() {
    loading.value = true
    try {
      containers.value = (await dockerApi.containers(projectName(), stackParams())) ?? []
    } catch (e) {
      containers.value = []
      toast.show('error', getErrorMessage(e))
    } finally {
      loading.value = false
    }
  }

  async function fetchStats() {
    try {
      stats.value = (await dockerApi.stats(projectName(), stackParams())) ?? []
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
      await dockerApi.action(projectName(), name, action, stackParams())
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
      await dockerApi.composeUp(projectName(), stackParams())
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
      await dockerApi.composeUpBuild(projectName(), stackParams())
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
      await dockerApi.composeDown(projectName(), stackParams())
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
    return dockerApi.logsUrl(projectName(), name, stackParams())
  }

  async function toggleInspect(name: string) {
    if (expandedContainer.value === name) {
      expandedContainer.value = null
      inspectData.value = null
      return
    }
    expandedContainer.value = name
    inspectLoading.value = true
    try {
      inspectData.value = await dockerApi.inspect(projectName(), name, stackParams())
    } catch (e) {
      toast.show('error', getErrorMessage(e))
      inspectData.value = null
    } finally {
      inspectLoading.value = false
    }
  }

  // --- Global scope ---

  async function fetchAllContainers() {
    allLoading.value = true
    try {
      const resp = await dockerApi.allContainers()
      allGroups.value = resp?.groups ?? []
    } catch (e) {
      allGroups.value = []
      toast.show('error', getErrorMessage(e))
    } finally {
      allLoading.value = false
    }
  }

  function startAllPolling() {
    stopAllPolling()
    fetchAllContainers()
    allInterval = setInterval(fetchAllContainers, 10000)
  }

  function stopAllPolling() {
    if (allInterval) {
      clearInterval(allInterval)
      allInterval = null
    }
  }

  async function globalAction(id: string, action: 'start' | 'stop' | 'restart' | 'kill' | 'remove') {
    allActionLoading.value = id
    try {
      await dockerApi.globalAction(id, action)
      await new Promise((r) => setTimeout(r, 1000))
      await fetchAllContainers()
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    } finally {
      allActionLoading.value = null
    }
  }

  async function stopAllRunning() {
    const running: string[] = []
    for (const g of allGroups.value) {
      for (const c of g.containers) if (c.state === 'running') running.push(c.id)
    }
    if (running.length === 0) return
    await Promise.all(running.map((id) => dockerApi.globalAction(id, 'stop').catch(() => {})))
    await fetchAllContainers()
  }

  function globalLogsUrl(id: string): string {
    return dockerApi.globalLogsUrl(id)
  }

  // Refresh compose info whenever the current project changes.
  watch(
    () => projectsStore.currentProject?.name,
    () => { fetchComposeInfo() },
    { immediate: true },
  )

  return {
    containers,
    stats,
    selectedContainer,
    loading,
    actionLoading,
    composeLoading,
    expandedContainer,
    inspectData,
    inspectLoading,
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
    toggleInspect,

    // Stage 15 — compose stack + scope
    composeInfo,
    selectedFiles,
    selectedProfiles,
    fetchComposeInfo,
    setSelectedFiles,
    setSelectedProfiles,
    toggleFile,
    toggleProfile,

    scopeTab,
    setScopeTab,

    // Global scope
    allGroups,
    allLoading,
    allActionLoading,
    globalRunningCount,
    globalTotalCount,
    fetchAllContainers,
    startAllPolling,
    stopAllPolling,
    globalAction,
    stopAllRunning,
    globalLogsUrl,
  }
})
