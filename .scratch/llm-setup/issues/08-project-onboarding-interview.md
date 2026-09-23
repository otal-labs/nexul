# 08 — Step 0: the project onboarding interview

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Every project — new or existing — should start with an interview that writes
its first memories: the tech and language, the paradigm (ECS for a game, OOP
for web, composition where the language pushes that way), testing strategy
(e2e? separate e2e repo? coverage floor?), and the principles the code should
follow. The owner calls this the most-overlooked step: it is what lets
cheaper models write code the team can trust.

To settle:

- Where does it run — a seeded play on the project, a step in the harness
  wizard, an MCP workflow prompt, or a page-level "onboard this project"
  button? It needs the user answering questions, so it is a conversation
  with the agent, not a form.
- What it writes: project-scoped memories (the memories domain already has
  per-project scope, versioning, and agent writes) — one memory per fact, or
  one practices-style document?
- The question catalog: who owns it, is it fixed or does the agent extend it
  by reading the repo first (an existing repo can answer half the questions
  itself before asking the human the rest)?
- How re-runs work when a project's stack changes.

## Answer

Grilled with the owner 2026-09-23. The owner's name for it is **the
interview**.

- **The interview memory is the output**: one project-scoped memory holding
  the project's stack, paradigm, testing strategy, principles, and
  vocabulary, written as rules rather than a transcript, under a length cap
  the agent respects when writing it.
- **It is included in every agent turn** — every play and every @Agent
  mention — as the context, practices, and guardrails that steer any agent
  the right way. (Memories already carry an always-included flag; the
  interview memory is always on and cannot be switched off per play.)
- **Each project has an Interview page** that views and edits that memory —
  one source, no separate document. The **Interview play** runs on that page,
  so the agent's questions and the interviewee's replies land there. Plays
  target a ticket or a doc today; this adds the project's interview as a
  third target.
- **How the play asks**: one question at a time, each with a recommended
  answer. The interviewee can tell the agent to scan the existing codebase
  for answers, after which the agent grills them to verify what it found.
- **Workspace Interview template** in workspace settings, seeded with the
  starting categories: stack and versions; architecture (paradigm such as
  ECS, OOP, composition, or functional, and module boundaries); error
  handling and logging; testing (unit or integration, e2e in the same repo
  or a separate one, coverage floor, test-first); code style (early return,
  naming, comment density); dependency policy; security and secrets;
  performance budgets; CI gates; branching, PR, and commit rules; docs and
  decision records; UI (design system, mobile-first, accessibility); the
  project's own vocabulary. A new project's interview starts from the
  template; edits to one don't touch the other.
- **Offered and pushed, never enforced**: the project wizard's last step
  offers the interview; skipping it gets an "are you sure?" confirm saying
  what is lost; a banner stays on the project until the interview exists.
  Unlike machine setup, a project without an interview is not blocked.
- **Re-runs amend** the existing interview; memory versioning makes any bad
  amendment revertible.
