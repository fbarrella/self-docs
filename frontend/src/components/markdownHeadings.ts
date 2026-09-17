/** Helpers for Markdown rendering, kept separate so components stay pure. */

export interface TocHeading {
  id: string
  text: string
  level: number
}

/** slugify produces heading-anchor ids. */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}

/**
 * collectHeadings extracts h1-h3 headings from raw Markdown, ignoring fenced
 * code blocks. It uses the same slug scheme as the rendered heading ids.
 */
export function collectHeadings(content: string): TocHeading[] {
  const headings: TocHeading[] = []
  let inFence = false

  for (const rawLine of content.split('\n')) {
    const line = rawLine.trimEnd()
    if (line.trimStart().startsWith('```')) {
      inFence = !inFence
      continue
    }
    if (inFence) continue

    const match = /^(#{1,3})\s+(.*)$/.exec(line)
    if (!match) continue
    const level = match[1].length
    const text = match[2].replace(/#+\s*$/, '').trim()
    if (!text) continue
    headings.push({ id: slugify(text), text, level })
  }
  return headings
}
