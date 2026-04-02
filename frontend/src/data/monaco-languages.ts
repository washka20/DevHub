const map: Record<string, string> = {
  typescript: 'typescript',
  javascript: 'javascript',
  vue: 'html',
  html: 'html',
  css: 'css',
  scss: 'scss',
  json: 'json',
  python: 'python',
  go: 'go',
  php: 'php',
  sql: 'sql',
  yaml: 'yaml',
  markdown: 'markdown',
  xml: 'xml',
  rust: 'rust',
  shell: 'shell',
  text: 'plaintext',
}

export function toMonacoLanguage(lang: string): string {
  return map[lang] ?? 'plaintext'
}
