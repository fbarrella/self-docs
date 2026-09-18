import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright config for the self-docs E2E smoke suite.
 *
 * By default the suite targets the Docker Compose stack (PRD 5) at
 * http://localhost:8080. Start it first with:
 *
 *   docker compose up --build -d
 *   cd frontend && npm run test:e2e
 *
 * Override the target with BASE_URL (e.g. a dev server) when needed.
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
