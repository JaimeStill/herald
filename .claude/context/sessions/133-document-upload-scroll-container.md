# 133 - Scroll container for document-upload module

## Summary

Fixed the unbounded growth of the `hd-document-upload` queue (it overflowed the viewport when many files were queued) by adding scroll semantics to the queue and pinning the queue header. Scope expanded mid-session to extract a reusable scroll primitive into the design system — a universal `scrollbar-gutter: stable` rule plus `.scroll-y` / `.scroll-x` utilities — and to migrate all seven of the app's scroll containers to the utility. Skill and project docs updated so future scroll containers default to the utility.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Queue scroll structure | Wrap mapped entries in a new `.queue-list` child of `.queue` | Keeps the `.queue-header` (title, count, Clear/Upload buttons) pinned while only the entries scroll — more useful than scrolling the whole queue. |
| `scrollbar-gutter` placement | New `base` layer between `reset` and `theme` with `* { scrollbar-gutter: stable; }` | `scrollbar-gutter` is a no-op on non-scrolling elements, so the universal selector is safe. Own layer keeps cross-cutting primitives separate from reset-layer normalizations. |
| Shadow-DOM reach | New `design/styles/scroll.module.css` imported per component | Light-DOM rules don't pierce shadow DOM. A shared CSS module reuses the existing `@styles/*` pattern (no base-class refactor, no build-time magic). |
| Utility class vs attribute | `.scroll-y` / `.scroll-x` classes | Matches existing `@styles/*` convention (`.btn`, `.input`, `.badge`, `.label`, `.card`). |
| Visual scrollbar spacing | Utility bundles `padding-inline-end: var(--space-2)` (or `padding-block-end` for `.scroll-x`) | CSS can't detect "is-scrolling" via a selector, so padding is baked into the utility. Non-scrolling elements don't use the utility, so there's no accidental padding elsewhere. |
| Scope of design-system work | Folded into #133 as Remediation 1 | The new scroll container introduced here benefits immediately, and the primitive lands in a natural place rather than waiting on a follow-up issue. |
| `@design/index.css` TS error | Folded in as Remediation 2 (three-line fix in `css.d.ts`) | Pre-existed on `main`, but the fix is trivial and we were touching the design-system entry point anyway. Keeps `tsc --noEmit` clean going forward. |

## Files Modified

**New files:**
- `app/client/design/core/base.css`
- `app/client/design/styles/scroll.module.css`

**Design system:**
- `app/client/design/index.css` — register `base` layer and import
- `app/client/css.d.ts` — add `declare module "*.css"` for side-effect imports

**Component migrations (remove local `overflow-y: auto`, import `scrollStyles`, apply `.scroll-y`):**
- `app/client/ui/modules/document-upload.{module.css,ts}` (new `.queue-list` scroll container + host/drop-zone flex sizing)
- `app/client/ui/modules/document-grid.{module.css,ts}`
- `app/client/ui/modules/prompt-list.{module.css,ts}`
- `app/client/ui/modules/prompt-form.{module.css,ts}` (both `.form-body` and inner `.defaults-content`)
- `app/client/ui/modules/classification-panel.{module.css,ts}` (all three `.panel-body` template usages)
- `app/client/ui/views/review-view.{module.css,ts}`

**Documentation context:**
- `.claude/CLAUDE.md` — new "Web Client Conventions" section
- `.claude/skills/web-development/SKILL.md` — CSS one-liner, Prefer/Avoid lists
- `.claude/skills/web-development/references/css.md` — five-layer stack, new `base` layer subsection, `scroll.module.css` in shared-styles table, rewritten scroll-container view pattern

## Patterns Established

- **Scroll containers use the `.scroll-y` / `.scroll-x` utility.** Never declare `overflow-y: auto` directly in a component's CSS. Layout (`flex: 1; min-height: 0;`, `max-height: ...`) lives in the component CSS; scroll behavior (`overflow`, `scrollbar-gutter`, axis padding) lives in the utility. The utility is imported alongside the existing `@styles/*` pattern.
- **The `base` layer** is the home for cross-cutting primitives (universal `scrollbar-gutter`, future similar rules) that aren't resets but apply broadly. Sits between `reset` and `theme` in the cascade.
- **Scroll header pinning** — when a scroll container has a fixed header (title, actions, counts), wrap entries in a dedicated scroll child rather than making the whole container scroll. See `document-upload`'s `.queue` / `.queue-list` split for the reference pattern.

## Validation Results

- `go vet ./...` — clean
- `mise run test` — 20/20 Go packages pass
- `bun run build` — Bun CSS-modules plugin picks up `scroll.module.css` cleanly, `dist/app.js` + `dist/app.css` build
- `bunx tsc --noEmit` — clean after R2 fix (was erroring on `@design/index.css` side-effect import, pre-existing on main)
- Manual UI verification — queue scrolls internally with ≥ 20 files, header pinned, drop zone fixed above, all six other scroll containers (document grid, prompt list, classification panel, prompt form body + defaults, review-view classification panel) verified visually with scrollbar breathing room
