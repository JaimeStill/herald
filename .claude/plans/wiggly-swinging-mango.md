# Plan: Herald Status Update Email

## Context

Jaime's last status update (evening of Feb 20) only announced Herald's existence, the March 19 delivery target, and the initial project management setup. Since then, one week of intense development has produced the entire service foundation (Phase 1, tagged v0.1.0) and nearly the entire classification engine (Phase 2). End-to-end classification has been verified against a live Azure AI Foundry deployment. One task remains before Phase 2 is complete — parallelizing the page analysis nodes — targeted for completion today (Feb 27).

The audience is leadership/stakeholders. The email should emphasize milestones, what they mean in practical terms, and trajectory toward the March 19 deadline.

## Approach

Draft a progress-category email to `_project/email.md` following the dev-blog voice from `~/tau/jaime/.claude/context/style-profile.md`:

- Professional, confident, conversational warmth over technical substance
- **Grounded and factual** — let the work and progress metrics speak for themselves
- **No overselling or sensationalism** — state what was done, what it means, and what's next
- Avoid vanity metrics (lines of code) or framing that reads as self-congratulatory
- Declarative assertions, not hedging — but also not boasting
- Em dashes for mid-sentence elaboration
- Short grounding opener → explanatory body → forward-looking close
- Links to GitHub resources for those who want to dig deeper

## Email Structure

### Opening
Ground the reader: one week since breaking ground, here's where things stand. Matter-of-fact framing — no "incredible week" or similar.

### Section 1: Service Foundation Complete (Phase 1 → v0.1.0)
What was built in the first half of the week:
- Document management — upload PDFs, store in Azure Blob Storage, track in PostgreSQL
- Full HTTP API with 5 document endpoints + 3 storage endpoints
- Database migration system, configuration infrastructure, lifecycle coordination
- **What it means**: Documents flow in, get stored, and are queryable. The plumbing is in place.

### Section 2: Classification Engine (Phase 2)
The headline achievement — the AI classification pipeline works end-to-end:
- 4-node workflow: render pages → analyze each page with GPT vision → conditionally enhance → synthesize document-level classification
- Prompt management system so classification behavior can be tuned without code changes
- Classifications domain — persist results, human validation/adjustment workflow
- **What it means**: Upload a PDF, hit the classify endpoint, and get back a structured classification with per-page analysis, confidence level, and rationale. Tested against live Azure AI Foundry.

### Section 3: What's Next
- One task remains: parallelizing page analysis (currently sequential ~13s/page, parallelization will dramatically reduce this). Targeting completion today.
- Phase 2 complete → transition to Phase 3 (web client) next week
- March 19 delivery target remains on track — Phase 2 wrapping in week 1 of 4 leaves strong runway for the web UI and deployment configuration

### Closing
Brief forward-looking close. Include resource links (repo, project board). No self-congratulation — just state the trajectory and next steps.

## Key Facts to Reference (objectively verifiable)

- **v0.1.0 tagged Feb 24** — Phase 1 complete
- **22 merged PRs** since breaking ground (implementation + planning)
- **4 domain systems wired**: documents, classifications, prompts, workflow
- **End-to-end verified** against live Azure AI Foundry GPT deployment
- **March 19 deadline**: Phase 2 targeting completion today (end of week 1)
- **Phase 3** (web client) starts next week — Lit 3.x SPA for document management and classification review
- Do NOT cite lines-of-code or similar vanity metrics

## Style Reference Files

- `~/tau/jaime/.claude/context/style-profile.md` — voice and tone conventions
- Previous email (provided by user) — direct sample of actual voice

## Output

Single file: `_project/email.md` containing the draft email body (markdown formatted, no frontmatter).

## Verification

- Read the draft aloud mentally for voice consistency with the Feb 20 email
- Confirm all facts match git history and project documentation
- Ensure leadership-appropriate framing (milestones and impact, not implementation details)
- Check that the email advances the narrative from "just broke ground" without re-covering the project announcement
- **Tone check**: Does any sentence read as boasting or overselling? If so, rewrite to be factual and grounded
