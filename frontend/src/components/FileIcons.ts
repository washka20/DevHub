// Material Icon Theme — inline SVG icons for file tree
// Maps file extensions and folder names to colored SVG strings

const fileIcons: Record<string, string> = {
  vue: '<svg viewBox="0 0 24 24"><path fill="#41b883" d="M1.791 3.851 12 21.471 22.209 3.936V3.85H18.24l-6.18 10.616L5.906 3.851z"/><path fill="#35495e" d="m5.907 3.851 6.152 10.617L18.24 3.851h-3.723L12.084 8.03 9.66 3.85z"/></svg>',

  ts: '<svg viewBox="0 0 16 16"><path fill="#0288d1" d="M2 2v12h12V2zm4 6h3v1H8v4H7V9H6zm5 0h2v1h-2v1h1a1.003 1.003 0 0 1 1 1v1a1.003 1.003 0 0 1-1 1h-2v-1h2v-1h-1a1.003 1.003 0 0 1-1-1V9a1.003 1.003 0 0 1 1-1"/></svg>',

  tsx: '<svg viewBox="0 0 16 16"><path fill="#0288d1" d="M2 2v12h12V2zm4 6h3v1H8v4H7V9H6zm5 0h2v1h-2v1h1a1.003 1.003 0 0 1 1 1v1a1.003 1.003 0 0 1-1 1h-2v-1h2v-1h-1a1.003 1.003 0 0 1-1-1V9a1.003 1.003 0 0 1 1-1"/></svg>',

  js: '<svg viewBox="0 0 16 16"><path fill="#f5de19" d="M2 2v12h12V2zm5.5 9.5c0 .828-.335 1.5-1.5 1.5s-1.5-.672-1.5-1.5h1c0 .276.224.5.5.5s.5-.224.5-.5V8h1zm3.5 1.5c-1.165 0-1.75-.672-1.75-1.5h1c0 .276.335.5.75.5s.75-.224.75-.5c0-.828-2.5-.5-2.5-2 0-.828.585-1.5 1.75-1.5s1.75.672 1.75 1.5h-1c0-.276-.335-.5-.75-.5s-.75.224-.75.5c0 .828 2.5.5 2.5 2 0 .828-.585 1.5-1.75 1.5z"/></svg>',

  jsx: '<svg viewBox="0 0 16 16"><path fill="#f5de19" d="M2 2v12h12V2zm5.5 9.5c0 .828-.335 1.5-1.5 1.5s-1.5-.672-1.5-1.5h1c0 .276.224.5.5.5s.5-.224.5-.5V8h1zm3.5 1.5c-1.165 0-1.75-.672-1.75-1.5h1c0 .276.335.5.75.5s.75-.224.75-.5c0-.828-2.5-.5-2.5-2 0-.828.585-1.5 1.75-1.5s1.75.672 1.75 1.5h-1c0-.276-.335-.5-.75-.5s-.75.224-.75.5c0 .828 2.5.5 2.5 2 0 .828-.585 1.5-1.75 1.5z"/></svg>',

  go: '<svg viewBox="0 0 32 32"><path fill="#00acc1" d="M2 12h4v2H2zm-2 4h6v2H0zm4 4h2v2H4zm16.954-5H14v3h3.239a4.42 4.42 0 0 1-3.531 2 2.65 2.65 0 0 1-2.053-.858 2.86 2.86 0 0 1-.628-2.28A4.515 4.515 0 0 1 15.292 13a2.73 2.73 0 0 1 1.749.584l2.962-1.185A5.6 5.6 0 0 0 15.292 10a7.526 7.526 0 0 0-7.243 6.5 5.614 5.614 0 0 0 5.659 6.5 7.526 7.526 0 0 0 7.243-6.5 6.4 6.4 0 0 0 .003-1.5"/><path fill="#00acc1" d="M26.292 10a7.526 7.526 0 0 0-7.243 6.5 5.614 5.614 0 0 0 5.659 6.5 7.526 7.526 0 0 0 7.243-6.5 5.614 5.614 0 0 0-5.659-6.5m2.681 6.137A4.515 4.515 0 0 1 24.708 20a2.65 2.65 0 0 1-2.053-.858 2.86 2.86 0 0 1-.628-2.28A4.515 4.515 0 0 1 26.292 13a2.65 2.65 0 0 1 2.053.858 2.86 2.86 0 0 1 .628 2.28Z"/></svg>',

  php: '<svg viewBox="0 0 16 16"><path fill="#7e57c2" d="M8 2C3.6 2 0 4.7 0 8s3.6 6 8 6 8-2.7 8-6-3.6-6-8-6zM4.7 9.5H3.8v1.3H2.7V6.2h2c1 0 1.7.7 1.7 1.6s-.7 1.7-1.7 1.7zm4.6 0H8.4l-.2 1.3H7.1l1-5.6h2c1 0 1.7.7 1.7 1.6 0 1-.7 1.7-1.7 1.7h-.8zm4.3-1.1c0 1-.7 1.6-1.7 1.6h-.8v1.3H10V6.2h2c1 0 1.6.7 1.6 1.6v.6z"/></svg>',

  json: '<svg viewBox="0 -960 960 960"><path fill="#f9a825" d="M560-160v-80h120q17 0 28.5-11.5T720-280v-80q0-38 22-69t58-44v-14q-36-13-58-44t-22-69v-80q0-17-11.5-28.5T680-720H560v-80h120q50 0 85 35t35 85v80q0 17 11.5 28.5T840-560h40v160h-40q-17 0-28.5 11.5T800-360v80q0 50-35 85t-85 35zm-280 0q-50 0-85-35t-35-85v-80q0-17-11.5-28.5T120-400H80v-160h40q17 0 28.5-11.5T160-600v-80q0-50 35-85t85-35h120v80H280q-17 0-28.5 11.5T240-680v80q0 38-22 69t-58 44v14q36 13 58 44t22 69v80q0 17 11.5 28.5T280-240h120v80z"/></svg>',

  css: '<svg viewBox="0 0 16 16"><path fill="#42a5f5" d="M2 1l1.2 13L8 16l4.8-2L14 1zm9 4.5H5.8l.2 2h4.7l-.4 4.5L8 13l-2.3-.8-.1-2h1.5l.1 1 .8.3.8-.3.1-1.5H5.3L4.9 4h6.2z"/></svg>',

  yaml: '<svg viewBox="0 0 24 24"><path fill="#ff5252" d="M13 9h5.5L13 3.5zM6 2h8l6 6v12c0 1.1-.9 2-2 2H6c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2m12 16v-2H9v2zm-4-4v-2H6v2z"/></svg>',

  yml: '<svg viewBox="0 0 24 24"><path fill="#ff5252" d="M13 9h5.5L13 3.5zM6 2h8l6 6v12c0 1.1-.9 2-2 2H6c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2m12 16v-2H9v2zm-4-4v-2H6v2z"/></svg>',

  md: '<svg viewBox="0 0 32 32"><path fill="#42a5f5" d="m14 10-4 3.5L6 10H4v12h4v-6l2 2 2-2v6h4V10zm12 6v-6h-4v6h-4l6 8 6-8z"/></svg>',

  py: '<svg viewBox="0 0 16 16"><path fill="#3572a5" d="M8 1C5.2 1 5.5 2.2 5.5 2.2v1.3H8v.5H3.5S1 3.7 1 8s2.2 4 2.2 4h1.3V10.5s-.1-2.2 2.2-2.2h3.8s2.1 0 2.1-2V3.2S13 1 8 1zM5.8 2.4c.4 0 .7.3.7.7s-.3.7-.7.7-.7-.3-.7-.7.3-.7.7-.7z"/><path fill="#fdd835" d="M8 15c2.8 0 2.5-1.2 2.5-1.2v-1.3H8v-.5h4.5S15 12.3 15 8s-2.2-4-2.2-4h-1.3v1.5s.1 2.2-2.2 2.2H5.5s-2.1 0-2.1 2v3.1S3 15 8 15zm2.2-1.4c-.4 0-.7-.3-.7-.7s.3-.7.7-.7.7.3.7.7-.3.7-.7.7z"/></svg>',

  sql: '<svg viewBox="0 0 16 16"><path fill="#e37933" d="M8 1C4.7 1 2 2.3 2 4v8c0 1.7 2.7 3 6 3s6-1.3 6-3V4c0-1.7-2.7-3-6-3zm0 2c2.8 0 4.5.9 4.5 1s-1.7 1-4.5 1S3.5 5.1 3.5 5 5.2 3 8 3z"/></svg>',

  xml: '<svg viewBox="0 0 16 16"><path fill="#e44d26" d="M2 1l1.2 13L8 16l4.8-2L14 1zm9.5 4.5H5.8l.2 2h5.2l-.4 4.5L8 13l-2.8-.8-.2-2h1.6l.1 1 1.3.4 1.3-.4.1-1.5H5.3L4.9 4h6.8z"/></svg>',

  html: '<svg viewBox="0 0 16 16"><path fill="#e44d26" d="M2 1l1.2 13L8 16l4.8-2L14 1zm9.5 4.5H5.8l.2 2h5.2l-.4 4.5L8 13l-2.8-.8-.2-2h1.6l.1 1 1.3.4 1.3-.4.1-1.5H5.3L4.9 4h6.8z"/></svg>',

  htm: '<svg viewBox="0 0 16 16"><path fill="#e44d26" d="M2 1l1.2 13L8 16l4.8-2L14 1zm9.5 4.5H5.8l.2 2h5.2l-.4 4.5L8 13l-2.8-.8-.2-2h1.6l.1 1 1.3.4 1.3-.4.1-1.5H5.3L4.9 4h6.8z"/></svg>',

  rs: '<svg viewBox="0 0 16 16"><path fill="#dea584" d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zm0 1.5a.8.8 0 1 1 0 1.6.8.8 0 0 1 0-1.6zM5 5h6v1H9v4h1v1H6v-1h1V6H5z"/></svg>',

  scss: '<svg viewBox="0 0 16 16"><path fill="#cd6799" d="M8 1C4.1 1 1 4.1 1 8s3.1 7 7 7 7-3.1 7-7-3.1-7-7-7zm3.6 10.1c-.3.5-.8.8-1.3.9-.6.1-1.2 0-1.7-.3-.3-.2-.5-.4-.7-.6-.4.2-.8.3-1.2.4-.7.1-1.3 0-1.8-.4-.3-.2-.5-.5-.6-.9-.1-.4 0-.8.2-1.1.4-.5.9-.8 1.5-.8.6-.1 1.1.1 1.5.4-.4-.6-.6-1.2-.5-1.9.1-.6.4-1.2.8-1.6.6-.6 1.3-.8 2.1-.7.7.1 1.2.5 1.5 1.1.2.5.2 1 0 1.5-.3.7-.8 1.2-1.4 1.5.2.2.3.5.3.8 0 .4-.1.7-.4 1-.2.2-.4.3-.7.3z"/></svg>',
}

