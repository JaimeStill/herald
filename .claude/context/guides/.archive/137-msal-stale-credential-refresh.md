# 137 - MSAL stale credential refresh after long idle

## Problem Context

After idle periods longer than the Entra refresh-token lifetime, returning to Herald leaves the user wedged. Inspection of `app/client/core/auth.ts:99-116` shows `Auth.getToken()` only handles `InteractionRequiredAuthError`; any other MSAL error path (cache corruption, nonce mismatch after long idle, `interaction_in_progress` conflict) silently returns `null`. Additionally, stale state that surfaces through `handleRedirectPromise()` during `init()` is not caught at all — the `init()` promise rejects and the shell never mounts. Users have to clear storage and cookies manually to recover.

## Architecture Approach

Follow the canonical MSAL SPA pattern documented in `@azure/msal-browser` — silent acquisition first, `acquireTokenRedirect(request)` as the fallback, and a wrapped `handleRedirectPromise()`:

1. **Wrap `handleRedirectPromise()`** in `init()` so stale redirect-return state (nonce mismatch, corrupted interaction state) does not reject the init promise. On failure, clear cache and let the app.ts bootstrap take the "no active account" path.
2. **Broaden the `getToken()` catch** to treat any non-Interaction error as cache corruption: clear the cache and fall through to `acquireTokenRedirect(request)`. Short-circuit on `BrowserAuthError` with code `interaction_in_progress` to avoid stacking a second redirect.
3. **Switch the fallback from `login()` to `acquireTokenRedirect(request)`** — this is the documented SPA pattern across every official MSAL sample. It preserves account/login-hint context.

Rejected refinements:
- **No proactive 60s expiry check.** MSAL's built-in `DEFAULT_TOKEN_RENEWAL_OFFSET_SEC = 300` already treats tokens within 5 minutes of expiry as expired and refreshes them during `acquireTokenSilent`. The issue's proposed 60s window is strictly less aggressive; adding it is dead code.
- **No `visibilitychange` listener.** The issue spec proposed one, but MSAL's native token lifecycle already handles refresh on the first post-idle API call, and the hardened `getToken()` catch handles failures cleanly. A listener would only shift *when* those same operations happen (tab-focus instead of first click), with a questionable UX tradeoff — it would redirect the user to Entra the moment they focus the tab rather than in response to an action they took. Two layers of defense (wrapped `handleRedirectPromise` + broadened catch) are sufficient.

## Implementation

### Step 1: Update `@azure/msal-browser` imports

`app/client/core/auth.ts` — add `BrowserAuthError` to the value imports:

```ts
import {
  BrowserAuthError,
  InteractionRequiredAuthError,
  PublicClientApplication,
} from "@azure/msal-browser";
```

### Step 2: Wrap `handleRedirectPromise()` in `init()`

`app/client/core/auth.ts` — wrap the existing `handleRedirectPromise` block (including the subsequent active-account selection) in a try/catch:

```ts
    try {
      const response: AuthenticationResult | null =
        await msalInstance.handleRedirectPromise();

      if (response?.account) {
        msalInstance.setActiveAccount(response.account);
      } else {
        const accounts = msalInstance.getAllAccounts();
        if (accounts.length === 1) {
          msalInstance.setActiveAccount(accounts[0]);
        }
      }
    } catch {
      await msalInstance.clearCache();
    }
```

### Step 3: Rewrite `getToken()` to follow the canonical MSAL pattern

`app/client/core/auth.ts` — replace the `getToken()` method body:

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
      if (
        e instanceof BrowserAuthError &&
        e.errorCode === "interaction_in_progress"
      ) {
        return null;
      }
      if (!(e instanceof InteractionRequiredAuthError)) {
        try {
          await msalInstance.clearCache();
        } catch {
          // clearCache() can throw on severely corrupted state; swallow.
        }
      }
      await msalInstance.acquireTokenRedirect(request);
      return null;
    }
  },
```

## Manual Validation

MSAL state lives in browser storage; use DevTools → **Application** to inspect and manipulate it.

### Where MSAL stores state

- **Local Storage** (our `cacheLocation` default) — persistent cache. In MSAL v5, token entries are **encrypted envelopes**: `{id, nonce, data, lastUpdatedAt}` where `data` is the AES-encrypted JSON blob containing `secret`, `expiresOn`, `target`, etc. You cannot directly edit `expiresOn` — surgical field editing isn't possible. You can, however, **delete** entries wholesale.
  - `msal.account.keys` — JSON array of account key hashes
  - `<homeAccountId>-<environment>-<realm>` — account object (encrypted envelope)
  - `<homeAccountId>-<environment>-accesstoken-<clientId>-<realm>-<scopes>` — access token (encrypted envelope)
  - `<homeAccountId>-<environment>-refreshtoken-<clientId>--` — refresh token (encrypted envelope)
  - `<homeAccountId>-<environment>-idtoken-<clientId>-<realm>-` — ID token (encrypted envelope)
  - `msal.token.keys.<clientId>` — plaintext JSON index listing access/refresh/id token keys
- **Session Storage** — transient interaction state (plaintext):
  - `msal.<clientId>.interaction.status` — set to `"interaction_in_progress"` during a redirect flow
  - `msal.<clientId>.request.state.<guid>`, `msal.<clientId>.nonce.idtoken` — per-request nonce/state

**Important MSAL v5 behavior:** the cache manager calls `isEncrypted()` on every read and silently **removes any entry it can't decrypt or parse** (`BrowserCacheManager.d.ts:36`). Corrupting the `data` field inside an envelope therefore does *not* produce a runtime error — it just triggers silent cleanup, and MSAL re-acquires the token via the refresh-token grant. To exercise the broadened catch in `getToken()`, simulate a condition that actually throws (network offline, or temporary instrumentation) rather than corrupting cache.

### Baseline

Start each scenario from a clean signed-in state: log out (or clear site data) → log in fresh → open DevTools. Between scenarios, the simplest reset is **Application → Storage → Clear site data** → log in again.

### Scenario 1: Happy path

1. Log in, load the documents view, confirm API calls succeed with no console errors.
2. Refresh the page — the shell should reload without another redirect (cached token still valid).

**Pass**: normal use works.

### Scenario 2: Missing access token, valid refresh token → MSAL-native silent refresh

Confirms the built-in renewal still fires and the new code does not interfere. (We can't edit `expiresOn` directly due to the encrypted envelope, so we simulate expiry by removing the access-token entry entirely — MSAL treats a cache miss the same way it treats an expired token.)

1. In Local Storage, delete the `…-accesstoken-…` entry.
2. Open `msal.token.keys.<clientId>` (plaintext JSON). Remove the deleted key from the `accessToken` array and save.
3. Leave the `…-refreshtoken-…` entry intact.
4. Trigger an API call (navigate to a view that fetches data).

**Pass**: no redirect, no console error. Network tab shows a silent POST to `https://login.microsoftonline.com/.../oauth2/v2.0/token`, and a fresh `…-accesstoken-…` entry reappears in Local Storage.

