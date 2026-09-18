import { expect, test } from '@playwright/test'
import { unique } from './helpers'

test.describe('markdown import', () => {
  test('imports a front-matter file into the target section', async ({ page }) => {
    const title = unique('Imported')
    const filename = `${title}.md`
    const contents = `---\ntitle: ${title}\ntags: [imported, e2e]\n---\n\n# ${title}\n\nImported body.`

    await page.goto('/workflows')
    await page.getByRole('button', { name: 'Import' }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Import Markdown')).toBeVisible()

    // Upload via the hidden file input.
    await dialog.locator('input[type="file"]').setInputFiles({
      name: filename,
      mimeType: 'text/markdown',
      buffer: Buffer.from(contents),
    })
    await expect(dialog.getByText(filename)).toBeVisible()

    await dialog
      .getByRole('button', { name: /Import/ })
      .last()
      .click()

    // Result summary and per-file row.
    await expect(dialog.getByText('1 created')).toBeVisible()
    await expect(dialog.locator('.import-result-row__status')).toContainText('created')

    // The imported document is listed and viewable.
    await dialog.getByRole('button', { name: 'Done' }).click()
    await expect(page.getByText(title)).toBeVisible()
  })
})
