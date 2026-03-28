/**
 * Git Graph Layout Engine — compact lane assignment алгоритм.
 *
 * Вход: коммиты в topo-order [{id, parents}, ...]
 * Выход: [{column, color, lines}, ...] — данные для SVG рендеринга.
 *
 * Алгоритм основан на подходе JetBrains (intellij-community):
 * - Жадное назначение lane'ов с агрессивным переиспользованием
 * - Первый parent продолжает lane (прямая линия)
 * - Освобождение lane'а как только коммит не нужен как parent
 * - Компактный поиск свободного lane'а от preferred позиции
 */

export interface TopoNode {
  id: string
  parents: string[]
}

export interface GraphLayoutNode {
  column: number
  color: string
  lines: LayoutLine[]
}

export interface LayoutLine {
  x1: number
  x2: number
  type: number // 0=BottomHalf, 1=TopHalf, 2=Full, 3=Fork, 4=MergeBack
  color: string
}

// Типы линий (совместимы с существующим SVG рендерингом)
const LINE_BOTTOM_HALF = 0
const LINE_TOP_HALF = 1
const LINE_FULL = 2
const LINE_FORK = 3
const LINE_MERGE_BACK = 4

const COLORS = [
  '#58a6ff', '#3fb950', '#f0883e', '#bc8cff',
  '#f85149', '#d29922', '#79c0ff', '#56d364',
  '#ff7b72', '#a5d6ff', '#7ee787', '#ffa657',
]

/**
 * Вычисляет layout для графа коммитов.
 * Возвращает массив GraphLayoutNode — по одному для каждого коммита (в том же порядке).
 */
export function computeGraphLayout(nodes: TopoNode[]): { layout: GraphLayoutNode[], maxWidth: number } {
  if (nodes.length === 0) {
    return { layout: [], maxWidth: 0 }
  }

  // === Phase 1: Назначение lanes ===
  const laneAssignment = assignLanes(nodes)

  // === Phase 2: Построение линий для каждой строки ===
  const layout = buildRowLines(nodes, laneAssignment)

  // === Phase 3: Считаем maxWidth ===
  let maxWidth = 0
  for (const node of layout) {
    if (node.column > maxWidth) maxWidth = node.column
    for (const line of node.lines) {
      if (line.x1 > maxWidth) maxWidth = line.x1
      if (line.x2 > maxWidth) maxWidth = line.x2
    }
  }

  return { layout, maxWidth }
}

interface LaneState {
  // Какой коммит "живёт" на каждом lane (null = свободен)
  lanes: (string | null)[]
  // Какой цвет назначен каждому lane
  colors: number[]
}

/**
 * Phase 1: Назначение lane (колонки) каждому коммиту.
 *
 * Ключевые принципы (как JetBrains):
 * - Первый parent продолжает lane (прямая линия)
 * - Lane освобождается МГНОВЕННО когда коммит обработан (его строка пройдена)
 * - Свободные lane'ы переиспользуются агрессивно (ближайший к preferred)
 * - Merge convergence: все lane'ы указывающие на один коммит схлопываются
 */
