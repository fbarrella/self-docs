import { expect, type APIRequestContext, type Page } from '@playwright/test'

/** Unique-ish suffix so runs do not collide on the same database. */
export function unique(prefix: string): string {
  return `${prefix}-${Date.now().toString(36)}-${Math.floor(Math.random() * 1e4)}`
}

/**
 * ensureMasterPassword configures the master password when none is set yet so
 * the private-unlock smoke test is deterministic. Returns the password to use,
 * or null when an unknown password is already configured (test should skip).
 */
export async function ensureMasterPassword(
  request: APIRequestContext,
  password: string,
): Promise<string | null> {
  const response = await request.get('/api/settings')
  expect(response.ok()).toBeTruthy()
  const body = (await response.json()) as { data: { master_password_set: boolean } }
  if (body.data.master_password_set) {
    return password
  }
  const set = await request.put('/api/settings/master-password', {
    data: { current_password: '', new_password: password },
  })
  return set.ok() ? password : null
}

/** createDocument creates a document via the API and returns its id. */
export async function createDocument(
  request: APIRequestContext,
  input: { title: string; content?: string; section?: string; tags?: string[] },
): Promise<string> {
  const response = await request.post('/api/documents', {
    data: {
      title: input.title,
      content: input.content ?? '',
      section: input.section ?? 'workflow',
      tags: input.tags ?? [],
    },
  })
  expect(response.ok()).toBeTruthy()
  const body = (await response.json()) as { id: string }
  return body.id
}

/** gotoDashboard navigates home and waits for the hero to render. */
export async function gotoDashboard(page: Page): Promise<void> {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Your Personal Knowledge Base.' })).toBeVisible()
}
