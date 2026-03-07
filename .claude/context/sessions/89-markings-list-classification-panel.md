# 89 - Markings List Element and Classification Panel Module

## Summary

Created the `hd-markings-list` pure element and `hd-classification-panel` stateful module for the document review view. The markings list renders security marking strings as badge tags. The classification panel loads a document's classification, displays it with sections for classification/confidence, markings, rationale, metadata, and validation status, and provides validate (agree with AI) and update (manual revision) actions. Also integrated the panel into the review view as a remediation, completing the review view composition originally scoped for #90.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Confidence badge colors | Custom `.confidence.high/medium/low` classes in panel CSS | Existing badge classes map to workflow stages, not confidence levels |
| Empty state navigation | `navigate("")` via button, not anchor tag | Consistent with client-side routing pattern |
| Review view composition in this task | Remediation R1 instead of deferring to #90 | Panel is unverifiable in isolation; composition is trivial (swap placeholder) |
| No component-level barrel imports | Removed `import "@ui/elements"` and `import "@ui/modules"` from components | `app.ts` registers all custom elements globally |

## Files Modified

- `app/client/ui/elements/markings-list.ts` (new)
- `app/client/ui/elements/markings-list.module.css` (new)
- `app/client/ui/modules/classification-panel.ts` (new)
- `app/client/ui/modules/classification-panel.module.css` (new)
- `app/client/ui/elements/index.ts` (barrel update)
- `app/client/ui/modules/index.ts` (barrel update)
- `app/client/ui/views/review-view.ts` (R1: replaced placeholder with classification panel)
- `app/client/ui/views/review-view.module.css` (R1: removed placeholder CSS rules)

## Patterns Established

- Confidence badge color mapping: `.confidence.high` (green), `.confidence.medium` (yellow), `.confidence.low` (red) — reusable if confidence badges appear elsewhere
- Module mode pattern: `@state() mode: "view" | "validate" | "update"` with dedicated render methods per mode — clean separation of view/form states

## Validation Results

- `bun run build` succeeds
- `go vet ./...` passes
- Visual verification: classification panel renders all sections correctly in review view
- Two-panel layout (PDF viewer + classification panel) displays as expected
