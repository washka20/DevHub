export function parseDiffToOriginalModified(unifiedDiff: string): { original: string; modified: string } {
  const lines = unifiedDiff.split('\n')
  const originalLines: string[] = []
  const modifiedLines: string[] = []
  let inHunk = false

  for (const line of lines) {
    if (line.startsWith('@@')) {
      inHunk = true
      continue
    }

    if (!inHunk) continue

    if (line.startsWith('+')) {
      modifiedLines.push(line.substring(1))
    } else if (line.startsWith('-')) {
      originalLines.push(line.substring(1))
    } else if (line.startsWith(' ')) {
      originalLines.push(line.substring(1))
      modifiedLines.push(line.substring(1))
    } else if (line === '') {
      originalLines.push('')
      modifiedLines.push('')
    }
  }

  return {
    original: originalLines.join('\n'),
    modified: modifiedLines.join('\n'),
  }
}

export function detectLanguageFromFilename(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  const map: Record<string, string> = {
    ts: 'typescript',
    tsx: 'typescript',
    js: 'javascript',
    jsx: 'javascript',
    vue: 'html',
    html: 'html',
    htm: 'html',
    css: 'css',
    scss: 'scss',
    less: 'css',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    go: 'go',
    py: 'python',
    php: 'php',
    sql: 'sql',
    xml: 'xml',
    rs: 'rust',
    sh: 'shell',
    bash: 'shell',
  }
  return map[ext] ?? 'plaintext'
}
