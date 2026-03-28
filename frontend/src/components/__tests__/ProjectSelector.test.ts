import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ProjectSelector from '../ProjectSelector.vue'
import { useProjectsStore } from '../../stores/projects'
import type { Project } from '../../types'

const mockProjects: Project[] = [
  { name: 'alpha', path: '/srv/alpha', is_git: true, has_makefile: true, has_docker: false },
  { name: 'beta', path: '/srv/beta', is_git: false, has_makefile: false, has_docker: true },
  { name: 'gamma', path: '/srv/gamma', is_git: true, has_makefile: false, has_docker: true },
]

function mountSelector() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const store = useProjectsStore()
  store.projects = [...mockProjects]
  store.currentProject = mockProjects[0]

  // Stub fetch globally for switchProject calls
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([]),
  }))

  const wrapper = mount(ProjectSelector, {
    global: { plugins: [pinia] },
  })

  return { wrapper, store }
}

describe('ProjectSelector', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('renders current project name', () => {
    const { wrapper } = mountSelector()

    expect(wrapper.find('.selector-name').text()).toBe('alpha')
  })

  it('shows "Select project" when no project is selected', () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const store = useProjectsStore()
    store.projects = []
    store.currentProject = null

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))

    const wrapper = mount(ProjectSelector, {
      global: { plugins: [pinia] },
    })

    expect(wrapper.find('.selector-name').text()).toBe('Select project')
  })

  it('opens dropdown on click', async () => {
    const { wrapper } = mountSelector()

    expect(wrapper.find('.selector-dropdown').exists()).toBe(false)

    await wrapper.find('.selector-btn').trigger('click')

    expect(wrapper.find('.selector-dropdown').exists()).toBe(true)
  })

  it('shows all projects in dropdown', async () => {
    const { wrapper } = mountSelector()

    await wrapper.find('.selector-btn').trigger('click')

    const items = wrapper.findAll('.dropdown-item')
    expect(items).toHaveLength(3)
    expect(items[0].find('.item-name').text()).toBe('alpha')
    expect(items[1].find('.item-name').text()).toBe('beta')
    expect(items[2].find('.item-name').text()).toBe('gamma')
  })

  it('closes dropdown on second click (toggle)', async () => {
    const { wrapper } = mountSelector()

    await wrapper.find('.selector-btn').trigger('click')
    expect(wrapper.find('.selector-dropdown').exists()).toBe(true)

    await wrapper.find('.selector-btn').trigger('click')
    expect(wrapper.find('.selector-dropdown').exists()).toBe(false)
  })

  it('marks current project as active in dropdown', async () => {
    const { wrapper } = mountSelector()

    await wrapper.find('.selector-btn').trigger('click')

    const items = wrapper.findAll('.dropdown-item')
    expect(items[0].classes()).toContain('active')
    expect(items[1].classes()).not.toContain('active')
  })

  it('displays feature badges for git, makefile, docker', () => {
    const { wrapper } = mountSelector()

    const badges = wrapper.findAll('.feature-badge')
    // alpha has is_git=true, has_makefile=true, has_docker=false => 2 badges (G, M)
    expect(badges).toHaveLength(2)
  })
})