const defaultFileIcon = '<svg viewBox="0 0 24 24" fill="none"><path fill="#42a5f5" d="M8 16h8v2H8zm0-4h8v2H8zm6-10H6c-1.1 0-2 .9-2 2v16c0 1.1.89 2 1.99 2H18c1.1 0 2-.9 2-2V8zm4 18H6V4h7v5h5z"/></svg>'

// Folder colors by directory name
const folderColors: Record<string, string> = {
  src: '#4caf50',
  lib: '#4caf50',
  views: '#ff7043',
  pages: '#ff7043',
  components: '#c0ca33',
  composables: '#c0ca33',
  hooks: '#c0ca33',
  stores: '#ff7043',
  store: '#ff7043',
  api: '#fbc02d',
  server: '#fbc02d',
  internal: '#fbc02d',
  config: '#00acc1',
  configs: '#00acc1',
  public: '#42a5f5',
  assets: '#42a5f5',
  static: '#42a5f5',
  test: '#ab47bc',
  tests: '#ab47bc',
  __tests__: '#ab47bc',
  docs: '#42a5f5',
  types: '#0288d1',
  utils: '#00acc1',
  helpers: '#00acc1',
  node_modules: '#8b949e',
  dist: '#8b949e',
  build: '#8b949e',
  cmd: '#fbc02d',
  pkg: '#fbc02d',
  handlers: '#ff7043',
  middleware: '#ff7043',
  router: '#ff7043',
  styles: '#cd6799',
  icons: '#c0ca33',
}

function getFolderIcon(dirname: string, isOpen: boolean): string {
  const color = folderColors[dirname.toLowerCase()] ?? '#42a5f5'
  if (isOpen) {
    return `<svg viewBox="0 0 32 32"><path fill="${color}" d="m27.4 8H15.124a2 2 0 0 1-1.28-.464l-1.288-1.072A2 2 0 0 0 11.276 6H4a2 2 0 0 0-2 2v3h28v-1a2 2 0 0 0-2-2z"/><path fill="${color}" opacity="0.7" d="M2 26a2 2 0 0 0 2 2h24a2 2 0 0 0 2-2V12H2z"/></svg>`
  }
  return `<svg viewBox="0 0 32 32"><path fill="${color}" d="m13.844 7.536-1.288-1.072A2 2 0 0 0 11.276 6H4a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h24a2 2 0 0 0 2-2V10a2 2 0 0 0-2-2H15.124a2 2 0 0 1-1.28-.464"/></svg>`
}

function getFileTypeIcon(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  return fileIcons[ext] ?? defaultFileIcon
}

export function getFileIcon(filename: string, isDir: boolean, isOpen: boolean): string {
  if (isDir) return getFolderIcon(filename, isOpen)
  return getFileTypeIcon(filename)
}
