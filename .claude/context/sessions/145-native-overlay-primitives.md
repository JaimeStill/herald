# 145 - Adopt native overlay primitives for tooltips, modals, and toasts

## Summary

Migrated Herald's three overlay classes — confirm dialog, toast stack, and (new) tooltip — to native browser top-layer primitives. `hd-confirm-dialog` now uses `<dialog>.showModal()` for focus trap, Escape handling, and `::backdrop` styling. `hd-toast-container` uses `popover="manual"` to stay above modal dialogs via top-layer stacking. New `hd-tooltip` uses `popover="hint"` with CSS Anchor Positioning to reveal truncated filenames/prompt names on hover or keyboard focus. Scope creep discovered mid-session: removed the universal `* { scrollbar-gutter: stable }` rule introduced in #133 (it leaked phantom gutter onto every `overflow: hidden` container), dropping the `base` cascade layer entirely.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Confirm dialog primitive | `<dialog>` + `.showModal()` | Focus trap, Escape → `cancel`, focus-return on close, and `::backdrop` all native. No `z-index` required |
| Prevent layout shift on dialog mount | `:host { display: contents }` | Default `display: inline` custom-element host contributed enough baseline to reflow the parent grid. `display: contents` removes the host's own box from the layout tree entirely |
| Tooltip popover type | `popover="hint"` (not `"auto"`) | Hints do not close open auto popovers, so hovering a tooltip inside a menu leaves the menu open. Hints do close other hints, so only one is ever visible |
| Tooltip truncation gating | None — tooltip always shows on hover | The tooltip is a dumb primitive; composing elements own any conditional logic. Keeps the element's API minimal and its behavior predictable |
| Confirm button color | `confirmKind: "danger" \| "primary" \| "neutral"` enum, default `"danger"` | Element is named generically (`hd-confirm-dialog`), and the hardcoded `btn-red` baked in destructive semantics that only coincidentally matched current callers. Enum maps caller intent to a class without leaking raw CSS names |
| Toast popover type | `popover="manual"` | No light-dismiss — outside clicks should not kill every toast. Manual show/hide lifecycle fits the service-driven toast bus |
| Toast positioning | Fixed positioning on inner `.stack`, not `:host` | UA `[popover]` stylesheet (`inset: 0; margin: auto; ...`) fights `:host` rules in the cascade and produces unreliable placement. Plain-div `.stack` has no UA popover rules competing with it |
| `scrollbar-gutter: stable` scope | Only on `.scroll-y` / `.scroll-x` utilities, never universal | `scrollbar-gutter: stable` reserves gutter space on any element whose `overflow` is `auto`, `scroll`, **or `hidden`** — not a no-op on non-scroll elements as #133 assumed. Universal application leaked a phantom ~15px gutter onto every `overflow: hidden` container |

## Files Modified

**New:**
- `app/client/ui/elements/tooltip.ts`
- `app/client/ui/elements/tooltip.module.css`

**Modified:**
- `app/client/ui/elements/confirm-dialog.ts` — `<dialog>.showModal()` refactor, new `confirmKind` property, `ConfirmKind` type export
- `app/client/ui/elements/confirm-dialog.module.css` — `:host { display: contents }`, `::backdrop` styling, `z-index` removed
- `app/client/ui/elements/toast.ts` — `popover="manual"` lifecycle in connected/disconnected callbacks
- `app/client/ui/elements/toast.module.css` — `:host` UA chrome reset, positioning returned to `.stack`, `z-index: 200` removed
- `app/client/ui/elements/document-card.ts` — `<hd-tooltip>` wraps filename span
- `app/client/ui/elements/prompt-card.ts` — `<hd-tooltip>` wraps name span
- `app/client/ui/elements/index.ts` — export `Tooltip`, re-export `ConfirmKind` type

**Deleted:**
- `app/client/design/core/base.css` — removed along with its import from `app/client/design/index.css` and the `base` entry from the `@layer` declaration

**Documentation:**
- `.claude/CLAUDE.md` — cascade-layers bullet updated to reflect four-layer stack and scoped `scrollbar-gutter`
- `.claude/skills/web-development/SKILL.md` — overlay-convention patterns gained `:host { display: contents }` note (modal) and inner-element-owns-layout note (popover stack)
- `.claude/skills/web-development/references/components.md` — new "Overlay Elements" section with the three patterns
- `.claude/skills/web-development/references/css.md` — cascade-layers section rewritten; `base` layer entry replaced with scroll-utility-only scrollbar-gutter scoping
- `CHANGELOG.md` — `v0.5.0-dev.132.145` entry

## Patterns Established

- **`:host { display: contents }` for overlay elements whose custom-element host should not participate in parent layout.** Applies to any overlay where the top-layer content (dialog, popover) should not pass a sliver of layout impact up through the mount point.
- **Inner element owns layout for popover hosts.** When `popover="manual"` / `"auto"` / `"hint"` is set on the host, UA `[popover]` defaults compete with author `:host` rules. Put positioning and sizing on an inner element that sits outside the popover cascade.
- **Semantic-variant enums for styled buttons.** When a component's presentation depends on caller intent, expose the intent as an enum (`"danger" | "primary" | "neutral"`) and map to classes internally. Keeps the styling system encapsulated while letting callers be explicit.
- **Overlay elements are dumb; gating is the caller's job.** The tooltip always shows on hover. Truncation detection, disabled flags, or conditional mounting belong in composing elements, not in the primitive itself.

## Validation Results

- `mise run vet` — clean
- `mise run test` — all Go tests pass (no client-side test infra exists)
- Manual browser validation — developer confirmed toasts bottom-centered, confirm dialog does not shift underlying view, tooltip shows/hides on hover and focus, all three existing dialog callers (single document delete, bulk document delete, prompt delete) work, no phantom scrollbar gutter on the app shell's right edge
