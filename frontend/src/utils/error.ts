/**
 * Safely extract error message from unknown catch parameter.
 */
export function getErrorMessage(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
