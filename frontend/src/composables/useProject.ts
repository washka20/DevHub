import { computed } from 'vue'
import { useProjectsStore } from '../stores/projects'
import { useGitStore } from '../stores/git'
import { useDockerStore } from '../stores/docker'

const STORAGE_KEY = 'devhub_current_project'

export function useProject() {
  const store = useProjectsStore()
  const gitStore = useGitStore()
  const dockerStore = useDockerStore()

  const currentProject = computed(() => store.currentProject)

  const projectApiUrl = computed(() => {
    if (!store.currentProject) return '/api/projects'
    return `/api/projects/${store.currentProject.name}`
  })

  function resetStores() {
    gitStore.status = { branch: '', modified: [], staged: [], untracked: [], ahead: 0, behind: 0 }
    gitStore.log = []
    gitStore.diff = ''
    dockerStore.containers = []
  }

  async function refreshStores() {
    const project = store.currentProject
    if (!project) return

    const fetches: Promise<void>[] = []
    if (project.is_git) {
      fetches.push(gitStore.fetchStatus())
      fetches.push(gitStore.fetchGraph())
    }
    if (project.has_docker) {
      fetches.push(dockerStore.fetchContainers())
    }
    await Promise.all(fetches)
  }

  async function switchProject(name: string) {
    resetStores()
    store.setCurrentProject(name)
    await refreshStores()
  }

  async function initProject() {
    await store.fetchProjects()
    const savedName = localStorage.getItem(STORAGE_KEY)
    if (savedName && store.projects.some((p) => p.name === savedName)) {
      store.setCurrentProject(savedName)
    }
    await refreshStores()
  }

  return {
    currentProject,
    projectApiUrl,
    switchProject,
    initProject,
  }
}
