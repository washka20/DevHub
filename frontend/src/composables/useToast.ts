import { ref } from 'vue'

export interface Toast {
  id: number
  type: 'success' | 'info' | 'error'
  message: string
}

const toasts = ref<Toast[]>([])
let nextId = 0

function show(type: Toast['type'], message: string, duration = 3000) {
  const id = ++nextId
  toasts.value.push({ id, type, message })
  if (toasts.value.length > 3) {
    toasts.value.shift()
  }
  setTimeout(() => remove(id), duration)
}

function remove(id: number) {
  const idx = toasts.value.findIndex(t => t.id === id)
  if (idx !== -1) toasts.value.splice(idx, 1)
}

export function useToast() {
  return {
    toasts,
    show,
    remove,
  }
}
