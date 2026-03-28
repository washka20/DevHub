export interface SiteTheme {
  '--bg-primary': string
  '--bg-secondary': string
  '--bg-tertiary': string
  '--border': string
  '--text-primary': string
  '--text-secondary': string
  '--accent-blue': string
  '--accent-green': string
  '--accent-red': string
  '--accent-orange': string
  '--accent-purple': string
  '--glow-green': string
  '--glow-blue': string
  '--glow-orange': string
  '--glow-red': string
  '--glow-purple': string
}

export const siteThemes: Record<string, SiteTheme> = {
  'github-dark': {
    '--bg-primary': '#0d1117',
    '--bg-secondary': '#161b22',
    '--bg-tertiary': '#1c2128',
    '--border': '#30363d',
    '--text-primary': '#f0f6fc',
    '--text-secondary': '#8b949e',
    '--accent-blue': '#58a6ff',
    '--accent-green': '#3fb950',
    '--accent-red': '#f85149',
    '--accent-orange': '#d29922',
    '--accent-purple': '#bc8cff',
    '--glow-green': '0 0 8px rgba(63, 185, 80, 0.4)',
    '--glow-blue': '0 0 8px rgba(88, 166, 255, 0.4)',
    '--glow-orange': '0 0 8px rgba(210, 153, 34, 0.4)',
    '--glow-red': '0 0 8px rgba(248, 81, 73, 0.4)',
    '--glow-purple': '0 0 8px rgba(188, 140, 255, 0.4)',
  },
  'dracula': {
    '--bg-primary': '#282a36',
    '--bg-secondary': '#21222c',
    '--bg-tertiary': '#2d2f3d',
    '--border': '#44475a',
    '--text-primary': '#f8f8f2',
    '--text-secondary': '#6272a4',
    '--accent-blue': '#bd93f9',
    '--accent-green': '#50fa7b',
    '--accent-red': '#ff5555',
    '--accent-orange': '#ffb86c',
    '--accent-purple': '#ff79c6',
    '--glow-green': '0 0 8px rgba(80, 250, 123, 0.4)',
    '--glow-blue': '0 0 8px rgba(189, 147, 249, 0.4)',
    '--glow-orange': '0 0 8px rgba(255, 184, 108, 0.4)',
    '--glow-red': '0 0 8px rgba(255, 85, 85, 0.4)',
    '--glow-purple': '0 0 8px rgba(255, 121, 198, 0.4)',
  },
  'one-dark': {
    '--bg-primary': '#282c34',
    '--bg-secondary': '#21252b',
    '--bg-tertiary': '#2c313a',
    '--border': '#3e4451',
    '--text-primary': '#abb2bf',
    '--text-secondary': '#5c6370',
    '--accent-blue': '#61afef',
    '--accent-green': '#98c379',
    '--accent-red': '#e06c75',
    '--accent-orange': '#d19a66',
    '--accent-purple': '#c678dd',
    '--glow-green': '0 0 8px rgba(152, 195, 121, 0.4)',
    '--glow-blue': '0 0 8px rgba(97, 175, 239, 0.4)',
    '--glow-orange': '0 0 8px rgba(209, 154, 102, 0.4)',
    '--glow-red': '0 0 8px rgba(224, 108, 117, 0.4)',
    '--glow-purple': '0 0 8px rgba(198, 120, 221, 0.4)',
  },
  'nord': {
    '--bg-primary': '#2e3440',
    '--bg-secondary': '#272c36',
    '--bg-tertiary': '#3b4252',
    '--border': '#434c5e',
    '--text-primary': '#eceff4',
    '--text-secondary': '#7b88a1',
    '--accent-blue': '#81a1c1',
    '--accent-green': '#a3be8c',
    '--accent-red': '#bf616a',
    '--accent-orange': '#d08770',
    '--accent-purple': '#b48ead',
    '--glow-green': '0 0 8px rgba(163, 190, 140, 0.4)',
    '--glow-blue': '0 0 8px rgba(129, 161, 193, 0.4)',
    '--glow-orange': '0 0 8px rgba(208, 135, 112, 0.4)',
    '--glow-red': '0 0 8px rgba(191, 97, 106, 0.4)',
    '--glow-purple': '0 0 8px rgba(180, 142, 173, 0.4)',
  },
  'monokai': {
    '--bg-primary': '#272822',
    '--bg-secondary': '#1e1f1c',
    '--bg-tertiary': '#2d2e2a',
    '--border': '#3e3d32',
    '--text-primary': '#f8f8f2',
    '--text-secondary': '#75715e',
    '--accent-blue': '#66d9ef',
    '--accent-green': '#a6e22e',
    '--accent-red': '#f92672',
    '--accent-orange': '#fd971f',
    '--accent-purple': '#ae81ff',
    '--glow-green': '0 0 8px rgba(166, 226, 46, 0.4)',
    '--glow-blue': '0 0 8px rgba(102, 217, 239, 0.4)',
    '--glow-orange': '0 0 8px rgba(253, 151, 31, 0.4)',
    '--glow-red': '0 0 8px rgba(249, 38, 114, 0.4)',
    '--glow-purple': '0 0 8px rgba(174, 129, 255, 0.4)',
  },
  'solarized-dark': {
    '--bg-primary': '#002b36',
    '--bg-secondary': '#00212b',
    '--bg-tertiary': '#073642',
    '--border': '#094050',
    '--text-primary': '#eee8d5',
    '--text-secondary': '#839496',
    '--accent-blue': '#268bd2',
    '--accent-green': '#859900',
    '--accent-red': '#dc322f',
    '--accent-orange': '#cb4b16',
    '--accent-purple': '#6c71c4',
    '--glow-green': '0 0 8px rgba(133, 153, 0, 0.4)',
    '--glow-blue': '0 0 8px rgba(38, 139, 210, 0.4)',
    '--glow-orange': '0 0 8px rgba(203, 75, 22, 0.4)',
    '--glow-red': '0 0 8px rgba(220, 50, 47, 0.4)',
    '--glow-purple': '0 0 8px rgba(108, 113, 196, 0.4)',
  },
  'tokyo-night': {
    '--bg-primary': '#1a1b26',
    '--bg-secondary': '#16161e',
    '--bg-tertiary': '#1f2335',
    '--border': '#292e42',
    '--text-primary': '#c0caf5',
    '--text-secondary': '#565f89',
    '--accent-blue': '#7aa2f7',
    '--accent-green': '#9ece6a',
    '--accent-red': '#f7768e',
    '--accent-orange': '#e0af68',
    '--accent-purple': '#bb9af7',
    '--glow-green': '0 0 8px rgba(158, 206, 106, 0.4)',
    '--glow-blue': '0 0 8px rgba(122, 162, 247, 0.4)',
    '--glow-orange': '0 0 8px rgba(224, 175, 104, 0.4)',
    '--glow-red': '0 0 8px rgba(247, 118, 142, 0.4)',
    '--glow-purple': '0 0 8px rgba(187, 154, 247, 0.4)',
  },
}

export const siteThemeNames: Record<string, string> = {
  'github-dark': 'GitHub Dark',
  'dracula': 'Dracula',
  'one-dark': 'One Dark',
  'nord': 'Nord',
  'monokai': 'Monokai',
  'solarized-dark': 'Solarized Dark',
  'tokyo-night': 'Tokyo Night',
}
