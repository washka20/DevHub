import { describe, it, expect } from 'vitest'
import { assignLanes, type GraphCommit } from '../git-graph'

describe('assignLanes', () => {
  it('assigns lane 0 to a linear chain of commits', () => {
    const commits: GraphCommit[] = [
      { hash: 'c3', parents: ['c2'] },
      { hash: 'c2', parents: ['c1'] },
      { hash: 'c1', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes).toHaveLength(3)
    expect(nodes[0].lane).toBe(0)
    expect(nodes[1].lane).toBe(0)
    expect(nodes[2].lane).toBe(0)
  })

  it('assigns different lanes for branching commits', () => {
    const commits: GraphCommit[] = [
      { hash: 'c4', parents: ['c3', 'c2'] },
      { hash: 'c3', parents: ['c1'] },
      { hash: 'c2', parents: ['c1'] },
      { hash: 'c1', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes[0].lane).toBe(0)
    expect(nodes[0].isMerge).toBe(true)
    expect(nodes[1].lane).toBe(0)
    expect(nodes[2].lane).toBe(1)
    expect(nodes[3].lane).toBe(0)
  })

  it('tracks edges for merge commits', () => {
    const commits: GraphCommit[] = [
      { hash: 'c3', parents: ['c2', 'c1'] },
      { hash: 'c2', parents: ['c0'] },
      { hash: 'c1', parents: ['c0'] },
      { hash: 'c0', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes[0].edges).toHaveLength(2)
    expect(nodes[0].edges[0].fromLane).toBe(0)
    expect(nodes[0].edges[0].toLane).toBe(0)
    expect(nodes[0].edges[1].fromLane).toBe(0)
    expect(nodes[0].edges[1].toLane).toBeGreaterThan(0)
  })

  it('returns maxLane reflecting total active lanes', () => {
    const commits: GraphCommit[] = [
      { hash: 'c3', parents: ['c2', 'c1'] },
      { hash: 'c2', parents: ['c0'] },
      { hash: 'c1', parents: ['c0'] },
      { hash: 'c0', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes[0].maxLane).toBeGreaterThanOrEqual(1)
  })

  it('handles 3 parallel branches converging', () => {
    const commits: GraphCommit[] = [
      { hash: 'm1', parents: ['a1', 'b1', 'c1'] },
      { hash: 'a1', parents: ['root'] },
      { hash: 'b1', parents: ['root'] },
      { hash: 'c1', parents: ['root'] },
      { hash: 'root', parents: [] },
    ]
    const nodes = assignLanes(commits)
    const lanes = new Set(nodes.slice(1, 4).map(n => n.lane))
    expect(lanes.size).toBe(3)
  })

  it('tracks activeLanes at each commit position', () => {
    const commits: GraphCommit[] = [
      { hash: 'c4', parents: ['c3', 'c2'] },
      { hash: 'c3', parents: ['c1'] },
      { hash: 'c2', parents: ['c1'] },
      { hash: 'c1', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes[1].activeLanes).toBeDefined()
    expect(nodes[1].activeLanes.length).toBeGreaterThanOrEqual(2)
  })

  it('handles root commit with no parents', () => {
    const commits: GraphCommit[] = [
      { hash: 'root', parents: [] },
    ]
    const nodes = assignLanes(commits)
    expect(nodes).toHaveLength(1)
    expect(nodes[0].lane).toBe(0)
    expect(nodes[0].edges).toHaveLength(0)
    expect(nodes[0].isMerge).toBe(false)
  })

  it('handles empty input', () => {
    const nodes = assignLanes([])
    expect(nodes).toHaveLength(0)
  })
})
