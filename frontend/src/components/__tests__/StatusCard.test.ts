import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusCard from '../StatusCard.vue'

describe('StatusCard', () => {
  it('renders label and value props', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Branch',
        value: 'main',
      },
    })

    expect(wrapper.find('.status-label').text()).toBe('Branch')
    expect(wrapper.find('.status-value').text()).toBe('main')
  })

  it('renders numeric value', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Commits',
        value: 42,
      },
    })

    expect(wrapper.find('.status-value').text()).toBe('42')
  })

  it('applies color style to value', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Status',
        value: 'OK',
        color: '#3fb950',
      },
    })

    const card = wrapper.find('.status-card')
    expect(card.attributes('style')).toContain('#3fb950')
  })

  it('uses default color when color prop is not provided', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Test',
        value: 'val',
      },
    })

    const card = wrapper.find('.status-card')
    expect(card.attributes('style')).toContain('--text-primary')
  })

  it('renders subtext when provided', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Branch',
        value: 'main',
        subtext: 'up to date',
      },
    })

    expect(wrapper.find('.status-subtext').exists()).toBe(true)
    expect(wrapper.find('.status-subtext').text()).toBe('up to date')
  })

  it('does not render subtext when not provided', () => {
    const wrapper = mount(StatusCard, {
      props: {
        label: 'Branch',
        value: 'main',
      },
    })

    expect(wrapper.find('.status-subtext').exists()).toBe(false)
  })
})
