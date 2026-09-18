import { expect, test } from '@playwright/test'
import { createDocument, gotoDashboard, unique } from './helpers'

test.describe('dashboard', () => {
  test('renders hero, cards, and sections', async ({ page, request }) => {
    const title = unique('Dashboard Doc')
    await createDocument(request, { title, content: 'dashboard smoke content' })

    await gotoDashboard(page)

    // Four navigation cards.
    await expect(page.getByRole('heading', { name: 'Workflows & Guides' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Project Notes' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Cheat Sheets' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Private Archive' })).toBeVisible()

    // Lower area sections.
    await expect(page.getByRole('heading', { name: 'Recently Updated' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Popular Tags' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Activity Feed' })).toBeVisible()

    // The created document appears in Recently Updated (title also appears in
    // the activity feed, so scope to the list row).
    await expect(page.locator('.ui-list-row__title', { hasText: title })).toBeVisible()
  })

  test('global search from the hero navigates to results', async ({ page, request }) => {
    const term = unique('searchterm')
    await createDocument(request, { title: `Searchable ${term}`, content: `body ${term}` })

    await gotoDashboard(page)
    await page.getByLabel('Global search').fill(term)
    await page.getByRole('button', { name: 'Search' }).first().click()

    await expect(page).toHaveURL(new RegExp(`/search\\?q=${term}`))
    await expect(page.getByText(`Searchable ${term}`)).toBeVisible()
  })
})
