import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CommandButton from '../CommandButton.vue'

describe('CommandButton', () => {
  it('renders the command name', () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'build' },
    })

    expect(wrapper.find('.cmd-name').text()).toBe('make build')
  })

  it('emits execute on click', async () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make test' },
    })

    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('execute')).toHaveLength(1)
  })

  it('shows loading indicator when loading=true', () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make deploy', loading: true },
    })

    expect(wrapper.find('.cmd-loading').exists()).toBe(true)
    expect(wrapper.find('.cmd-loading').text()).toBe('...')
  })

  it('does not show loading indicator when loading=false', () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make deploy', loading: false },
    })

    expect(wrapper.find('.cmd-loading').exists()).toBe(false)
  })

  it('is disabled when disabled=true', () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make build', disabled: true },
    })

    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('is disabled when loading=true', () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make build', loading: true },
    })

    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('does not emit execute when disabled', async () => {
    const wrapper = mount(CommandButton, {
      props: { name: 'make build', disabled: true },
    })

    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('execute')).toBeUndefined()
  })
})
