# Topology-Based Git Graph Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ASCII-based git graph with topology-based lane assignment and Bezier curve rendering, like PhpStorm/IntelliJ.

**Architecture:** Backend sends commit hashes + parent hashes. Frontend builds a DAG, assigns lanes via greedy column allocation, renders SVG with cubic Bezier curves for cross-lane edges. Graph-only rows eliminated — every row is a commit.

**Tech Stack:** Go backend, Vue 3 + TypeScript frontend, Vitest for frontend tests, Go testing for backend.

---

## File Structure

| File | Purpose |
|------|---------|
| `internal/git/git.go` | Modify `Log()` to include parent hashes, remove `GraphOnly` |
| `internal/git/git_test.go` | Update test for new `Log()` format |
| `internal/api/handlers.go` | No changes needed (passes through) |
| `frontend/src/types/index.ts` | Add `parents` field, remove `graph`/`graph_only` |
| `frontend/src/lib/git-graph.ts` | **NEW** — lane assignment algorithm (pure logic, no Vue) |
| `frontend/src/lib/__tests__/git-graph.test.ts` | **NEW** — TDD tests for lane assignment |
| `frontend/src/stores/git.ts` | Update `parseCommits` for new fields |
| `frontend/src/views/GitView.vue` | Replace ASCII SVG with topology-based rendering |

---

### Task 1: Backend — Add Parent Hashes to Commit

**Files:**
- Modify: `internal/git/git.go`
- Test: `internal/git/git_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/git/git_test.go`, add a test for parent hashes:

```go
func TestLog_WithParents(t *testing.T) {
	logOutput := `abc1234567890abc1234567890abc1234567890ab|abc1234|initial commit|John|2 hours ago||
def5678901234def5678901234def5678901234de|def5678|add feature|Jane|1 hour ago|HEAD -> main|abc1234567890abc1234567890abc1234567890ab
ghi9012345678ghi9012345678ghi9012345678gh|ghi9012|merge branch|Bob|30 min ago||def5678901234def5678901234def5678901234de abc1234567890abc1234567890abc1234567890ab`

	mock := &MockRunner{calls: []MockCall{
		{Output: logOutput},
	}}

	svc := NewGitService(mock)
	commits, err := svc.Log("/test", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}

	// First commit has no parents (root)
	if len(commits[0].Parents) != 0 {
		t.Errorf("root commit should have 0 parents, got %d", len(commits[0].Parents))
	}

	// Second commit has 1 parent
	if len(commits[1].Parents) != 1 || commits[1].Parents[0] != "abc1234567890abc1234567890abc1234567890ab" {
		t.Errorf("expected 1 parent abc..., got %v", commits[1].Parents)
	}

	// Merge commit has 2 parents
	if len(commits[2].Parents) != 2 {
		t.Errorf("merge commit should have 2 parents, got %d: %v", len(commits[2].Parents), commits[2].Parents)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/washka/project/devhub && go test ./internal/git/ -run TestLog_WithParents -v`
Expected: FAIL — `commits[0].Parents` field doesn't exist

- [ ] **Step 3: Update Commit struct and Log() function**

In `internal/git/git.go`, update the Commit struct:

