import ReactMarkdown from 'react-markdown'
import type { ComponentPropsWithoutRef, ReactNode } from 'react'
import remarkGfm from 'remark-gfm'
import { Children, isValidElement } from 'react'
import { slugify } from './markdownHeadings'

function nodeText(children: ReactNode): string {
  return Children.toArray(children)
    .map((child) => {
      if (typeof child === 'string') return child
      if (typeof child === 'number') return String(child)
      if (isValidElement<{ children?: ReactNode }>(child)) return nodeText(child.props.children)
      return ''
    })
    .join('')
}

/**
 * MarkdownView renders GFM Markdown with stable heading ids so the page can
 * build a table of contents. It is intentionally dependency-light and reuses
 * the same styles as the editor preview.
 */
export function MarkdownView({ content }: { content: string }) {
  const headingId = (children: ReactNode) => slugify(nodeText(children))

  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children, ...props }: ComponentPropsWithoutRef<'h1'>) => (
            <h1 id={headingId(children)} {...props}>
              {children}
            </h1>
          ),
          h2: ({ children, ...props }: ComponentPropsWithoutRef<'h2'>) => (
            <h2 id={headingId(children)} {...props}>
              {children}
            </h2>
          ),
          h3: ({ children, ...props }: ComponentPropsWithoutRef<'h3'>) => (
            <h3 id={headingId(children)} {...props}>
              {children}
            </h3>
          ),
          a: ({ href = '', ...props }: ComponentPropsWithoutRef<'a'>) => (
            <a href={href} target="_blank" rel="noreferrer noopener" {...props} />
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
