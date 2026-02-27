# Objective #27 — Classifications Domain

**Phase:** [Phase 2 — Classification Engine](phase.md)
**Issue:** [#27](https://github.com/JaimeStill/herald/issues/27)
**Milestone:** v0.2.0 - Classification Engine

## Scope

Implement the full classifications vertical — persistence layer, business operations (classify/validate/update), HTTP endpoints, API wiring, and documentation. The classifications domain stores, queries, validates, and updates classification results produced by the workflow engine. It manages document status transitions through the classification lifecycle (pending → review → complete).

This objective merges the scope originally split across #27 (Classifications Domain) and #28 (Classification HTTP Endpoints) into a single cohesive objective.

**Out of scope**: Batch classification endpoint (client-orchestrated), SSE/streaming progress (Phase 3).

## Sub-Issues

| # | Sub-Issue | Status | Dependencies |
|---|-----------|--------|--------------|
| [#47](https://github.com/JaimeStill/herald/issues/47) | Classifications domain — types, system, and repository | Open | — |
| [#48](https://github.com/JaimeStill/herald/issues/48) | Classifications handler, API wiring, and API Cartographer docs | Open | #47 |
| [#51](https://github.com/JaimeStill/herald/issues/51) | Parallelize classify and enhance workflow nodes | Open | — |

## Architecture Decisions

1. **`Classify` lives on the System interface**: The `classifications.repo` holds `*workflow.Runtime` as a constructor dependency. `Classify(ctx, documentID)` internally calls `workflow.Execute()`, stores the result, and transitions document status. The handler calls `sys.Classify(ctx, documentID)` — thin and consistent with documents/prompts patterns.

2. **Validate and Update are alternatives, not sequential**: Both transition document status `review → complete`. Validate = human agrees with AI classification. Update = human manually overwrites `classification` and `rationale`. Re-classification (another `Classify` call) resets validation fields via upsert semantics.

3. **No additional migration**: Update overwrites the existing `classification` and `rationale` columns directly — no separate adjustment columns needed.

4. **Layered runtime convention**: `classifications.New` receives raw infrastructure deps and peer systems, constructs `workflow.Runtime` internally. The API composition root (`api.NewDomain`) does not import workflow — it passes `AgentConfig`, `storage.System`, `documents.System`, and `prompts.System` through. This keeps workflow composition encapsulated within the classifications domain.

5. **Upsert semantics on Classify**: `ON CONFLICT (document_id) DO UPDATE` overwrites all inference fields and resets `validated_by`/`validated_at` to NULL.
