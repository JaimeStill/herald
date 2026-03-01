# Objective: Web Client Foundation and Build System

**Issue**: #57
**Phase**: Phase 3 — Web Client (v0.3.0)

## Scope

Establish the complete web client infrastructure: Go-side template/asset serving (`pkg/web/`), Lit 3.x SPA with native Bun builds (no Vite), Air hot reload for development, client-side routing, design system, core API layer, and the web-development skill. This is the foundation that all subsequent web objectives depend on.

## Sub-Issues

| # | Sub-Issue | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | `pkg/web/` template and static file infrastructure | #62 | Open | — |
| 2 | Web client build system, design, and client application | #63 | Open | — |
| 3 | Go web app module, server integration, and dev experience | #64 | Open | #62, #63 |
| 4 | Web development skill | #65 | Open | #62, #63, #64 |

## Dependency Graph

```
#62 (pkg/web/)          #63 (Client App)
       \                      /
        v                    v
     #64 (Go Integration)
              |
              v
     #65 (Skill)
```

Sub-issues #62 and #63 can proceed in parallel. #64 requires both. #65 comes last.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CSS module handling | `{ type: 'css' }` + Bun plugin emitting `CSSStyleSheet` | Lit 3+ accepts `CSSStyleSheet` directly in `static styles` — no `unsafeCSS()` wrapper. Mirrors native CSS Module Scripts semantics. Plugin at `web/app/plugins/css-modules.ts` discriminates attributed imports from side-effect imports. |
| Global CSS extraction | Side-effect imports flow to Bun default pipeline | `import './design/index.css'` (no attribute) → Bun extracts to `dist/app.css`. No plugin interception. |
| Plugin location | `web/app/plugins/css-modules.ts` | Isolated from build scripts (`scripts/`). Clean separation of build orchestration from plugin logic. |
| Output filenames | Fixed names (`app.js`, `app.css`) | Required for stable `//go:embed` globs. Cache-busting handled at HTTP layer (ETags, cache headers). |
| Module mount path | `/app` | Coexists with `/api` module. SPA catch-all via fallback handler. |
| Dev workflow | Two terminals (Bun watch + Air) | Clean separation: Bun watches client source, Air watches Go + dist/. No single-process orchestration complexity. |
