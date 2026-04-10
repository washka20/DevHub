import { ref, computed, type Ref } from 'vue'

interface VirtualScrollReturn {
  scrollTop: Ref<number>
  containerHeight: Ref<number>
  visibleRange: Ref<{ startIdx: number; endIdx: number }>
  offsetY: Ref<number>
  totalHeight: Ref<number>
  onScroll: (e: Event) => void
}

export function useVirtualScroll(
  totalItems: Ref<number>,
  itemHeight: number,
  overscan = 10,
): VirtualScrollReturn {
  const scrollTop = ref(0)
  const containerHeight = ref(600)

  const totalHeight = computed(() => totalItems.value * itemHeight)

  const visibleRange = computed(() => {
    const startIdx = Math.max(0, Math.floor(scrollTop.value / itemHeight) - overscan)
    const endIdx = Math.min(
      totalItems.value,
      Math.ceil((scrollTop.value + containerHeight.value) / itemHeight) + overscan,
    )
    return { startIdx, endIdx }
  })

  const offsetY = computed(() => visibleRange.value.startIdx * itemHeight)

  function onScroll(e: Event) {
    const el = e.target as HTMLElement
    scrollTop.value = el.scrollTop
    containerHeight.value = el.clientHeight
  }

  return {
    scrollTop,
    containerHeight,
    visibleRange,
    offsetY,
    totalHeight,
    onScroll,
  }
}
