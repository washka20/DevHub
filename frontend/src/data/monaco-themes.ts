import { siteThemes } from './site-themes'
import type { SiteTheme } from './site-themes'

export interface MonacoThemeDef {
  base: 'vs' | 'vs-dark'
  inherit: boolean
  rules: Array<{ token: string; foreground?: string; fontStyle?: string }>
  colors: Record<string, string>
}

function strip(hex: string): string {
  return hex.replace('#', '')
}

function isLight(bg: string): boolean {
  const h = bg.replace('#', '')
  const r = parseInt(h.substring(0, 2), 16)
  const g = parseInt(h.substring(2, 4), 16)
  const b = parseInt(h.substring(4, 6), 16)
  return (r * 299 + g * 587 + b * 114) / 1000 > 128
}

export function buildMonacoTheme(themeName: string): MonacoThemeDef {
  const t: SiteTheme = siteThemes[themeName] || siteThemes['github-dark']
  const light = isLight(t['--bg-primary'])
  const strColor = light ? '0a3069' : 'a5d6ff'
  const opColor = light ? 'cf222e' : 'ff7b72'

  return {
    base: light ? 'vs' : 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: strip(t['--text-secondary']), fontStyle: 'italic' },
      { token: 'keyword', foreground: strip(t['--accent-red']) },
      { token: 'string', foreground: strColor },
      { token: 'string.escape', foreground: strip(t['--accent-orange']) },
      { token: 'number', foreground: strip(t['--accent-blue']) },
      { token: 'constant', foreground: strip(t['--accent-blue']) },
      { token: 'type', foreground: strip(t['--accent-orange']) },
      { token: 'type.identifier', foreground: strip(t['--accent-orange']) },
      { token: 'function', foreground: strip(t['--accent-purple']) },
      { token: 'variable', foreground: strip(t['--text-primary']) },
      { token: 'tag', foreground: strip(t['--accent-green']) },
      { token: 'attribute.name', foreground: strip(t['--accent-blue']) },
      { token: 'attribute.value', foreground: strColor },
      { token: 'operator', foreground: opColor },
      { token: 'delimiter', foreground: strip(t['--text-secondary']) },
      { token: 'regexp', foreground: strip(t['--accent-blue']) },
    ],
    colors: {
      'editor.background': t['--bg-primary'],
      'editor.foreground': t['--text-primary'],
      'editor.lineHighlightBackground': light ? '#00000008' : '#ffffff08',
      'editorCursor.foreground': t['--accent-blue'],
      'editor.selectionBackground': light ? '#0969da26' : '#58a6ff33',
      'editorLineNumber.foreground': t['--text-secondary'] + '80',
      'editorLineNumber.activeForeground': t['--text-secondary'],
      'editorGutter.background': t['--bg-primary'],
      'editorBracketMatch.background': light ? '#0969da1a' : '#58a6ff33',
      'editorBracketMatch.border': t['--border'],
      'minimap.background': t['--bg-secondary'],
    },
  }
}

export function monacoThemeId(name: string): string {
  return `devhub-${name}`
}