function assignLanes(nodes: TopoNode[]): Map<string, { column: number, colorIdx: number }> {
  const result = new Map<string, { column: number, colorIdx: number }>()

  // activeLanes[i] = commitId который ОЖИДАЕТСЯ на этом lane (т.е. ещё не обработан)
  // Когда мы обрабатываем коммит, lane'ы указывающие на него "подхватываются"
  const activeLanes: (string | null)[] = []
  const laneColors: number[] = []
  let nextColorIdx = 0

  for (let row = 0; row < nodes.length; row++) {
    const node = nodes[row]
    const parents = node.parents

    // --- Step 1: Найти lane'ы которые ожидают этот коммит ---
    const matchingLanes: number[] = []
    for (let i = 0; i < activeLanes.length; i++) {
      if (activeLanes[i] === node.id) {
        matchingLanes.push(i)
      }
    }

    let column: number
    let colorIdx: number

    if (matchingLanes.length > 0) {
      // Берём самый левый matching lane
      column = matchingLanes[0]
      colorIdx = laneColors[column]

      // АГРЕССИВНО закрываем ВСЕ остальные matching lanes
      for (let i = 1; i < matchingLanes.length; i++) {
        activeLanes[matchingLanes[i]] = null
      }
    } else {
      // Новый коммит (head ветки) — ближайший свободный lane к 0
      column = findFreeLane(activeLanes, 0)
      colorIdx = nextColorIdx++
      while (activeLanes.length <= column) {
        activeLanes.push(null)
        laneColors.push(0)
      }
      laneColors[column] = colorIdx
    }

    // Убеждаемся что lane существует
    while (activeLanes.length <= column) {
      activeLanes.push(null)
      laneColors.push(0)
    }

    result.set(node.id, { column, colorIdx })

    // --- Step 2: Первый parent продолжает ЭТОТ lane ---
    if (parents.length > 0) {
      activeLanes[column] = parents[0]
    } else {
      activeLanes[column] = null // root — освобождаем
    }

    // --- Step 3: Дополнительные parent'ы (merge) ---
    for (let p = 1; p < parents.length; p++) {
      const parentId = parents[p]

      // Может parent уже ожидается на каком-то lane (другая ветка указывает на него)
      const existingLane = activeLanes.indexOf(parentId)
      if (existingLane !== -1) {
        // Уже есть — ничего выделять не нужно, edge просто будет к этому lane
        continue
      }

      // Выделяем новый lane рядом с текущим
      const newLane = findFreeLane(activeLanes, column + 1)
      while (activeLanes.length <= newLane) {
        activeLanes.push(null)
        laneColors.push(0)
      }
      activeLanes[newLane] = parentId
      laneColors[newLane] = nextColorIdx++
    }

    // --- Step 4: Compact trailing nulls ---
    while (activeLanes.length > 0 && activeLanes[activeLanes.length - 1] === null) {
      activeLanes.pop()
      laneColors.pop()
    }
  }

  return result
}

/**
 * Phase 2: Для каждой строки строим линии (какие вертикальные/диагональные линии рисовать).
 */
