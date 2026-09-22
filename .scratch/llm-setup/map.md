# Wayfinder map: LLM setup for paired harnesses

**Label:** wayfinder:map

## Destination

A locked design, ready to slice into implementation tickets: the MCP server
owns setup state for every paired computer (an overall "setup confirmed"
boolean plus one "skills installed" boolean per provider inside the harness),
false by default and writable only through MCP tools, so a new user's agent
performs the skill installation, verifies it, and confirms it — and plays run
as well on their machine as on the owner's. The README's inspiration section
also credits the second skill set.

## Notes

- Domain: `internal/pairing` (paired computers, providers, defaults) and
  `internal/mcp`. A "provider" is the driver inside a paired T3 Code instance
  (claude, opencode, codex…) — `t3client.Providers()` lists them. "Per
  harness" in the original brief means per provider on one paired computer.
- Standing requirement from the owner, treat as fixed: the booleans default
  to false and can be flipped **only via MCP** — never from the web UI —
  because the point is that an LLM assessed the machine and confirmed the
  setup actually works.
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

## Not yet specified

- Exact MCP tool names and shapes for reading and confirming setup state
  (one per use case, `<verb>_<object>`), and whether a workflow prompt guides
  the installing agent — hangs on the setup-flow decision.
- Schema migration for the new state and how it hangs off `pairing.Computer`
  — hangs on the data-model decision.
- Whether unconfirmed setup feeds the existing `NotConfiguredReason` path
  that @Agent replies with, or only the plays surface — hangs on the
  surfaces decision.
- Whether the install instructions themselves ship as a seeded play, a
  workflow prompt, or docs on nexul.io — hangs on the setup-flow decision.
- CONTEXT.md vocabulary and any ADR for the MCP-only write rule — written
  once the decisions above resolve.

## Out of scope

<!-- work consciously ruled beyond the destination; nothing ruled out yet -->