```go
type Commit struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short_hash"`
	Message   string   `json:"message"`
	Author    string   `json:"author"`
	Date      string   `json:"date"`
	Refs      []string `json:"refs"`
	Parents   []string `json:"parents"`
}
```

Replace the `Log()` function entirely:

```go
func (g *GitService) Log(dir string, limit int, offset int) ([]Commit, error) {
	args := []string{"log", "--all",
		"--format=%H|%h|%s|%an|%ar|%D|%P", "-n", strconv.Itoa(limit)}
	if offset > 0 {
		args = append(args, "--skip", strconv.Itoa(offset))
	}
	out, err := g.runner.Run(dir, "git", args...)
	if err != nil {
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 7 {
			continue
		}

		var refs []string
		if strings.TrimSpace(parts[5]) != "" {
			for _, ref := range strings.Split(parts[5], ", ") {
				ref = strings.TrimSpace(ref)
				if ref != "" {
					refs = append(refs, ref)
				}
			}
		}

		var parents []string
		if strings.TrimSpace(parts[6]) != "" {
			for _, p := range strings.Fields(parts[6]) {
				parents = append(parents, p)
			}
		}

		commits = append(commits, Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      parts[4],
			Refs:      refs,
			Parents:   parents,
		})
	}
	return commits, nil
}
```

Key changes:
- Removed `--graph` and `--oneline` flags (no more ASCII graph)
- Added `%P` to format (parent hashes, space-separated)
- Format now has 7 fields: `%H|%h|%s|%an|%ar|%D|%P`
- Removed `GraphOnly` field entirely
- No regex needed — each line is a commit

- [ ] **Step 4: Update existing TestLog to match new format**

Update the existing `TestLog` test in `git_test.go`:

```go
func TestLog(t *testing.T) {
	logOutput := `abc1234567890abc1234567890abc1234567890ab|abc1234|initial commit|John|2 hours ago|HEAD -> main|
def5678901234def5678901234def5678901234de|def5678|add feature|Jane|1 hour ago||abc1234567890abc1234567890abc1234567890ab`

	mock := &MockRunner{calls: []MockCall{
		{Output: logOutput},
	}}

	svc := NewGitService(mock)
	commits, err := svc.Log("/test", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}

	if commits[0].ShortHash != "abc1234" {
		t.Errorf("expected short hash abc1234, got %s", commits[0].ShortHash)
	}
	if commits[0].Message != "initial commit" {
		t.Errorf("expected message 'initial commit', got %s", commits[0].Message)
	}
	if len(commits[0].Refs) != 1 {
		t.Errorf("expected 1 ref, got %d: %v", len(commits[0].Refs), commits[0].Refs)
	}
}
```

- [ ] **Step 5: Run all tests**

Run: `cd /home/washka/project/devhub && go test ./internal/git/ -v`
Expected: ALL PASS

- [ ] **Step 6: Build and verify**

Run: `cd /home/washka/project/devhub && go build -o devhub ./cmd/`
Expected: BUILD OK

- [ ] **Step 7: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "refactor: заменить ASCII graph на parent hashes в git log API"
```

---

### Task 2: Frontend — Lane Assignment Algorithm (Pure Logic, TDD)

**Files:**
- Create: `frontend/src/lib/git-graph.ts`
- Create: `frontend/src/lib/__tests__/git-graph.test.ts`

This is the core algorithm. It takes commits with parent hashes and outputs lane (column) assignments and edge data for SVG rendering.

- [ ] **Step 1: Write types and first failing test — linear history**

Create `frontend/src/lib/__tests__/git-graph.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { assignLanes, type GraphCommit, type GraphNode } from '../git-graph'

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
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/washka/project/devhub/frontend && npx vitest run src/lib/__tests__/git-graph.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Write minimal implementation for linear history**

Create `frontend/src/lib/git-graph.ts`:

```typescript
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
}

const COLORS = [
  '#58a6ff', '#3fb950', '#f0883e', '#bc8cff',
  '#f85149', '#d29922', '#79c0ff', '#56d364',
]

