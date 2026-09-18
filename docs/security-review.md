# self-docs - Private Vault Security Review

Scope: the Private Archive master-password flow and the private-isolation
invariants from PRD 3.2, 3.3 and 5, plus T2.8 hardening. Reviewed against the
code as of T6.2 (2026-09-18). Single-user, local-first deployment (PRD 5).

## 1. Summary

The Private Archive is protected by a bcrypt-hashed master password, short-lived
sliding in-memory sessions, a login rate limiter, and a repository layer that
excludes private content from every public read path. A review of every endpoint
found one leak (public tag deletion could target private-only tags) which has
been fixed and covered by a regression test.

## 2. Password handling

- Hashing: bcrypt (`golang.org/x/crypto/bcrypt`) with the library default cost.
  The plaintext is never stored, logged, or returned.
- Storage: only `settings.master_password_hash`. No endpoint returns it;
  `GET /api/settings` exposes only a boolean `master_password_set`.
- Verification: `bcrypt.CompareHashAndPassword`, which compares in constant time.
- Longest input is capped at bcrypt's 72-byte limit; longer passwords are
  rejected (`ErrPasswordTooLong`) rather than silently truncated.
- Seeding: `MASTER_PASSWORD_HASH` is written once at boot only when the stored
  value is empty, so a restart cannot overwrite a password changed in the UI.
- Change flow: `PUT /api/settings/master-password` requires a valid private
  session or the correct current password, validates the new length, stores the
  new hash, and clears **all** sessions so previously unlocked clients cannot
  continue.

## 3. Sessions

- Tokens are 32 bytes from `crypto/rand`, encoded base64url (opaque, not
  guessable).
- Stored in memory only (`SessionStore`), so sessions end on process restart.
  This is acceptable for a single-process, local-first deployment.
- Lifetime: 15 minutes, sliding on each authenticated private request.
- Transport: `Authorization: Bearer <token>` or the `selfdocs_private` HttpOnly
  cookie (Secure flag configurable; scoped to `/api/private`).
- `POST /api/private/lock` invalidates the token; `SessionStore.Clear` wipes all
  tokens when the master password changes.
- A 401 on a private request clears the client token (frontend).

## 4. Brute-force protection

- `POST /api/private/unlock` is rate limited per client key: 5 failed attempts
  within 15 minutes trigger a 429 lockout for the same window. Successful
  unlocks reset the counter.
- The client key is `ClientIP`. Gin is configured with an empty trusted-proxy
  list by default, so `X-Forwarded-For` is ignored unless `TRUSTED_PROXIES` is
  set. This prevents spoofing the rate-limit key. Set `TRUSTED_PROXIES` to your
  reverse proxy's CIDR(s) when deployed behind one.
- Wrong-password responses are a generic `incorrect password` (no distinction
  between "not configured" and "wrong").

## 5. Private isolation (leak audit)

Invariant: `is_private = true` documents never appear in public listings, tags,
search, dashboard, or activity, regardless of session state.

| Endpoint | Private handling | Verified |
| --- | --- | --- |
| `POST /api/documents` | `section=private` → 403 | test |
| `GET /api/documents` | `is_private=false` predicate; `section=private` → 403 | test |
| `GET /api/documents/:id` | private treated as 404 | test |
| `PUT`/`DELETE /api/documents/:id` | private treated as 404 | test |
| `GET /api/documents/tree` | `section=private` → 403; query filters private | test |
| `POST /api/documents/import` | `section=private` → 403 | test |
| `GET /api/tags` | private-only tags hidden | test |
| `GET /api/tags/popular` | counts exclude private docs | test |
| `GET /api/tags/:name/documents` | private-only tag → 404 | test |
| `DELETE /api/tags/:name` | private-only tag → 404, not deleted | **fixed in T6.2** |
| `GET /api/search` | always excludes private; `section=private` → 403 | test |
| `GET /api/dashboard` | recently_updated / popular_tags / activity exclude private | test |
| `GET /api/activity` | private rows are never written or returned | test |
| `GET /api/private/*` | requires a valid session (401 otherwise) | test |
| `GET /api/settings` | returns no secret material | test |

### 5.1 Finding fixed in T6.2

`DELETE /api/tags/:name` previously deleted any tag by name without checking
whether it was attached only to private documents. A public caller could (a)
distinguish a private-only tag from an unknown one by status code and (b) delete
it. Fixed by requiring `TagRepository.GetVisibleByNormalized` first, so a
private-only tag behaves as unknown (404) and is not removed. Cache invalidation
was also added. Regression coverage in
`backend/internal/handler/tag_handler_test.go` (TestTagsPrivateIsolation).

### 5.2 Other leak vectors considered

- **Side channels:** search snippets, dashboard counts, tag counts, and activity
  text are all computed from `is_private=false` queries; deleting a document
  keeps an activity title snapshot only for non-private actions.
- **Errors:** repository errors map to generic messages; private documents
  return `not_found` rather than `forbidden`, which avoids confirming existence.
- **Timing:** password comparison is constant time; unlock responses do not
  reveal whether the password is unset (generic 401).
- **Unlock in the activity feed:** the `unlocked` action carries no document and
  no title, so it leaks nothing about private content.

## 6. Request hardening (T2.8)

- Request IDs, structured access logs, panic recovery, and security headers
  (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`).
- JSON bodies capped (1 MiB); imports capped per file (`MAX_IMPORT_BYTES`),
  returning 413 on violation.
- CORS restricted to `CORS_ORIGINS` with credentials allowed.
- Internal errors are logged server-side and returned as a generic 500.

## 7. Residual risks and future work

1. **Encryption at rest (PRD 5):** private content is currently stored in
   plaintext `documents.content`. A database compromise exposes it. Planned:
   add `content_encrypted bytea` + `content_nonce`, derive a key from the master
   password, null the plaintext column for private rows, and exclude private
   content from `search_vector` (see `docs/data-model.md` section 7).
2. **Sessions are in-memory:** a multi-replica deployment would need a shared
   session store (Redis) or signed stateless tokens. Not a concern for the
   supported single-instance topology.
3. **Rate limiter is in-memory and per-process:** restarts reset counters; a
   distributed deployment would need a shared limiter.
4. **No TLS by default:** the master password and session token travel in
   plaintext unless the deployment terminates TLS. Deploy behind HTTPS and set
   `secure` cookies (the router supports `SecureCookies`).

## 8. Checklist

- [x] Master password hashed with bcrypt; never returned or logged.
- [x] Constant-time password comparison.
- [x] Opaque, random session tokens with sliding 15-minute expiry.
- [x] Unlock rate limited; lockout enforced for correct and incorrect passwords.
- [x] Rate-limit key not spoofable via forwarded headers by default.
- [x] No public endpoint returns private documents, titles, counts, tags, or
      activity.
- [x] Private-only tags are invisible and cannot be deleted publicly.
- [x] Changing the master password invalidates all sessions.
- [x] No secret material in `/api/settings`.
- [ ] Encryption at rest for private content (future work, PRD 5).