function buildRowLines(
  nodes: TopoNode[],
  laneAssignment: Map<string, { column: number, colorIdx: number }>,
): GraphLayoutNode[] {
  // Индекс: commitId → row
  const commitRow = new Map<string, number>()
  for (let row = 0; row < nodes.length; row++) {
    commitRow.set(nodes[row].id, row)
  }

  // Для каждой строки собираем "активные линии" — lane'ы которые проходят через эту строку
  // Линия проходит через строку если есть edge (child→parent) где child.row < row < parent.row

  // Собираем все edges
  interface Edge {
    childRow: number
    parentRow: number
    childCol: number
    parentCol: number
    colorIdx: number
  }

  const edges: Edge[] = []
  for (let row = 0; row < nodes.length; row++) {
    const node = nodes[row]
    const childInfo = laneAssignment.get(node.id)!
    for (let p = 0; p < node.parents.length; p++) {
      const parentId = node.parents[p]
      const parentInfo = laneAssignment.get(parentId)
      if (!parentInfo) continue // parent за пределами набора

      const parentRow = commitRow.get(parentId)
      if (parentRow === undefined) continue

      edges.push({
        childRow: row,
        parentRow,
        childCol: childInfo.column,
        parentCol: parentInfo.column,
        colorIdx: p === 0 ? childInfo.colorIdx : parentInfo.colorIdx,
      })
    }
  }

  // Строим layout для каждой строки
  const layout: GraphLayoutNode[] = []

  for (let row = 0; row < nodes.length; row++) {
    const node = nodes[row]
    const info = laneAssignment.get(node.id)!
    const color = COLORS[info.colorIdx % COLORS.length]
    const lines: LayoutLine[] = []

    // Для каждого edge определяем какую линию он создаёт на этой строке
    for (const edge of edges) {
      if (edge.childRow > row || edge.parentRow < row) continue // edge не проходит через эту строку

      const edgeColor = COLORS[edge.colorIdx % COLORS.length]

      if (edge.childRow === row && edge.parentRow === row) {
        // Edge начинается и заканчивается на этой строке (shouldn't happen but handle)
        continue
      }

      if (edge.childRow === row) {
        // Edge НАЧИНАЕТСЯ на этой строке (от этого коммита вниз к parent'у)
        if (edge.childCol === edge.parentCol) {
          // Прямая линия вниз (same column) — нижняя половина
          lines.push({ x1: edge.childCol, x2: edge.childCol, type: LINE_BOTTOM_HALF, color: edgeColor })
        } else {
          // Fork — от текущей позиции к другой колонке
          lines.push({ x1: edge.childCol, x2: edge.parentCol, type: LINE_FORK, color: edgeColor })
        }
      } else if (edge.parentRow === row) {
        // Edge ЗАКАНЧИВАЕТСЯ на этой строке (приходит сверху к этому коммиту-parent'у)
        if (edge.childCol === edge.parentCol) {
          // Прямая линия сверху (same column) — верхняя половина
          lines.push({ x1: edge.parentCol, x2: edge.parentCol, type: LINE_TOP_HALF, color: edgeColor })
        } else {
          // MergeBack — от другой колонки к текущей
          lines.push({ x1: edge.parentCol, x2: edge.childCol, type: LINE_MERGE_BACK, color: edgeColor })
        }
      } else {
        // Edge ПРОХОДИТ через эту строку (child выше, parent ниже)
        // Определяем на какой колонке проходит линия
        // Для прямых edges (same column) — просто вертикальная линия
        if (edge.childCol === edge.parentCol) {
          lines.push({ x1: edge.childCol, x2: edge.childCol, type: LINE_FULL, color: edgeColor })
        } else {
          // Диагональный edge — на промежуточных строках рисуем вертикальную линию
          // на колонке child'а (линия "задерживается" на колонке пока не дойдёт до строки parent-1)
          // На строке parent-1 делаем curve
          if (row === edge.parentRow - 1) {
            // Предпоследняя строка перед parent'ом — рисуем merge curve
            lines.push({ x1: edge.parentCol, x2: edge.childCol, type: LINE_MERGE_BACK, color: edgeColor })
          } else if (row === edge.childRow + 1) {
            // Первая строка после child'а — fork
            lines.push({ x1: edge.childCol, x2: edge.parentCol, type: LINE_FORK, color: edgeColor })
          } else {
            // Промежуточные строки — вертикальная линия на колонке child'а
            lines.push({ x1: edge.childCol, x2: edge.childCol, type: LINE_FULL, color: edgeColor })
          }
        }
      }
    }

    layout.push({ column: info.column, color, lines })
  }

  return layout
}

/** Найти ближайший свободный lane к позиции `preferred`. */
function findFreeLane(lanes: (string | null)[], preferred: number): number {
  if (preferred < 0) preferred = 0

  // Попробовать preferred
  if (preferred < lanes.length && lanes[preferred] === null) return preferred
  if (preferred >= lanes.length) return preferred

  // Расширяемся в обе стороны
  for (let offset = 1; offset <= lanes.length + 1; offset++) {
    const right = preferred + offset
    if (right < lanes.length && lanes[right] === null) return right
    if (right >= lanes.length) return right

    const left = preferred - offset
    if (left >= 0 && lanes[left] === null) return left
  }

  return lanes.length
}

/** Расширить массив lanes если нужно. */
function ensureLaneExists(state: LaneState, index: number): void {
  while (state.lanes.length <= index) {
    state.lanes.push(null)
    state.colors.push(0)
  }
}