export function assignLanes(commits: GraphCommit[]): GraphNode[] {
  // activeLanes[lane] = hash of commit that "owns" this lane going downward
  const activeLanes: (string | null)[] = []
  const hashToNode = new Map<string, GraphNode>()
  const result: GraphNode[] = []

  for (const commit of commits) {
    let lane = -1
    const edges: GraphEdge[] = []

    // Find if any active lane points to this commit (a child listed us as parent)
    for (let i = 0; i < activeLanes.length; i++) {
      if (activeLanes[i] === commit.hash) {
        if (lane === -1) {
          lane = i // First match — this is our lane
        } else {
          // Additional lane merging into us — close it
          activeLanes[i] = null
        }
      }
    }

    // No lane found — allocate new one
    if (lane === -1) {
      lane = activeLanes.indexOf(null)
      if (lane === -1) {
        lane = activeLanes.length
        activeLanes.push(null)
      }
    }

    // Set first parent as continuation of our lane
    if (commit.parents.length > 0) {
      activeLanes[lane] = commit.parents[0]
    } else {
      activeLanes[lane] = null // Root commit — lane ends
    }

    // Additional parents (merge) — allocate or reuse lanes
    for (let p = 1; p < commit.parents.length; p++) {
      const parentHash = commit.parents[p]
      let parentLane = activeLanes.indexOf(parentHash)
      if (parentLane === -1) {
        // Allocate a new lane for this parent
        parentLane = activeLanes.indexOf(null)
        if (parentLane === -1) {
          parentLane = activeLanes.length
          activeLanes.push(null)
        }
        activeLanes[parentLane] = parentHash
      }
      edges.push({
        parentHash,
        fromLane: lane,
        toLane: parentLane,
        color: COLORS[parentLane % COLORS.length],
      })
    }

    // First parent edge (straight down or cross-lane)
    if (commit.parents.length > 0) {
      const firstParentLane = lane // first parent continues our lane
      edges.unshift({
        parentHash: commit.parents[0],
        fromLane: lane,
        toLane: firstParentLane,
        color: COLORS[lane % COLORS.length],
      })
    }

    // Compact: remove trailing nulls
    while (activeLanes.length > 0 && activeLanes[activeLanes.length - 1] === null) {
      activeLanes.pop()
    }

    const node: GraphNode = {
      hash: commit.hash,
      lane,
      color: COLORS[lane % COLORS.length],
      edges,
      maxLane: Math.max(activeLanes.length - 1, lane),
      isMerge: commit.parents.length > 1,
    }

    hashToNode.set(commit.hash, node)
    result.push(node)
  }

  return result
}
```

- [ ] **Step 4: Run test**

Run: `cd /home/washka/project/devhub/frontend && npx vitest run src/lib/__tests__/git-graph.test.ts`
Expected: PASS

- [ ] **Step 5: Add test — simple branch and merge**

Add to the test file:

```typescript
  it('assigns different lanes for branching commits', () => {
    // c4 merged c3 and c2. c3 and c2 both have parent c1.
    const commits: GraphCommit[] = [
      { hash: 'c4', parents: ['c3', 'c2'] },  // merge commit
      { hash: 'c3', parents: ['c1'] },
      { hash: 'c2', parents: ['c1'] },
      { hash: 'c1', parents: [] },
    ]

    const nodes = assignLanes(commits)

    // c4 is a merge — should be on lane 0
    expect(nodes[0].lane).toBe(0)
    expect(nodes[0].isMerge).toBe(true)

    // c3 continues lane 0 (first parent of c4)
    expect(nodes[1].lane).toBe(0)

    // c2 is on a different lane (second parent of c4)
    expect(nodes[2].lane).toBe(1)

    // c1 should collapse back to lane 0
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

    // Merge commit should have 2 edges
    expect(nodes[0].edges).toHaveLength(2)
    // First edge: straight down to c2
    expect(nodes[0].edges[0].fromLane).toBe(0)
    expect(nodes[0].edges[0].toLane).toBe(0)
    // Second edge: curve to c1's lane
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

    // At the merge commit, 2 lanes are active
    expect(nodes[0].maxLane).toBeGreaterThanOrEqual(1)
  })
```

- [ ] **Step 6: Run tests**

Run: `cd /home/washka/project/devhub/frontend && npx vitest run src/lib/__tests__/git-graph.test.ts`
Expected: ALL PASS

- [ ] **Step 7: Add test — multiple parallel branches**

```typescript
  it('handles 3 parallel branches converging', () => {
    const commits: GraphCommit[] = [
      { hash: 'm1', parents: ['a1', 'b1', 'c1'] }, // 3-way merge
      { hash: 'a1', parents: ['root'] },
      { hash: 'b1', parents: ['root'] },
      { hash: 'c1', parents: ['root'] },
      { hash: 'root', parents: [] },
    ]

    const nodes = assignLanes(commits)

    // All branches should be on different lanes
    const lanes = new Set(nodes.slice(1, 4).map(n => n.lane))
    expect(lanes.size).toBe(3)
  })
```

- [ ] **Step 8: Run tests**

Run: `cd /home/washka/project/devhub/frontend && npx vitest run src/lib/__tests__/git-graph.test.ts`
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git add frontend/src/lib/git-graph.ts frontend/src/lib/__tests__/git-graph.test.ts
git commit -m "feat: добавить алгоритм lane assignment для topology-based графа (TDD)"
```

---

