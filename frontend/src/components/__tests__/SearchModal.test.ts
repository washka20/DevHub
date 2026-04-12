import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import SearchModal from '../SearchModal.vue'
import { useProjectsStore } from '../../stores/projects'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

let wrapper: VueWrapper

function mountModal(visible = true) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const store = useProjectsStore()
  store.projects = [
    { name: 'test-project', path: '/srv/test-project', is_git: true, has_makefile: false, has_docker: false },
  ]
  store.currentProject = store.projects[0]

  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([]),
  }))

  wrapper = mount(SearchModal, {
    props: { visible },
    global: { plugins: [pinia] },
    attachTo: document.body,
  })

  return wrapper
}

describe('SearchModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    wrapper?.unmount()
    vi.useRealTimers()
  })

  it('renders when visible is true', () => {
    mountModal(true)
    expect(document.querySelector('.modal-overlay')).not.toBeNull()
    expect(document.querySelector('.search-input')).not.toBeNull()
  })

  it('does not render when visible is false', () => {
    mountModal(false)
    expect(document.querySelector('.modal-overlay')).toBeNull()
  })

  it('emits close on ESC key', async () => {
    const w = mountModal(true)
    const content = document.querySelector('.modal-content') as HTMLElement
    content.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await w.vm.$nextTick()
    expect(w.emitted('close')).toBeTruthy()
  })

  it('calls search API after debounce when query >= 2 chars', async () => {
    vi.useRealTimers()
    const searchResults = [{ file: 'a.go', line: 1, column: 0, content: 'test match' }]
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(searchResults),
    })
    vi.stubGlobal('fetch', mockFetch)

    // Mount as not visible first, then switch to visible
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useProjectsStore()
    store.projects = [
      { name: 'test-project', path: '/srv/test-project', is_git: true, has_makefile: false, has_docker: false },
    ]
    store.currentProject = store.projects[0]

    wrapper = mount(SearchModal, {
      props: { visible: false },
      global: { plugins: [pinia] },
      attachTo: document.body,
    })

    // Now make visible — this triggers the watch which resets query
    await wrapper.setProps({ visible: true })
    await nextTick()

    // Now set query and call onInput
    wrapper.vm.query = 'test'
    await nextTick()
    wrapper.vm.onInput()

    // Wait for debounce
    await new Promise(r => setTimeout(r, 400))
    await flushPromises()

    expect(mockFetch).toHaveBeenCalled()
    const callUrl = mockFetch.mock.calls[0][0] as string
    expect(callUrl).toContain('/files/search?q=test')
  })

  it('does not search for queries shorter than 2 chars', async () => {
    vi.useRealTimers()
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    })
    vi.stubGlobal('fetch', mockFetch)

    const w = mountModal(true)
    await nextTick()

    w.vm.query = 't'
    w.vm.onInput()

    await new Promise(r => setTimeout(r, 400))
    await flushPromises()

    expect(mockFetch).not.toHaveBeenCalled()
  })
})
