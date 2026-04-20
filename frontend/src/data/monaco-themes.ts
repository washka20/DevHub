export interface MonacoThemeDef {
  base: 'vs' | 'vs-dark'
  inherit: boolean
  rules: Array<{ token: string; foreground?: string; fontStyle?: string }>
  colors: Record<string, string>
}

type Mode = 'dark' | 'light'

function strip(hex: string): string {
  return hex.replace('#', '')
}

/**
 * Resolves a CSS custom property at runtime to a hex (or hex-like) string Monaco
 * can consume. Falls back to a sensible warm-palette value if the variable hasn't
 * been applied yet (theme-aware fallback).
 */
function readVar(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  // Monaco accepts only hex (#rrggbb / #rrggbbaa). If the token is OKLCH or
  // color-mix(), fall back so the editor stays readable until we get a hex.
  if (!v || !v.startsWith('#')) return fallback
  return v
}

const DARK_FALLBACK = {
  bg0: '#17140f',
  bg1: '#1f1b14',
  bg2: '#2a251c',
  fg:  '#f5efe0',
  fg2: '#bdb3a1',
  fg3: '#8a8170',
  line: '#3d3528',
  accent: '#d7a965',
  ok:   '#7fc591',
  bad:  '#e07a73',
  warn: '#d8a85a',
  info: '#7faecc',
  mag:  '#b58cc8',
}

const LIGHT_FALLBACK = {
  bg0: '#faf7f0',
  bg1: '#ffffff',
  bg2: '#f2ece0',
  fg:  '#1c1810',
  fg2: '#5b5240',
  fg3: '#8a8170',
  line: '#d9d0b9',
  accent: '#bf8138',
  ok:   '#3a8a55',
  bad:  '#bf3d2f',
  warn: '#a87a32',
  info: '#3471a8',
  mag:  '#8b3f9c',
}

function palette(mode: Mode) {
  const fb = mode === 'light' ? LIGHT_FALLBACK : DARK_FALLBACK
  return {
    bg0:    readVar('--bg-0',    fb.bg0),
    bg1:    readVar('--bg-1',    fb.bg1),
    bg2:    readVar('--bg-2',    fb.bg2),
    fg:     readVar('--fg',      fb.fg),
    fg2:    readVar('--fg-2',    fb.fg2),
    fg3:    readVar('--fg-3',    fb.fg3),
    line:   readVar('--line',    fb.line),
    accent: fb.accent,  // OKLCH tokens — keep fallback
    ok:     fb.ok,
    bad:    fb.bad,
    warn:   fb.warn,
    info:   fb.info,
    mag:    fb.mag,
  }
}

export function buildMonacoTheme(mode: Mode = 'dark'): MonacoThemeDef {
  const p = palette(mode)
  const isLight = mode === 'light'

  return {
    base: isLight ? 'vs' : 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: strip(p.fg3), fontStyle: 'italic' },
      { token: 'keyword', foreground: strip(p.bad) },
      { token: 'string', foreground: strip(p.info) },
      { token: 'string.escape', foreground: strip(p.warn) },
      { token: 'number', foreground: strip(p.info) },
      { token: 'constant', foreground: strip(p.info) },
      { token: 'type', foreground: strip(p.warn) },
      { token: 'type.identifier', foreground: strip(p.warn) },
      { token: 'function', foreground: strip(p.mag) },
      { token: 'variable', foreground: strip(p.fg) },
      { token: 'tag', foreground: strip(p.ok) },
      { token: 'attribute.name', foreground: strip(p.info) },
      { token: 'attribute.value', foreground: strip(p.info) },
      { token: 'operator', foreground: strip(p.bad) },
      { token: 'delimiter', foreground: strip(p.fg3) },
      { token: 'regexp', foreground: strip(p.info) },
    ],
    colors: {
      'editor.background': p.bg0,
      'editor.foreground': p.fg,
      'editor.lineHighlightBackground': isLight ? '#00000008' : '#ffffff08',
      'editorCursor.foreground': p.accent,
      'editor.selectionBackground': isLight ? '#bf81384d' : '#d7a96540',
      'editorLineNumber.foreground': p.fg3 + '80',
      'editorLineNumber.activeForeground': p.fg2,
      'editorGutter.background': p.bg0,
      'editorBracketMatch.background': isLight ? '#bf813833' : '#d7a96533',
      'editorBracketMatch.border': p.line,
      'minimap.background': p.bg1,
    },
  }
}

export function monacoThemeId(mode: Mode = 'dark'): string {
  return `devhub-${mode}`
}
