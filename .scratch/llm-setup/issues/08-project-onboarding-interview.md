# 08 — Step 0: the project onboarding interview

**Type:** grilling
**Status:** open
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
