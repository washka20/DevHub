import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TerminalOutput from '../TerminalOutput.vue'

describe('TerminalOutput', () => {
  it('renders lines array', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: ['line 1', 'line 2', 'line 3'],
      },
    })

    const lineEls = wrapper.findAll('.terminal-line')
    expect(lineEls).toHaveLength(3)
    expect(lineEls[0].text()).toBe('line 1')
    expect(lineEls[1].text()).toBe('line 2')
    expect(lineEls[2].text()).toBe('line 3')
  })

  it('shows placeholder when lines are empty and not running', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: [],
        running: false,
      },
    })

    expect(wrapper.find('.terminal-placeholder').exists()).toBe(true)
    expect(wrapper.find('.terminal-placeholder').text()).toBe('Output will appear here...')
  })

  it('does not show placeholder when lines are empty but running', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: [],
        running: true,
      },
    })

    expect(wrapper.find('.terminal-placeholder').exists()).toBe(false)
  })

  it('shows running indicator with running class', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: ['building...'],
        running: true,
      },
    })

    const runningEl = wrapper.find('.running')
    expect(runningEl.exists()).toBe(true)
    expect(runningEl.text()).toContain('Running...')
  })

  it('does not show running indicator when running=false', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: ['done'],
        running: false,
      },
    })

    // There should be no element with .running class that contains "Running..."
    const runningEls = wrapper.findAll('.running')
    const hasRunningIndicator = runningEls.some((el) => el.text().includes('Running...'))
    expect(hasRunningIndicator).toBe(false)
  })

  it('strips ANSI escape codes from lines', () => {
    const wrapper = mount(TerminalOutput, {
      props: {
        lines: ['\x1b[32mGreen text\x1b[0m', '\x1b[1;31mBold red\x1b[0m'],
      },
    })

    const lineEls = wrapper.findAll('.terminal-line')
    expect(lineEls[0].text()).toBe('Green text')
    expect(lineEls[1].text()).toBe('Bold red')
  })
})
