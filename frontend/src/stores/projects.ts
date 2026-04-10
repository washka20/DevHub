import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useToast } from '../composables/useToast'
import { getErrorMessage } from '../utils/error'
import type { Project } from '../types'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref<Project[]>([])
  const currentProject = ref<Project | null>(null)

  const toast = useToast()

  async function fetchProjects() {
    try {
      const res = await fetch('/api/projects')
      if (!res.ok) throw new Error(`Failed to fetch projects: ${res.statusText}`)
      projects.value = await res.json()
      if (!currentProject.value && projects.value.length > 0) {
        currentProject.value = projects.value[0]
      }
    } catch (e) {
      toast.show('error', getErrorMessage(e))
    }
  }

  function setCurrentProject(name: string) {
    const found = projects.value.find((p) => p.name === name)
    if (found) {
      currentProject.value = found
      localStorage.setItem('devhub_current_project', name)
    }
  }

  return { projects, currentProject, fetchProjects, setCurrentProject }
})
