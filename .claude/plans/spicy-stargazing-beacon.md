# Plan: Draft Management Update Email

## Context

Jaime was given a directive this week to deliver the document classification service as soon as possible. He has a hard deadline of **March 19** (~4 weeks). He's already coordinated with customer-site partners and will be exclusively focused on Herald delivery with zero bandwidth for external engagements. The email explains this pivot to management.

## Approach

Write the email as a **progress-category** post following the style profile conventions.

### Voice Calibration
- **Tone**: Professional, confident, conversational warmth over technical substance
- **Register**: Formal enough for leadership, informal enough to feel like a person talking
- **Cadence**: Short grounding statement → longer explanatory → medium impact
- **Vocabulary**: Favor "foundational, proven, focused, established, accelerate" — avoid "utilize, synergy, paradigm, simply"
- **Audience**: Management — lean toward impact framing with enough technical grounding to convey credibility

### Structure (Progress Post Pattern)

1. **Opening** — Set context: the pivot to deliver document classification as the immediate priority, with a March 19 delivery target

2. **Section: R&D Foundation** — Brief arc establishing that this isn't starting from scratch:
   - **classify-docs** came first — the experiment that proved sequential page-by-page classification with GPT vision works (96.3% accuracy). This predated both go-agents-orchestration and agent-lab.
   - **agent-lab** — the full research platform that explored dynamic workflows, observability, profiles, and multi-stage pipelines. Valuable R&D, but built for experimentation, not focused delivery.
   - **TAU kernel** — the universal agent runtime effort. Still a relevant project worth pursuing *after* Herald ships.

3. **Section: Herald — Focused Delivery** — Frame Herald as a web service built exclusively for agentic document classification. Not a stripped-down agent-lab — a purpose-built service that eliminates the extraneous dynamic workflow concerns (observer infrastructure, multi-workflow registries, profile-based A/B testing, checkpoint persistence) in favor of a single, proven classification workflow targeting ~1M DoD PDF documents.

4. **Section: Current Status & Trajectory** — Phase 1 (Service Foundation) in progress. Completed: scaffolding, configuration system, lifecycle coordination, HTTP infrastructure, database toolkit, storage abstraction (6 merged PRs). Next: infrastructure assembly and API module integration, then Phase 2 (Classification Engine). 4 phases total targeting v0.4.0.

5. **Section: Delivery Commitment** — March 19 deadline. Already coordinated with customer-site partners. Exclusively focused — zero bandwidth for external engagements.

6. **Closing** — Forward-looking confidence statement

### Key Narrative Points
- The R&D journey validated *what works* — Herald applies those findings to a focused delivery
- classify-docs proved the classification approach before agent-lab or go-agents-orchestration existed
- Herald is purpose-built for document classification, not a general-purpose workflow platform
- TAU kernel remains a relevant future effort, sequenced after Herald delivery
- Concrete progress and clear 4-week trajectory

## Output
- File: `_project/email.md`
- Format: Markdown, no frontmatter (email draft, not blog post)
- Length: ~400-600 words — concise enough to scan, detailed enough to inform

## Verification
- Read against style profile: tone, cadence, vocabulary, structure
- Ensure no hedging language, no corporate buzzwords, no hyperbole
- Confirm narrative: context → R&D foundation → focused delivery → status → commitment → forward look