### Task 3: Frontend — Update Types and Store

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/git.ts`

- [ ] **Step 1: Update Commit interface**

In `frontend/src/types/index.ts`, replace the Commit interface:

```typescript
export interface Commit {
  hash: string
  short_hash: string
  message: string
  author: string
  date: string
  refs: string[]
  parents: string[]
}
```

Removed: `graph`, `graph_only`. Added: `parents`.

- [ ] **Step 2: Update parseCommits in store**

In `frontend/src/stores/git.ts`, update `parseCommits`:

```typescript
  function parseCommits(data: unknown[]): Commit[] {
    return (data ?? []).map((c: Record<string, unknown>) => ({
      hash: (c.hash ?? '') as string,
      short_hash: (c.short_hash ?? (typeof c.hash === 'string' ? (c.hash as string).slice(0, 7) : '')) as string,
      message: (c.message ?? '') as string,
      author: (c.author ?? '') as string,
      date: (c.date ?? '') as string,
      refs: Array.isArray(c.refs) ? c.refs as string[] : [],
      parents: Array.isArray(c.parents) ? c.parents as string[] : [],
    }))
  }
```

- [ ] **Step 3: Run frontend type check**

Run: `cd /home/washka/project/devhub/frontend && npx vue-tsc --noEmit 2>&1 | head -30`
Expected: Type errors in GitView.vue (expected — we'll fix in Task 4)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/stores/git.ts
git commit -m "refactor: обновить типы Commit — parents вместо graph"
```

---

### Task 4: Frontend — Replace Graph Rendering in GitView

**Files:**
- Modify: `frontend/src/views/GitView.vue`

This is the biggest change — replace all ASCII graph rendering with topology-based SVG.

- [ ] **Step 1: Replace graph imports and computed**

In the `<script setup>` section of GitView.vue:

Remove: `GRAPH_COLORS`, `GRAPH_COL_WIDTH`, `GRAPH_ROW_HEIGHT`, `GraphCell` interface, `parseGraphLine()`, `graphSvgWidth()`.

Add imports and computed:

```typescript
import { assignLanes, type GraphNode } from '../lib/git-graph'

const LANE_WIDTH = 16
const ROW_HEIGHT = 28
const NODE_RADIUS = 4
const MERGE_RADIUS = 5
const GRAPH_COLORS = [
  '#58a6ff', '#3fb950', '#f0883e', '#bc8cff',
  '#f85149', '#d29922', '#79c0ff', '#56d364',
]

const graphNodes = computed<GraphNode[]>(() => {
  const commits = gitStore.commits.map(c => ({
    hash: c.hash,
    parents: c.parents ?? [],
  }))
  return assignLanes(commits)
})

function graphWidth(node: GraphNode): number {
  return (node.maxLane + 1) * LANE_WIDTH + 16
}

function nodeX(lane: number): number {
  return lane * LANE_WIDTH + 8
}

function edgePath(fromLane: number, toLane: number): string {
  const x1 = nodeX(fromLane)
  const x2 = nodeX(toLane)
  const h = ROW_HEIGHT
  if (x1 === x2) return '' // straight lines handled separately
  // Cubic Bezier: start at bottom of current row, end at top of next row
  return `M ${x1},${h / 2 + NODE_RADIUS} C ${x1},${h} ${x2},${h} ${x2},${h + h / 2 - NODE_RADIUS}`
}
```

- [ ] **Step 2: Replace the log template**

Replace the entire `<!-- ==================== TAB: LOG ==================== -->` section with:

