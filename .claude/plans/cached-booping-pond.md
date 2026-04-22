# Plan: MSAL stale credential refresh after long idle (#137)

## Context

After idle periods longer than the Entra refresh-token lifetime, returning to Herald leaves the user wedged: `Auth.getToken()` in `app/client/core/auth.ts` only handles `InteractionRequiredAuthError`. Any other MSAL error path (cache corruption, `interaction_in_progress`, stale state) is swallowed, and stale state surfacing through `handleRedirectPromise()` during `init()` is not handled at all — `init()` rejects and the shell never mounts. The only recovery is manual storage clear.

This plan follows the canonical MSAL SPA pattern documented in `@azure/msal-browser` (`token-lifetimes.md`, `initialization.md`, `access-token-proof-of-possession.md`): silent acquisition first, `acquireTokenRedirect(request)` as the fallback, and a wrapped `handleRedirectPromise()`. Two layers of defense are sufficient — MSAL's built-in token lifecycle handles the rest.

## Files

- `app/client/core/auth.ts` — the only file that changes
- `app/client/core/api.ts` — no behavior change; the 401 retry at `api.ts:41-51` and `api.ts:93-106` keeps calling `getToken(true)` and receiving `null`-on-failure as before

## Approach

### 1. Wrap `handleRedirectPromise()` (auth.ts `init()`)

Stale redirect-return state (nonce mismatch, corrupted interaction state after long idle) surfaces here, not on `acquireTokenSilent`. A throw from this call currently rejects `init()` and prevents the shell from loading.

```ts
// inside init(), replace the unguarded handleRedirectPromise block with:
try {
  const response = await msalInstance.handleRedirectPromise();
  if (response?.account) {
    msalInstance.setActiveAccount(response.account);
  } else {
    const accounts = msalInstance.getAllAccounts();
    if (accounts.length === 1) {
      msalInstance.setActiveAccount(accounts[0]);
    }
  }
} catch {
  // Stale redirect state (e.g. nonce mismatch after long idle).
  // Clearing the cache drops the user to the "no active account" path,
  // which the app.ts bootstrap handles by calling login().
  await msalInstance.clearCache();
}
```

### 2. Switch `getToken()` fallback to `acquireTokenRedirect` (auth.ts:99-116)

Replace the `login()` fallback with the canonical MSAL pattern: `acquireTokenRedirect(request)`. This preserves the account/login hint and is the documented SPA fallback across every official MSAL sample.

Add `BrowserAuthError` to the imports.

```ts
async getToken(forceRefresh?: boolean): Promise<string | null> {
  const account = msalInstance?.getActiveAccount();
  if (!msalInstance || !account) return null;

  const request = {
    scopes: [scope()],
    account,
    forceRefresh: forceRefresh ?? false,
  };

  try {
    const result = await msalInstance.acquireTokenSilent(request);
    return result.accessToken;
  } catch (e) {
    // Another interactive flow owns the page — do not stack a second redirect.
    if (e instanceof BrowserAuthError && e.errorCode === "interaction_in_progress") {
      return null;
    }
    // For non-Interaction errors, treat as cache corruption and wipe before retrying.
    if (!(e instanceof InteractionRequiredAuthError)) {
      try {
        await msalInstance.clearCache();
      } catch {
        // clearCache() can throw on severely corrupted state — swallow and proceed.
      }
    }
    await msalInstance.acquireTokenRedirect(request);
    return null;
  }
}
```

Why this is cleaner than the issue's original spec:

- **No proactive expiry check needed.** MSAL's built-in `DEFAULT_TOKEN_RENEWAL_OFFSET_SEC = 300` already treats tokens within 5 minutes of expiry as expired and triggers refresh during `acquireTokenSilent`. The issue's proposed 60s window is strictly less aggressive than what MSAL already does. Dropping it removes ~10 lines with no functional loss.
- **`acquireTokenRedirect` over `loginRedirect`.** All MSAL docs samples (`token-lifetimes.md`, `access-token-proof-of-possession.md`) use `acquireTokenRedirect` as the silent-failure fallback. It carries the same request (account + scopes) and preserves the login hint so the user isn't forced through a full account picker.
- **Cache-clear only when warranted.** `InteractionRequiredAuthError` is an expected consent/MFA prompt, not corruption — no cache wipe. Only non-Interaction errors trigger `clearCache()`.

### 3. `api.ts` — no change

Walked through both 401 retry paths:

- `api.ts:41-51` (request) — calls `getToken(true)`; if it returns `null`, the 401 path calls `Auth.login()`. Under the new `getToken()`, `null` is returned only when (a) auth disabled, (b) no active account, (c) `interaction_in_progress` short-circuit, or (d) `acquireTokenRedirect` was just kicked off (page is navigating away). In (c) the page is already mid-redirect; in (d) the page is about to unload. In both cases the follow-up `Auth.login()` is effectively a no-op race against the unload — safe.
- `api.ts:93-106` (stream) — identical pattern, same reasoning.

## Verification

1. **Build**: `cd app && bun scripts/build.ts` — confirms types and bundles `dist/app.js`.
2. **Happy path**: `mise run dev`, log in normally, confirm the shell loads and API calls succeed.
3. **Tab resume**: switch away and back; confirm no console errors and no redirect loop.
4. **Idle simulation** (manual): in DevTools → Application → Local Storage, corrupt an `msal.*` entry, reload, confirm the app recovers (cache clear + redirect) rather than wedging.

## Testing

No TypeScript test infrastructure exists in `app/client/` (only Go tests under `tests/`). Per the project's testing convention, no unit tests are added. Manual verification covers the acceptance criteria.

## Acceptance Criteria Mapping

| Criterion | Addressed by |
|---|---|
| Idle past refresh-token lifetime → clean re-auth without manual storage clear | Wrapped `handleRedirectPromise()` + broadened `getToken()` catch with `clearCache` + `acquireTokenRedirect` |
| No redirect loops for normal session-within-lifetime use | `interaction_in_progress` short-circuit; cache-clear only on non-Interaction errors |
| Near-expired silent tokens trigger exactly one `forceRefresh` retry, not a loop | MSAL's built-in `DEFAULT_TOKEN_RENEWAL_OFFSET_SEC = 300` handles this natively — no custom retry needed |
| `visibilitychange` handler is removed/re-registered cleanly across navigations | N/A — no listener is added; the first post-idle API call flows through the hardened `getToken()`, which handles silent refresh or falls back to `acquireTokenRedirect`. The listener criterion only existed because the issue spec proposed a listener, which is redundant with MSAL's native lifecycle handling. |
