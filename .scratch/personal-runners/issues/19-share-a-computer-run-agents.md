# 19 — Share a computer: Run agents

**Status:** needs-triage

**Blocked by:** 18, and the owner's answer to the spec's open question 6

Read first: the spec (Later: sharing a computer), ADRs 0063, 0102, 0111.

## What to decide, then build

A grantee's turn on the owner's computer reaches Nexul over MCP with the owner's token, so it would act as
the owner. Before this is built: research whether T3 Code can give one thread its own MCP credential (a
per-turn token Nexul mints for the starter), and record the answer as an ADR. Only then honour
`run_agents` in target resolution, the run dialog and project links (amending ADR 0102).

## Acceptance criteria

- [ ] The research and the owner's decision are recorded first.
