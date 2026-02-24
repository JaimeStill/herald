# Objective #25 — Prompts Domain

**Phase:** [Phase 2 — Classification Engine](phase.md)
**Issue:** [#25](https://github.com/JaimeStill/herald/issues/25)
**Milestone:** v0.2.0 - Classification Engine

## Scope

Implement the full CRUD domain for named prompt instruction overrides, following the documents domain pattern. Each prompt override targets a specific workflow stage (init/classify/enhance) and provides tunable instructions for that stage.

**Out of scope**: Hard-coded default prompts (those live in `workflow/prompts.go`), prompt loading logic for workflow execution, output format definitions.

## Sub-Issues

| # | Sub-Issue | Status | Dependencies |
|---|-----------|--------|--------------|
| [#34](https://github.com/JaimeStill/herald/issues/34) | Prompts domain implementation | Open | — |

## Architecture Decisions

1. **Two-layer prompt composition**: The final system prompt sent to the LLM is composed of two layers — tunable instructions (from DB overrides or hard-coded defaults) and hard-coded output format (defined in `workflow/prompts.go`). The prompts domain manages only the instruction layer. This separation allows tuning classification reasoning without risking broken output formats that the workflow depends on for JSON parsing.

2. **Column naming**: The `prompts` table uses `instructions` (not `system_prompt`) to make the two-layer architecture self-documenting at the schema level.

3. **Stage validation**: The `stage` column has a DB CHECK constraint (`init`, `classify`, `enhance`). The domain also validates stage values at the application layer before hitting the DB, providing clearer error messages.

4. **Single sub-issue**: The full CRUD domain is a single sub-issue because it follows the well-established documents domain pattern with no external service integration. Splitting would create unnecessary overhead.
