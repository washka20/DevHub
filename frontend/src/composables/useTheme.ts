import { ref, watch } from 'vue'

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'devhub.theme'

function readInitial(): Theme {
  if (typeof document !== 'undefined') {
    const attr = document.documentElement.getAttribute('data-theme')
    if (attr === 'dark' || attr === 'light') return attr
  }
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'dark' || saved === 'light') return saved
  } catch {}
  return 'dark'
}

const theme = ref<Theme>(readInitial())

function apply(t: Theme) {
  document.documentElement.setAttribute('data-theme', t)
  try {
    localStorage.setItem(STORAGE_KEY, t)
  } catch {}
}

watch(theme, apply, { immediate: true })

export function useTheme() {
  function setTheme(t: Theme) { theme.value = t }
  function toggleTheme() { theme.value = theme.value === 'dark' ? 'light' : 'dark' }
  return { theme, setTheme, toggleTheme }
}