### Scenario 3: No refresh token → `InteractionRequiredAuthError` → `acquireTokenRedirect`

Tests the Interaction-required path — now routed through `acquireTokenRedirect` instead of `loginRedirect`.

1. In Local Storage, delete **both** the `…-accesstoken-…` and `…-refreshtoken-…` entries.
2. Open `msal.token.keys.<clientId>` and clear the `accessToken` and `refreshToken` arrays (or remove the specific keys), save.
3. Leave the account entry and `msal.account.keys` intact so MSAL still thinks a user is signed in.
4. Trigger an API call.

**Pass**: app redirects to Entra. If the user's SSO session at Entra is still valid it completes silently and returns authenticated; otherwise it prompts. **No cache wipe** — the account entry survives the round trip (this is the distinguishing difference from Scenario 4).

### Scenario 4: Non-Interaction error in `getToken()` → `clearCache()` + `acquireTokenRedirect`

Tests the broadened catch. Cache corruption no longer works as a trigger (MSAL v5 auto-removes invalid entries silently), so use temporary instrumentation — analogous to Scenario 6.

A naive `throw` in the try block creates an **infinite redirect loop**: every post-redirect `getToken()` call throws again, fires another redirect, returns, throws, etc. Gate the throw with a sessionStorage flag so it only fires once per tab session. `clearCache()` only touches MSAL's own `msal.*` entries, so the flag survives across the redirect.

1. In `app/client/core/auth.ts`, temporarily gate the first statement of the `getToken()` `try` block behind a one-shot sessionStorage flag:
   ```ts
   try {
     if (sessionStorage.getItem("__s4_fired") !== "1") {
       sessionStorage.setItem("__s4_fired", "1");
       throw new Error("simulated non-interaction failure");
     }
     const result = await msalInstance.acquireTokenSilent(request);
     return result.accessToken;
   } catch (e) { ... }
   ```
2. Rebuild (`cd app && bun scripts/build.ts`) and reload the app.
3. Trigger an API call (or wait for the first view's automatic fetch).

**Pass on first call**: all `msal.*` entries are cleared from Local Storage (`clearCache()` fired) and the app redirects to Entra. On return, a fresh cache is populated. On the next `getToken()` call the flag bypasses the throw and the app continues normally — no loop.

Cleanup:
- Revert the instrumentation in `auth.ts`.
- Delete `__s4_fired` from Session Storage (or run **Clear site data**).

### Scenario 5: `interaction_in_progress` short-circuit

Tests that the new code does not stack a second redirect when one is already pending.

1. In Session Storage, add key `msal.<clientId>.interaction.status` with value `"interaction_in_progress"` (use the same `clientId` visible in the existing session-storage keys).
2. Trigger a call that goes through `getToken` (navigate to a view, or call `Auth.getToken()` from the console).

**Pass**: the call resolves to `null` (or the API retry path surfaces `"Authentication required"`), **no new redirect fires**, no console error. Delete the key before moving on.

### Scenario 6: Stale `handleRedirectPromise()` state → `init()` catch

Hardest to simulate naturally (normally triggered by a days-old, never-returned-to redirect). Use temporary instrumentation:

1. Edit `auth.ts` inside the `try` block surrounding `handleRedirectPromise()` to `throw new Error("simulated")`.
2. Reload the app.

**Pass**: no unhandled rejection in the console, `msal.*` entries in Local Storage disappear, `app.ts` bootstrap sees no active account and redirects to Entra for a fresh login.

Revert the instrumentation before moving on.

## Validation Criteria

- [ ] `cd app && bun scripts/build.ts` completes without type or bundle errors
- [ ] `mise run dev` runs; fresh login completes and the shell loads (Scenario 1)
- [ ] Missing access token (simulated expiry) silently refreshes via refresh token (Scenario 2)
- [ ] Missing access + refresh tokens redirects cleanly without wiping the account entry (Scenario 3)
- [ ] Simulated non-Interaction throw triggers `clearCache()` + redirect (Scenario 4)
- [ ] `interaction_in_progress` flag prevents a stacked redirect (Scenario 5)
- [ ] Simulated `handleRedirectPromise()` throw recovers cleanly (Scenario 6)
- [ ] No redirect loop during normal session-within-lifetime use
