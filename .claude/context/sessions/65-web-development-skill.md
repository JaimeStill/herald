# 65 - Web Development Skill

## Summary

Created the Herald-specific web development Claude skill documenting Lit component patterns, service infrastructure, CSS architecture, API layer, router, build system, and Go integration conventions established by the Phase 3 foundation sub-issues (#62–#64). The skill uses progressive disclosure — a lean SKILL.md (155 lines) with 7 topic-specific reference files for detailed code examples.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Skill structure | SKILL.md + 7 reference files | Progressive disclosure keeps the always-loaded SKILL.md under 500 lines while providing full code examples on demand per topic |
| Reference organization | By concern (components, services, css, api, router, build, go-integration) | Maps directly to architectural boundaries — when working on CSS, load the CSS reference |
| What stays in SKILL.md | Architecture overview, naming conventions, anti-patterns, template pattern summaries | These are short, always-relevant, and needed regardless of which specific area you're working in |
| CLAUDE.md additions | Skill maintenance responsibility + frontend-design skill reference | Ensures the skill stays current as architecture evolves, and that view interface planning uses the frontend-design skill |

## Files Modified

- `.claude/skills/web-development/SKILL.md` — created (155 lines)
- `.claude/skills/web-development/references/components.md` — created (204 lines)
- `.claude/skills/web-development/references/services.md` — created (95 lines)
- `.claude/skills/web-development/references/css.md` — created (141 lines)
- `.claude/skills/web-development/references/api.md` — created (128 lines)
- `.claude/skills/web-development/references/router.md` — created (76 lines)
- `.claude/skills/web-development/references/build.md` — created (84 lines)
- `.claude/skills/web-development/references/go-integration.md` — created (114 lines)
- `.claude/CLAUDE.md` — updated (added Web Development Skill Maintenance + Frontend Design under AI Responsibilities)

## Patterns Established

- **Skill progressive disclosure**: SKILL.md as lean overview + `references/` directory with per-concern detail files. Each reference is self-contained and loaded on demand.
- **Skill maintenance as AI responsibility**: CLAUDE.md now mandates updating the web-development skill whenever the web architecture changes.
- **Frontend-design skill integration**: CLAUDE.md directs use of Anthropic's frontend-design skill for view interface planning tasks.
