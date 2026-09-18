/**
 * Private Archive session token storage.
 *
 * The backend also sets an HttpOnly cookie, but keeping the bearer token in
 * sessionStorage makes the session resilient to cross-origin cookie policies
 * during local development and lets us attach Authorization explicitly.
 */

const STORAGE_KEY = 'selfdocs.privateToken'
const EXPIRY_KEY = 'selfdocs.privateExpiresAt'

let memoryToken: string | null = null

function readStorage(key: string): string | null {
  try {
    return window.sessionStorage.getItem(key)
  } catch {
    return null
  }
}

function writeStorage(key: string, value: string): void {
  try {
    window.sessionStorage.setItem(key, value)
  } catch {
    /* storage unavailable; memory fallback remains */
  }
}

function removeStorage(key: string): void {
  try {
    window.sessionStorage.removeItem(key)
  } catch {
    /* ignore */
  }
}

/** setPrivateToken stores the token and its expiry. */
export function setPrivateToken(token: string, expiresAt: string): void {
  memoryToken = token
  writeStorage(STORAGE_KEY, token)
  writeStorage(EXPIRY_KEY, expiresAt)
}

/** getPrivateToken returns the token if present and unexpired. */
export function getPrivateToken(): string | null {
  if (memoryToken) return memoryToken
  const token = readStorage(STORAGE_KEY)
  const expiry = readStorage(EXPIRY_KEY)
  if (!token) return null
  if (expiry && new Date(expiry).getTime() <= Date.now()) {
    clearPrivateToken()
    return null
  }
  memoryToken = token
  return token
}

/** clearPrivateToken removes the stored token. */
export function clearPrivateToken(): void {
  memoryToken = null
  removeStorage(STORAGE_KEY)
  removeStorage(EXPIRY_KEY)
}