```vue
      <!-- ==================== TAB: LOG ==================== -->
      <div v-if="gitStore.activeTab === 'log'" class="log-layout">
        <div class="log-main" @scroll="onLogScroll">
          <div v-if="gitStore.commits.length === 0" class="empty-state">
            <span class="empty-text">No commits</span>
          </div>
          <div v-else class="log-list">
            <div
              v-for="(c, idx) in gitStore.commits"
              :key="c.hash"
              class="log-row"
              :class="{ 'log-row-selected': gitStore.selectedCommit?.hash === c.hash }"
              @click="selectCommit(c.hash)"
            >
              <!-- Graph column -->
              <div
                class="log-graph-col"
                :style="{ width: (graphNodes[idx] ? graphWidth(graphNodes[idx]) : 32) + 'px' }"
              >
                <svg
                  v-if="graphNodes[idx]"
                  :width="graphWidth(graphNodes[idx])"
                  :height="ROW_HEIGHT"
                  class="graph-svg"
                >
                  <!-- Active lane lines (vertical pipes for all active lanes) -->
                  <template v-for="edge in graphNodes[idx].edges" :key="edge.parentHash">
                    <!-- Straight down (same lane) -->
                    <line
                      v-if="edge.fromLane === edge.toLane"
                      :x1="nodeX(edge.fromLane)" y1="0"
                      :x2="nodeX(edge.toLane)" :y2="ROW_HEIGHT"
                      :stroke="edge.color" stroke-width="2" stroke-linecap="round"
                    />
                    <!-- Cross-lane curve -->
                    <path
                      v-else
                      :d="`M ${nodeX(edge.fromLane)},${ROW_HEIGHT / 2} C ${nodeX(edge.fromLane)},${ROW_HEIGHT * 0.85} ${nodeX(edge.toLane)},${ROW_HEIGHT * 0.85} ${nodeX(edge.toLane)},${ROW_HEIGHT}`"
                      :stroke="edge.color" stroke-width="2" fill="none" stroke-linecap="round"
                    />
                  </template>
                  <!-- Pass-through lanes (other active lanes not in edges) -->
                  <!-- We render vertical lines for lanes that are active but not part of this commit's edges -->

                  <!-- Commit node -->
                  <circle
                    :cx="nodeX(graphNodes[idx].lane)"
                    :cy="ROW_HEIGHT / 2"
                    :r="graphNodes[idx].isMerge ? MERGE_RADIUS : NODE_RADIUS"
                    :fill="graphNodes[idx].isMerge ? '#0d1117' : graphNodes[idx].color"
                    :stroke="graphNodes[idx].color"
                    :stroke-width="graphNodes[idx].isMerge ? 2 : 0"
                  />
                </svg>
              </div>
              <!-- Commit info -->
              <div class="log-commit-col">
                <span class="log-hash">{{ c.short_hash }}</span>
                <span v-for="r in c.refs" :key="r" class="ref-badge" :class="getRefClass(r)">
                  {{ getRefLabel(r) }}
                </span>
                <span class="log-msg">{{ c.message }}</span>
              </div>
              <div class="log-meta-col">
                <span class="log-author">{{ c.author }}</span>
                <span class="log-time">{{ c.date }}</span>
              </div>
            </div>
          </div>
          <div v-if="gitStore.logLoadingMore" class="log-loading-more">
            Loading more commits...
          </div>
          <div v-else-if="!gitStore.logHasMore && gitStore.commits.length > 0" class="log-end">
            End of history
          </div>
        </div>

        <!-- Commit detail panel (keep existing) -->
```

- [ ] **Step 3: Remove old graph CSS**

Remove `.log-row-graph-only` CSS class. Remove the old graph-only template code. Keep all other log CSS.

- [ ] **Step 4: Run in browser and verify**

Run: open `http://localhost:5173/` → Git → Log tab
Expected: Commits with colored dots on lanes, cross-lane curves for merges

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/GitView.vue
git commit -m "feat: topology-based git graph rendering с Bezier кривыми"
```

---

### Task 5: Refinement — Pass-Through Lane Lines

The basic rendering from Task 4 shows commit nodes and merge curves, but doesn't draw vertical "pass-through" lines for lanes that are active but don't belong to the current commit. This is what makes the graph look connected.

**Files:**
- Modify: `frontend/src/lib/git-graph.ts`
- Modify: `frontend/src/lib/__tests__/git-graph.test.ts`

- [ ] **Step 1: Write failing test for activeLanes tracking**

Add to `git-graph.test.ts`:

```typescript
  it('tracks activeLanes at each commit position', () => {
    const commits: GraphCommit[] = [
      { hash: 'c4', parents: ['c3', 'c2'] },
      { hash: 'c3', parents: ['c1'] },
      { hash: 'c2', parents: ['c1'] },
      { hash: 'c1', parents: [] },
    ]

    const nodes = assignLanes(commits)

    // At c3 (index 1), lane 1 should still be active (c2 hasn't been processed yet)
    expect(nodes[1].activeLanes).toBeDefined()
    expect(nodes[1].activeLanes.length).toBeGreaterThanOrEqual(2)
  })
