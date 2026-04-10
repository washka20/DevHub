/**
 * Relative time: "just now", "5m ago", "2h ago", "3d ago", or locale date.
 */
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

/**
 * Locale-formatted date: "Apr 10, 2026".
 */
export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

/**
 * Check if a date string is in the past.
 */
export function isOverdue(dateStr: string | null): boolean {
  if (!dateStr) return false
  return new Date(dateStr) < new Date()
}

/**
 * Formatted date with "(overdue)" suffix when applicable.
 */
export function formatDueDate(dateStr: string): string {
  const formatted = formatDate(dateStr)
  return isOverdue(dateStr) ? `${formatted} (overdue)` : formatted
}
