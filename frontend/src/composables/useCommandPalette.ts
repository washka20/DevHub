import { ref } from 'vue'

const open = ref(false)

export function openCommandPalette() { open.value = true }
export function closeCommandPalette() { open.value = false }
export function toggleCommandPalette() { open.value = !open.value }

export function useCommandPalette() {
  return { open, openCommandPalette, closeCommandPalette, toggleCommandPalette }
}

export interface CommandItem {
  id: string
  label: string
  group: string
  hint?: string
  /** Plain-string keywords for fuzzy match in addition to label */
  keywords?: string
  /** Keyboard shortcut label shown at the right of the row */
  shortcut?: string
  /** Optional inline SVG markup for the left glyph */
  iconHtml?: string
  /** Invoked when the item is selected */
  run: () => void | Promise<void>
}

const RECENT_KEY = 'devhub.palette.recent'
const RECENT_MAX = 5

export function loadRecentIds(): string[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed.filter((s) => typeof s === 'string') : []
  } catch { return [] }
}

export function recordRecent(id: string) {
  const cur = loadRecentIds().filter((x) => x !== id)
  cur.unshift(id)
  const next = cur.slice(0, RECENT_MAX)
  try { localStorage.setItem(RECENT_KEY, JSON.stringify(next)) } catch {}
}

/** Simple fuzzy: keep items whose lowercased haystack contains every query char in order. */
export function fuzzyFilter(items: CommandItem[], query: string): CommandItem[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  const out: { item: CommandItem; score: number }[] = []
  for (const it of items) {
    const hay = (it.label + ' ' + (it.keywords || '') + ' ' + it.group).toLowerCase()
    let idx = 0
    for (const ch of q) {
      idx = hay.indexOf(ch, idx)
      if (idx < 0) break
      idx++
    }
    if (idx > 0) {
      // shorter haystack + prefix match = higher score
      const prefix = hay.startsWith(q) ? 100 : 0
      const lengthBias = 50 - Math.min(50, hay.length / 4)
      out.push({ item: it, score: prefix + lengthBias })
    }
  }
  out.sort((a, b) => b.score - a.score)
  return out.map((x) => x.item)
}