```

- [ ] **Step 2: Add `activeLanes` to GraphNode**

In `frontend/src/lib/git-graph.ts`, add to `GraphNode`:

```typescript
export interface GraphNode {
  hash: string
  lane: number
  color: string
  edges: GraphEdge[]
  maxLane: number
  isMerge: boolean
  activeLanes: (string | null)[]  // snapshot of active lanes at this row
}
```

And in `assignLanes()`, before pushing to result, add:

```typescript
    const node: GraphNode = {
      hash: commit.hash,
      lane,
      color: COLORS[lane % COLORS.length],
      edges,
      maxLane: Math.max(activeLanes.length - 1, lane),
      isMerge: commit.parents.length > 1,
      activeLanes: [...activeLanes],  // snapshot
    }
```

- [ ] **Step 3: Run tests**

Run: `cd /home/washka/project/devhub/frontend && npx vitest run src/lib/__tests__/git-graph.test.ts`
Expected: ALL PASS

- [ ] **Step 4: Add pass-through lines to GitView SVG**

In the log row SVG template, before the commit circle, add:

```vue
                  <!-- Pass-through lanes -->
                  <line
                    v-for="(laneHash, laneIdx) in graphNodes[idx].activeLanes"
                    :key="'pass-' + laneIdx"
                    v-show="laneHash && laneIdx !== graphNodes[idx].lane"
                    :x1="nodeX(laneIdx)" y1="0"
                    :x2="nodeX(laneIdx)" :y2="ROW_HEIGHT"
                    :stroke="GRAPH_COLORS[laneIdx % GRAPH_COLORS.length]"
                    stroke-width="2" stroke-linecap="round"
                    opacity="0.6"
                  />
```

- [ ] **Step 5: Verify in browser**

Expected: Vertical colored lines connecting commits across rows, creating a "railroad track" effect.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/lib/git-graph.ts frontend/src/lib/__tests__/git-graph.test.ts frontend/src/views/GitView.vue
git commit -m "feat: добавить pass-through lane lines для связности графа"
```

---

### Task 6: Polish — Graph Width Consistency and Cleanup

**Files:**
- Modify: `frontend/src/views/GitView.vue`

- [ ] **Step 1: Compute consistent graph width across all visible commits**

In GitView script, add:

```typescript
const maxGraphWidth = computed(() => {
  if (graphNodes.value.length === 0) return 32
  const maxLane = Math.max(...graphNodes.value.map(n => n.maxLane))
  return (maxLane + 1) * LANE_WIDTH + 16
})
```

- [ ] **Step 2: Use consistent width in template**

Replace dynamic width per-row with consistent width:

```vue
              <div class="log-graph-col" :style="{ width: maxGraphWidth + 'px' }">
                <svg v-if="graphNodes[idx]" :width="maxGraphWidth" :height="ROW_HEIGHT" class="graph-svg">
```

- [ ] **Step 3: Remove all old ASCII graph code**

Remove from GitView.vue any remaining references to: `parseGraphLine`, `graphSvgWidth`, `GraphCell` interface, graph-only template blocks, `.log-row-graph-only` CSS.

- [ ] **Step 4: Remove `graph` and `graph_only` from DashboardView and AppSidebar if referenced**

Check and remove any references to `c.graph` or `c.graph_only` in other views.

- [ ] **Step 5: Full browser verification**

Open all 4 tabs (Changes, Log, Branches, Dashboard). Ensure:
- Log shows connected graph with curves
- Infinite scroll loads more commits with graph continuing
- No console errors
- Colors are consistent per lane

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: cleanup — убрать ASCII graph, единая ширина графа"
```

---

## Verification Checklist

1. `cd /home/washka/project/devhub && go test ./internal/git/ -v` — all Go tests pass
2. `cd /home/washka/project/devhub/frontend && npx vitest run` — all frontend tests pass
3. Build: `cd /home/washka/project/devhub && go build -o devhub ./cmd/`
4. Restart backend, open `http://localhost:5173/` → Git → Log
5. Verify: colored lanes, Bezier curves on merges, pass-through vertical lines
6. Scroll down — more commits load, graph continues seamlessly
7. Click a commit — detail panel opens
