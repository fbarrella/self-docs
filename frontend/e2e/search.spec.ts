import { expect, test } from '@playwright/test'
import { createDocument, unique } from './helpers'

test.describe('search', () => {
  test('finds public documents by content and shows tags', async ({ page, request }) => {
    const term = unique('contentterm')
    await createDocument(request, {
      title: unique('Search Hit'),
      content: `contains ${term} in the body`,
      tags: ['searchtag'],
    })

    await page.goto(`/search?q=${term}`)
    await expect(page.getByText('1 result for')).toBeVisible()
    await expect(page.locator('.search-result')).toHaveCount(1)
    // The term may be split into several <mark> fragments by the highlighter.
    await expect(page.locator('.search-result__snippet')).toContainText(term)
  })

  test('shows a zero-state for unmatched queries', async ({ page }) => {
    await page.goto(`/search?q=${unique('nomatch')}`)
    await expect(page.getByText('No results found')).toBeVisible()
  })

  test('prompts before a query is entered', async ({ page }) => {
    await page.goto('/search')
    await expect(page.getByText('Search your knowledge base')).toBeVisible()
  })
})
