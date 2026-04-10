/**
 * Shorten a CWD path, replacing /home/user with ~.
 */
export function shortCwd(cwd: string): string {
  const home = '/home/'
  const idx = cwd.indexOf('/', home.length)
  if (idx > 0) return '~' + cwd.slice(idx)
  return cwd
}
