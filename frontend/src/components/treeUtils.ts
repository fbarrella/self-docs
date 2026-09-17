import type { DocumentNode } from '../api/types'

/**
 * findNodePath returns the chain of nodes from a root to the node with the
 * given id, used to render breadcrumbs.
 */
export function findNodePath(
  nodes: DocumentNode[],
  id: string,
  trail: DocumentNode[] = [],
): DocumentNode[] {
  for (const node of nodes) {
    const next = [...trail, node]
    if (node.id === id) return next
    const found = findNodePath(node.children, id, next)
    if (found.length > 0) return found
  }
  return []
}
