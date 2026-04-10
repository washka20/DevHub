import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import go from 'highlight.js/lib/languages/go'
import typescript from 'highlight.js/lib/languages/typescript'
import javascript from 'highlight.js/lib/languages/javascript'
import bash from 'highlight.js/lib/languages/bash'
import yaml from 'highlight.js/lib/languages/yaml'
import json from 'highlight.js/lib/languages/json'
import css from 'highlight.js/lib/languages/css'
import sql from 'highlight.js/lib/languages/sql'
import python from 'highlight.js/lib/languages/python'
import xml from 'highlight.js/lib/languages/xml'
import rust from 'highlight.js/lib/languages/rust'
import diff from 'highlight.js/lib/languages/diff'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import markdown from 'highlight.js/lib/languages/markdown'

hljs.registerLanguage('go', go)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('json', json)
hljs.registerLanguage('css', css)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('python', python)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('vue', xml)
hljs.registerLanguage('rust', rust)
hljs.registerLanguage('diff', diff)
hljs.registerLanguage('dockerfile', dockerfile)
hljs.registerLanguage('markdown', markdown)

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  highlight(str: string, lang: string) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${hljs.highlight(str, { language: lang }).value}</code></pre>`
      } catch { /* fallback */ }
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  },
})

// Task list rule: convert [ ] and [x] to interactive checkboxes
md.core.ruler.after('inline', 'task-lists', (state) => {
  const tokens = state.tokens
  for (let i = 0; i < tokens.length; i++) {
    if (tokens[i].type !== 'inline') continue
    const content = tokens[i].content
    if (!/^\[[ xX]\]\s/.test(content)) continue

    const checked = content[1] !== ' '
    tokens[i].content = content.slice(3).trim()

    for (let j = i - 1; j >= 0; j--) {
      if (tokens[j].type === 'list_item_open') {
        tokens[j].attrJoin('class', 'task-list-item')
        break
      }
    }
    for (let j = i - 1; j >= 0; j--) {
      if (tokens[j].type === 'bullet_list_open') {
        tokens[j].attrJoin('class', 'task-list')
        break
      }
    }

    const checkToken = new state.Token('html_inline', '', 0)
    checkToken.content = `<input type="checkbox" class="md-checkbox" ${checked ? 'checked' : ''} disabled />`
    tokens[i].children?.unshift(checkToken)
  }
})

// Rewrite GitLab image URLs to go through our authenticated proxy.
// Handles:
//   - /uploads/hash/file.png  (relative to project)
//   - https://gitlab.host/group/project/uploads/hash/file.png (absolute, same host)
const defaultImageRender = md.renderer.rules.image || function (tokens: any[], idx: number, options: any, _env: any, self: any) {
  return self.renderToken(tokens, idx, options)
}

md.renderer.rules.image = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  const src = token.attrGet('src')
  if (src && env.projectWebUrl) {
    const proxyUrl = toProxyUrl(src, env.projectWebUrl)
    if (proxyUrl) token.attrSet('src', proxyUrl)
  }
  return defaultImageRender(tokens, idx, options, env, self)
}

function toProxyUrl(src: string, projectWebUrl: string): string | null {
  // Relative /uploads/... path → construct full URL, proxy
  if (src.startsWith('/uploads/')) {
    const fullUrl = projectWebUrl + src
    return `/api/gitlab/proxy?url=${encodeURIComponent(fullUrl)}`
  }

  // Absolute URL on the same GitLab host → proxy
  try {
    const srcHost = new URL(src).host
    const gitlabHost = new URL(projectWebUrl).host
    if (srcHost === gitlabHost) {
      return `/api/gitlab/proxy?url=${encodeURIComponent(src)}`
    }
  } catch {
    // Not a valid URL, leave as-is
  }

  return null
}

export interface MarkdownRenderOptions {
  projectWebUrl?: string
}

export function useMarkdown() {
  function render(source: string, opts?: MarkdownRenderOptions): string {
    if (!source) return ''
    const env: Record<string, unknown> = {}
    if (opts?.projectWebUrl) {
      env.projectWebUrl = opts.projectWebUrl
    }
    return md.render(source, env)
  }

  return { render }
}
