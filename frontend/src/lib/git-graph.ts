export interface GraphCommit {
  hash: string
  parents: string[]
}

export interface GraphEdge {
  parentHash: string
  fromLane: number
  toLane: number
  color: string
}

export interface GraphNode {
  hash: string
  lane: number
  color: string
  edges: GraphEdge[]
  maxLane: number
  isMerge: boolean
  activeLanes: (string | null)[]
}

const COLORS = [
  '#58a6ff', '#3fb950', '#f0883e', '#bc8cff',
  '#f85149', '#d29922', '#79c0ff', '#56d364',
]

/**
 * Find the nearest free lane to `preferred`, searching outward.
 * This keeps the graph compact by reusing nearby lanes.
 */
function findNearestFreeLane(activeLanes: (string | null)[], preferred: number): number {
  // Try preferred first
  if (preferred < activeLanes.length && activeLanes[preferred] === null) return preferred

  // Search outward from preferred
  for (let dist = 1; dist <= activeLanes.length; dist++) {
    const right = preferred + dist
    if (right < activeLanes.length && activeLanes[right] === null) return right
    const left = preferred - dist
    if (left >= 0 && activeLanes[left] === null) return left
  }

  // No free lane — append
  return activeLanes.length
}

export function assignLanes(commits: GraphCommit[]): GraphNode[] {
  const activeLanes: (string | null)[] = []
  const result: GraphNode[] = []

  for (const commit of commits) {
    let lane = -1
    const edges: GraphEdge[] = []

    // Find ALL lanes that point to this commit (multiple children may reference us)
    const matchingLanes: number[] = []
    for (let i = 0; i < activeLanes.length; i++) {
      if (activeLanes[i] === commit.hash) {
        matchingLanes.push(i)
      }
    }

    if (matchingLanes.length > 0) {
      // Take the leftmost lane as our own
      lane = matchingLanes[0]
      // Close all other lanes that pointed to us (merge convergence)
      for (let i = 1; i < matchingLanes.length; i++) {
        activeLanes[matchingLanes[i]] = null
      }
    }

    // No lane found — allocate nearest to lane 0
    if (lane === -1) {
      const free = findNearestFreeLane(activeLanes, 0)
      if (free >= activeLanes.length) activeLanes.push(null)
      lane = free
    }

    // Set first parent as continuation of our lane
    if (commit.parents.length > 0) {
      activeLanes[lane] = commit.parents[0]
    } else {
      activeLanes[lane] = null
    }

    // First parent edge (straight down on same lane)
    if (commit.parents.length > 0) {
      edges.push({
        parentHash: commit.parents[0],
        fromLane: lane,
        toLane: lane,
        color: COLORS[lane % COLORS.length],
      })
    }

    // Additional parents (merge) — find or allocate lanes near our position
    for (let p = 1; p < commit.parents.length; p++) {
      const parentHash = commit.parents[p]

      // Check if this parent already has a lane (another branch points to it)
      let parentLane = activeLanes.indexOf(parentHash)
      if (parentLane === -1) {
        // Allocate nearest free lane to our position
        const free = findNearestFreeLane(activeLanes, lane + 1)
        if (free >= activeLanes.length) activeLanes.push(null)
        parentLane = free
        activeLanes[parentLane] = parentHash
      }

      edges.push({
        parentHash,
        fromLane: lane,
        toLane: parentLane,
        color: COLORS[parentLane % COLORS.length],
      })
    }

    // Compact: remove trailing nulls to keep graph tight
    while (activeLanes.length > 0 && activeLanes[activeLanes.length - 1] === null) {
      activeLanes.pop()
    }

    result.push({
      hash: commit.hash,
      lane,
      color: COLORS[lane % COLORS.length],
      edges,
      maxLane: Math.max(activeLanes.length - 1, lane, 0),
      isMerge: commit.parents.length > 1,
      activeLanes: [...activeLanes],
    })
  }

  return result
}
