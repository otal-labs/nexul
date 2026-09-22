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
  — two install locations cover all three skill-capable providers:
  `~/.claude/skills/` serves Claude Code **and** opencode (which scans it
  natively), `~/.agents/skills/` serves Codex; T3 Code adds no layer of its
  own, and Cursor/Grok drivers have no skill discovery at all.
- [What cursor/plugins pstack is](issues/02-pstack-contents.md) — an
  MIT-licensed engineering-discipline pack (38 skills, `poteto-mode` router)
  in Cursor's plugin marketplace; standard SKILL.md format so the content is
  portable, but no non-Cursor install path exists, and its `tdd` and `teach`
  skills collide by name with mattpocock/skills — a second set only with
  curation, never a flat merge.
- [Which skill sets "setup" installs](issues/03-which-skill-sets.md) —
  mattpocock/skills is the default the wizard installs (users may amend their
  copies); pstack is a README credit only; no Nexul-shipped skill for now —
  Nexul's own step-by-step wizard is the setup surface; "setup done" means
  the wizard completed on that computer; booleans never auto-flip and are
  enforcing (an unconfirmed provider cannot run agent work).

## Not yet specified

- Exact MCP tool names and shapes for reading and confirming setup state
  (one per use case, `<verb>_<object>`) — hangs on the setup-flow decision.
- Schema migration for the new state and how it hangs off `pairing.Computer`
  — hangs on the data-model decision.
- The wizard's pre-selection catalog: which parts of the default set are
  toggleable, and the later optional pstack complement (Lauren's principle
  skills, `architect`, `interrogate`, curated around the `tdd`/`teach`
  collisions) — hangs on the setup-flow decision.
- How "amend the flow per project" concretely manifests (plays the user
  edits + memories the interview seeds, or something more) — hangs on the
  step-0 and lifecycle tickets.
- Whether shaping/flow (the owner's steps 1–2) need anything Nexul-side
  beyond the installed skills and the existing plays — hangs on step 0 and
  the testing-step ticket.
- CONTEXT.md vocabulary and any ADR for the MCP-only write rule and the
  enforcement block — written once the decisions above resolve.

## Out of scope

<!-- work consciously ruled beyond the destination; nothing ruled out yet -->
