import { expect, test } from '@playwright/test'
import { ensureMasterPassword, unique } from './helpers'

const MASTER_PASSWORD = process.env.E2E_MASTER_PASSWORD ?? 'e2e-master-pass'

test.describe('private archive', () => {
  test('locked route prompts, wrong password errors, correct unlocks', async ({
    page,
    request,
  }) => {
    const configured = await ensureMasterPassword(request, MASTER_PASSWORD)
    test.skip(configured === null, 'master password already set to an unknown value')

    // Navigating to /private while locked opens the master-password modal.
    await page.goto('/private')
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText('Unlock Private Archive')).toBeVisible()

    // Wrong password surfaces an error.
    await dialog.getByLabel('Master password').fill('definitely-wrong')
    await dialog.getByRole('button', { name: 'Unlock' }).click()
    await expect(dialog.getByText('Incorrect password')).toBeVisible()

    // Correct password reveals the private archive.
    await dialog.getByLabel('Master password').fill(configured!)
    await dialog.getByRole('button', { name: 'Unlock' }).click()
    await expect(page.getByRole('heading', { name: 'Private Archive', level: 1 })).toBeVisible()
    await expect(page.getByText('Unlocked')).toBeVisible()

    // Create a private note and confirm it is not searchable publicly.
    const secretTitle = unique('Private Note')
    const secretTerm = unique('secrettoken')
    await page.getByRole('button', { name: 'New private note' }).first().click()
    await page.getByRole('textbox', { name: 'Title' }).fill(secretTitle)
    await page.locator('.w-md-editor textarea').fill(`private body ${secretTerm}`)
    await page.getByRole('button', { name: 'Create note' }).click()
    await expect(page.getByRole('heading', { name: secretTitle })).toBeVisible()

    // Public search must never return the private content.
    await page.goto(`/search?q=${secretTerm}`)
    await expect(page.getByText('No results found')).toBeVisible()

    // Lock again via the header menu.
    await page.goto('/private')
    await page.getByRole('button', { name: 'Lock' }).click()
    await expect(page.getByText('Private Archive is locked')).toBeVisible()
  })
})
