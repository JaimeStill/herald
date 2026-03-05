# Objective: Prompt Management View

**Issue**: #60
**Phase**: Phase 3 — Web Client (v0.3.0)

## Scope

Build the prompt management view — a CRUD interface for named prompt overrides with stage filtering and active toggle. Accessible at `/app/prompts`. The prompt domain layer (types + stateless service) and backend API are already complete. This objective is purely UI work.

## Sub-Issues

| # | Sub-Issue | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | Prompt card pure element and search request type | #82 | Open | — |
| 2 | Prompt list module with stage filtering and activation | #83 | Open | #82 |
| 3 | Prompt form module and view integration | #84 | Open | #82, #83 |

## Dependency Graph

```
#82 (Card + Search Type)
      |
      v
#83 (List Module)
      |
      v
#84 (Form + View Assembly)
```

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| State management | `@state()` + direct service calls | Follows established codebase pattern. Modules own their data, call services directly. No Signal.State/context needed. |
| Layout | Vertical card list, not grid | Prompts are text-heavy (name, description, instructions). Vertical list is more scannable. |
| Active toggle | On card, not in form | Activate/deactivate is atomic (auto-deactivates previous active for stage). Quick-toggle is more intuitive. |
| Form values | FormData extraction on submit | Per web-development skill convention. Not controlled inputs. |
| Stage in edit | Dropdown disabled | Changing stage is semantically unusual and could cause unexpected deactivation. |
| Design approach | Utilitarian consistency | Stay within existing Herald design tokens and system fonts. Clean, functional, production-focused. |
