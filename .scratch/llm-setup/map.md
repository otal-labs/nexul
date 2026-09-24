# Wayfinder map: LLM setup for paired harnesses

**Label:** wayfinder:map

## Destination

A locked design, ready to slice into implementation tickets, in two halves
(the owner folded the lifecycle vision into this one map on 2026-09-22 —
no users yet, move quick):

1. **Machine setup.** The MCP server owns setup state for every paired
   computer: an overall "setup confirmed" boolean plus one boolean per
   provider, false by default, writable only through MCP tools, and
   **enforcing** — a provider on a computer cannot run agent work until its
   boolean is true. Nexul's own step-by-step wizard walks a new user through
   setting the harness up, installing the default skill set
   (mattpocock/skills) with pre-selections. The README credits pstack.
2. **The guided lifecycle.** Nexul guides every project through the loop so
   cheaper models write trustworthy code: a step-0 onboarding interview that
   writes a project's first memories (tech, paradigm, testing strategy,
   principles), shaping and flow via the skills, bug tickets that always link
   the ticket they came from, a testing step that serves testers and QA (not
   just devs), and "done" that indexes the why. Memories accumulate and the
   flow gets amendable per project.

## Notes

- Domain: `internal/pairing` (paired computers, providers, defaults) and
  `internal/mcp`. A "provider" is the driver inside a paired T3 Code instance
  (claude, opencode, codex…) — `t3client.Providers()` lists them. "Per
  harness" in the original brief means per provider on one paired computer.
- Standing requirement from the owner, treat as fixed: the booleans default
  to false and can be flipped **only via MCP** — never from the web UI —
  because the point is that an LLM assessed the machine and confirmed the
  setup actually works.
- Also fixed by the owner: nothing auto-flips a boolean to false — not a
  skills release, not re-pairing, not a nightly update. State is per
  computer; a new computer or a newly appearing provider starts at false.
- No "v1"/version framing anywhere — Nexul has no versioned feature tiers;
  say "for now" or "first cut".
- HITL tickets run `/grilling` and `/domain-modeling`. Questions to the owner
  in plain language, never spec codes.
- README ticket: the inspiration section may name open-source repos
  (mattpocock/skills and pingdotgg/t3code already are); it may not read as a
  product comparison.
- Research findings land in `research/` beside this map.

## Decisions so far

<!-- one line per resolved ticket: gist plus link -->

- [How each T3 Code provider discovers skills](issues/01-skill-discovery-per-provider.md)
  — every T3 Code driver discovers skills natively (a first pass wrongly
  exempted Cursor and Grok; corrected 2026-09-22), and two install locations
  cover five of the six: `~/.claude/skills/` serves Claude, opencode, and
  Cursor; `~/.agents/skills/` serves Codex, Cursor, and Grok; only
  Antigravity has a separate mechanism. T3 Code adds no layer of its own.
- [What cursor/plugins pstack is](issues/02-pstack-contents.md) — an
  MIT-licensed engineering-discipline pack (38 skills, `poteto-mode` router)
  in Cursor's plugin marketplace; standard SKILL.md format so the content is
  portable, but no non-Cursor install path exists, and its `tdd` and `teach`
  skills collide by name with mattpocock/skills — a second set only with
  curation, never a flat merge.
- [Setup-state data model](issues/04-setup-state-data-model.md) — the
  overall boolean is stored (wizard-set via MCP, UI read-only), provider
  state is keyed by driver kind not instance id, every driver gets a boolean
  (no exempt category — all six discover skills), the confirm call is honor
  system taking the computer's id, the un-confirm path exists, and the
  schema uses nullable confirmed-at timestamps written only by MCP use-cases.
- [The setup flow itself](issues/05-setup-flow.md) — the wizard fires a
  built-in setup turn through the paired harness, the one run exempt from
  the block; coarse pre-selection (whole mattpocock set, both install
  locations); confirmation needs files present plus the harness re-reporting
  each driver's discovered skills, recorded on confirm; re-enterable and
  idempotent; nexul.io points at the wizard.
- [Surfaces and consequences of setup state](issues/06-surfaces-and-consequences.md)
  — a run needs the computer's and the provider's boolean both true; the
  block sits at target resolution (one guard for @Agent and plays); the
  refusal is a new not-configured reply linking to the wizard, plays fail on
  press rather than greying out; the pairing row shows badge, per-provider
  lines, and a Set up button opening the wizard dialog; pickers tag
  unconfirmed providers "needs setup"; owner-only, with events and live push.
- [README credits pstack](issues/07-readme-pstack-mention.md) — added as a
  closing paragraph in "Where the idea comes from": a set worth keeping beside
  mattpocock/skills, not one of the two repositories that shaped Nexul.
- [Step 0: the project onboarding interview](issues/08-project-onboarding-interview.md)
  — the Interview play writes one project interview memory (rules, length
  capped) included in every agent turn; each project's Interview page views
  and edits it and hosts the play; the agent may scan the codebase then
  grill to verify; a workspace Interview template seeds the questions;
  offered in the project wizard with an "are you sure?" on skip and a banner
  until done, never blocking; re-runs amend.
- [Bugs and the done boundary](issues/09-bugs-and-the-done-boundary.md) —
  bugs require a "found in" link and carry a template; done is never
  reopened (before done the card moves back, after done a linked bug is
  filed); the fixing agent gets one hop of origin context; humans and agents
  file bugs; every ticket gets a reporter (agents show as "Nexul · for
  <person>"); "blocked by" links with a board icon and a play confirm, never
  a gate. Signals over gates, room for mistakes. Every ticket type carries a
  body template (bug: steps, expected, actual, screenshot; feature and task
  seeded too), edited on the type in project settings.
- [The testing step](issues/10-testing-step.md) — a Test this panel (live
  URL, acceptance criteria, Pass/Fail) in testing columns; Fail opens the bug
  template, posts to the ticket thread, and moves the card back; tickets get
  Developer (renamed Assignee), Tester, and Reporter; bugs leave the create
  dialog (Report a bug, Fail, or "origin unknown"); a Test with AI play;
  an optional never-deployed tests repository; and a project-wizard deploy
  branches step with a per-row network, optional overrides, and URL-safe
  hostnames.
- [Which skill sets "setup" installs](issues/03-which-skill-sets.md) —
  mattpocock/skills is the default the wizard installs (users may amend their
  copies); pstack is a README credit only; no Nexul-shipped skill for now —
  Nexul's own step-by-step wizard is the setup surface; "setup done" means
  the wizard completed on that computer; booleans never auto-flip and are
  enforcing (an unconfirmed provider cannot run agent work).

## Not yet specified

- Whether "amend the flow per project" needs anything beyond what is now
  settled (editable plays, the workspace Interview template, interview
  re-runs, the decisions log) — revisit once the remaining lifecycle
  tickets resolve.
- Whether shaping/flow (the owner's steps 1–2) need anything Nexul-side
  beyond the installed skills and the existing plays — hangs on step 0 and
  the testing-step ticket.
- CONTEXT.md vocabulary and any ADR for the MCP-only write rule and the
  enforcement block — written once the decisions above resolve.

## Out of scope

<!-- work consciously ruled beyond the destination -->

- Per-skill pre-selection and the optional pstack complement (Lauren's
  principle skills, `architect`, `interrogate`, curated around the
  `tdd`/`teach` collisions) — the first cut installs the whole default set
  ([The setup flow itself](issues/05-setup-flow.md)); a later effort adds
  choice.
