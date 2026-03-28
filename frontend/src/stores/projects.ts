import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Project } from '../types'

export const useProjectsStore = defineStore('projects', () => {
  const projects = ref<Project[]>([])
  const currentProject = ref<Project | null>(null)

  async function fetchProjects() {
    const res = await fetch('/api/projects')
    projects.value = await res.json()
    if (!currentProject.value && projects.value.length > 0) {
      currentProject.value = projects.value[0]
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
