import type { DiffLine } from '../types'

export function parseDiff(raw: string): DiffLine[] {
  const lines = raw.split('\n')
  const result: DiffLine[] = []
  let oldLine = 0
  let newLine = 0

  for (const line of lines) {
    if (line.startsWith('@@')) {
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
      if (match) {
        oldLine = parseInt(match[1], 10)
        newLine = parseInt(match[2], 10)
      }
      result.push({ type: 'header', content: line, oldLineNo: null, newLineNo: null })
    } else if (line.startsWith('diff ') || line.startsWith('index ') || line.startsWith('---') || line.startsWith('+++')) {
      result.push({ type: 'header', content: line, oldLineNo: null, newLineNo: null })
    } else if (line.startsWith('+')) {
      result.push({ type: 'add', content: line.substring(1), oldLineNo: null, newLineNo: newLine })
      newLine++
    } else if (line.startsWith('-')) {
      result.push({ type: 'remove', content: line.substring(1), oldLineNo: oldLine, newLineNo: null })
      oldLine++
    } else {
      const content = line.startsWith(' ') ? line.substring(1) : line
      if (line === '') continue
      result.push({ type: 'context', content, oldLineNo: oldLine, newLineNo: newLine })
      oldLine++
      newLine++
    }
  }

  return result
}
