import { expect, test } from '@playwright/test'
import { unique } from './helpers'

test.describe('document authoring', () => {
  test('create, view, edit, and delete a document', async ({ page }) => {
    const title = unique('E2E Doc')
    const editedTitle = `${title} (edited)`

    // Create via the editor UI. Scope by textbox role because the Markdown
    // toolbar also exposes an "Insert title" control.
    await page.goto('/editor')
    await expect(page.getByRole('heading', { name: 'New document' })).toBeVisible()
    await page.getByRole('textbox', { name: 'Title' }).fill(title)
    await page.getByRole('textbox', { name: 'Tags' }).fill('e2e, smoke')
    await page.getByRole('textbox', { name: 'Tags' }).press('Enter')

    // MDEditor exposes a textarea; type Markdown into it.
    const editor = page.locator('.w-md-editor textarea')
    await editor.fill('# Heading\n\nSome **markdown** body.')
    await page.getByRole('button', { name: 'Create document' }).click()

    // Redirected to the viewer with rendered content.
    await expect(page).toHaveURL(/\/documents\/[0-9a-f-]+/)
    const url = page.url()
    await expect(page.getByRole('heading', { name: title, level: 1 })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Heading' })).toBeVisible()
    await expect(page.getByText('Some')).toBeVisible()

    // Edit. Saving stays on the editor and confirms via status text.
    await page.getByRole('button', { name: 'Edit' }).click()
    await expect(page.getByRole('heading', { name: 'Edit document' })).toBeVisible()
    const titleInput = page.getByRole('textbox', { name: 'Title' })
    await titleInput.fill(editedTitle)
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(page.locator('.toast--success').last()).toContainText('Document saved')

    // Re-visit the viewer to confirm the persisted title.
    await page.goto(url)
    await expect(page.getByRole('heading', { name: editedTitle, level: 1 })).toBeVisible()

    // Delete from the editor (confirm the browser dialog).
    await page.getByRole('button', { name: 'Edit' }).click()
    await expect(page.getByRole('heading', { name: 'Edit document' })).toBeVisible()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByRole('button', { name: 'Delete' }).click()
    await expect(page).toHaveURL(/\/documents$/)
    await expect(page.getByText(editedTitle)).toHaveCount(0)
  })

  test('listing filters by tag', async ({ page, request }) => {
    const tag = unique('tag')
    await request.post('/api/documents', {
      data: { title: unique('Tagged'), content: 'x', section: 'workflow', tags: [tag] },
    })

    await page.goto('/documents')
    // Popular tags are limited; search the tag via the URL instead.
    await page.goto(`/documents?tag=${tag}`)
    await expect(page.getByText('No documents found')).toHaveCount(0)
    await expect(page.locator('.doc-card')).toHaveCount(1)
  })
})
