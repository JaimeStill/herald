# 120 — Wire Token Injection into API Layer and Add Header User Menu

## Summary

Connected the MSAL auth service to the API transport layer so all `request()` and `stream()` calls carry bearer tokens when auth is enabled. Added 401 retry with silent token refresh. Added user display name and logout button to the app header via plain DOM hydration. During live validation with Azure Entra, discovered and fixed four issues: hardcoded scope name, OIDC audience/issuer mismatch, header layout with three flex children, and PDF iframe unauthorized.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Token injection location | Module-private `authHeaders()` in `api.ts` | Cross-cutting concern belongs at the transport layer, not in each service |
| 401 retry strategy | Single force-refresh, then login redirect | If force-refresh fails, the session is truly invalid |
| User menu rendering | Plain DOM hydration in `app.ts` | Header is server-rendered HTML; a Lit element for static name + button is over-engineering |
| OAuth scope configurability | `Scope` field on auth config, client composes full format | Deployments may use different scope names; server stores bare name, client composes `api://<client_id>/<scope>` |
| OIDC verifier audience | `api://<client_id>` with `SkipIssuerCheck` | Entra access tokens use app URI as audience and v1 issuer format, both mismatching OIDC discovery |
| PDF viewer with auth | Fetch blob via authenticated `download()`, use `blob:` URL | Iframes can't carry Authorization headers; blob URLs work transparently |
| Header layout | `margin-left: auto` on nav, `border-left` divider on user-menu | Clean `Brand ... [Nav] | [User Menu]` layout without wrapper elements |

## Files Modified

- `app/client/core/api.ts` — token injection, 401 retry for `request()` and `stream()`
- `app/client/core/auth.ts` — configurable scope from injected config
- `app/client/app.ts` — user menu hydration (name + logout button)
- `app/client/design/app/app.css` — header layout fix, user-menu styles with divider
- `app/client/ui/views/review-view.ts` — authenticated blob fetch for PDF viewer
- `app/app.go` — `Scope` field on `ClientAuthConfig`
- `cmd/server/modules.go` — pass scope through to client auth config
- `pkg/auth/config.go` — `Scope` field on `Config` and `Env`, merge/env/derive support
- `internal/config/config.go` — `HERALD_AUTH_SCOPE` env var mapping
- `pkg/middleware/auth.go` — verifier audience `api://` prefix, `SkipIssuerCheck`
- `config.auth.json` — auth overlay with scope
- `README.md` — Entra dev setup docs, auth dev server command
- `tests/config/auth_test.go` — 6 new scope tests (default, explicit, env, merge)

## Patterns Established

- **Configurable OAuth scope**: bare scope name in server config, client composes full `api://<client_id>/<scope>` URI. Default derived in `deriveDefaults()`.
- **Authenticated blob viewing**: when auth is enabled, fetch blobs through the API transport layer and use `blob:` URLs for iframes. Revoke on teardown.
- **OIDC access token validation**: Entra access tokens need `api://` audience prefix and `SkipIssuerCheck` due to v1/v2 issuer mismatch.

## Validation Results

- `go vet ./...` — clean
- `go test ./tests/...` — all pass (6 new scope tests)
- `bun run build` — succeeds
- Live validation with Azure Entra: login flow, token injection, 401 retry, user menu display, PDF viewing, logout — all working
