# 137 - MSAL stale credential refresh after long idle

## Summary

Hardened `app/client/core/auth.ts` against the "wedged after long idle" state where users had to manually clear storage and cookies to recover. The fix wraps `handleRedirectPromise()` in `init()` so stale redirect-return state does not reject the init promise, broadens the `getToken()` catch so non-Interaction errors trigger a cache clear, and switches the silent-failure fallback from `loginRedirect()` to `acquireTokenRedirect(request)` — the documented MSAL SPA pattern.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Proactive expiry check (≤60s + `forceRefresh`) | Dropped | MSAL's built-in `DEFAULT_TOKEN_RENEWAL_OFFSET_SEC = 300` already treats tokens within 5 minutes of expiry as expired and refreshes them automatically. The issue's 60s window is strictly less aggressive — pure dead code. |
| `visibilitychange` listener | Dropped | Only shifts *when* the same operations happen (tab-focus instead of first click). Arguably worse UX — would redirect users to Entra the instant they refocus the tab, before any action. Two layers of defense (wrapped `handleRedirectPromise` + broadened `getToken` catch) are sufficient. |
| Silent-failure fallback | `acquireTokenRedirect(request)` | Every official MSAL docs sample (`token-lifetimes.md`, `access-token-proof-of-possession.md`, `v1-migration.md`) uses `acquireTokenRedirect`, not `loginRedirect`. Preserves account/login-hint context so the user is not dropped back to a full account picker. |
| `interaction_in_progress` handling | Short-circuit to `null` | Calling `acquireTokenRedirect` on top of an in-flight flow would throw immediately. Returning `null` lets the existing redirect complete. |
| Cache-clear scope | Non-Interaction errors only | `InteractionRequiredAuthError` is an expected consent/MFA prompt, not corruption. Clearing cache unnecessarily would force the user through a full account picker on normal re-auth. |

## Files Modified

- `app/client/core/auth.ts` — added `BrowserAuthError` import, wrapped `handleRedirectPromise()` in try/catch, rewrote `getToken()` with the canonical MSAL SPA pattern, refreshed JSDoc on `init()` and `getToken()`

## Patterns Established

- **Canonical MSAL SPA error handling**: future code that calls `acquireTokenSilent` should follow this three-branch catch shape — `interaction_in_progress` short-circuit, `InteractionRequiredAuthError` passthrough to `acquireTokenRedirect`, everything else as cache corruption with `clearCache()` + `acquireTokenRedirect`.
- **`handleRedirectPromise()` must be wrapped**: any future MSAL initialization must guard this call because stale state (nonce mismatch after long idle) can otherwise reject the init promise and leave the shell unmounted.
- **MSAL v5 cache is encrypted**: future diagnostic work on the browser cache must account for the `{id, nonce, data, lastUpdatedAt}` envelope. Field-level edits (e.g., `expiresOn`) are not possible; delete entries wholesale and let MSAL re-acquire. MSAL v5 also auto-removes entries it cannot decrypt, so corruption does not reliably surface errors to application code.

## Validation Results

- `bun scripts/build.ts` completes cleanly, `dist/app.js` + `dist/app.css` rebuild
- `go vet ./...` clean
- Manual scenarios from the implementation guide all passed:
  - **Scenario 1** (happy path): normal login and use, no console errors
  - **Scenario 2** (missing access token, valid refresh token): MSAL-native silent refresh completed in ~1s via a single `/oauth2/v2.0/token` POST
  - **Scenario 3** (missing access + refresh tokens): clean redirect to Entra, silent re-authentication, ~4-5s round trip
  - **Scenario 4** (simulated non-Interaction error): cache cleared + `acquireTokenRedirect` fired, then normal operation resumed (one-shot `sessionStorage` gate kept the test from looping)

## Patterns Rejected

- **Naive always-throw instrumentation for Scenario 4** produced an infinite redirect loop because every post-redirect `getToken()` call threw again. The one-shot `sessionStorage.getItem("__s4_fired")` gate — which survives `clearCache()` because that method only touches `msal.*` keys — is the correct pattern for this kind of test harness.
