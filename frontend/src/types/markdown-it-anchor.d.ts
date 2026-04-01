declare module 'markdown-it-anchor' {
  import type MarkdownIt from 'markdown-it'
  interface PermalinkOpts {
    safariReaderFix?: boolean
    class?: string
  }
  interface AnchorOptions {
    permalink?: unknown
    slugify?: (s: string) => string
  }
  const plugin: MarkdownIt.PluginWithOptions<AnchorOptions> & {
    permalink: { headerLink: (opts?: PermalinkOpts) => unknown }
  }
  export default plugin
}
