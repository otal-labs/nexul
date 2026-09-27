# 05 — Docs, ADRs, glossary

**Status:** ready-for-agent
**Type:** task
**Blocked by:** 01, 02, 03, 04

## Scope

- ADRs: native services supersede ADR 0069's compose stack and ADR 0072, and
  the remainder of 0015; OpenObserve native behind `/openobserve/` amends 0008;
  per-host credentials with one-time enrollment codes; automations placement
  and host-scoped tokens amend 0046; 0052 loses its container exception.
- `CONTEXT.md`: Runner, Automations host, Instance upgrade, and new terms for
  enrollment code and host credential; remove terms for things that no longer
  exist.
- Website docs: install, upgrade, runners, logs, automations, local
  development, CI and releases, index and install switcher copy, README, bug
  template, `AGENTS.md` service-binary quick reference.
- Delete `.scratch/per-runner-credentials/` (folded into this effort) and
  update `docs/agents/issue-tracker.md`'s list.
