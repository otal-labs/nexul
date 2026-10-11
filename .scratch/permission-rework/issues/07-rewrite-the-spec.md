# 07 — Rewrite the spec and tickets

**Type:** task
**Status:** resolved
**Blocked by:** 06

## Question

Rewrite `.scratch/permission-overrides/spec.md`, its tickets, ADR 0148 (and the ADRs it supersedes: 0097, 0028
in part) and `CONTEXT.md` to the decisions on this map, with each locked look linked. When it merges, the
map is complete and this directory is deleted.

## Answer

Rewritten on 2026-10-11 in `.scratch/permission-overrides/`:

- `spec.md` covers the model end to end (roles only, several or none, opt-in projects with In this project, Kept out,
  Create projects and Manage this project's access, channels and plays workspace-wide, pills, the wipe and the
  one-time setup, the evaluation order and source sentences, computers, instance areas, performance numbers, MCP, the
  UI with each locked look linked, risks, and the owner's decisions), with a worked example that has no Everyone and a
  person holding two roles. Status ready-for-agent.
- `issues/01` to `issues/24` are ticket 06's slices, each marked master or `permission-switch`.
- ADR 0148 is rewritten and renamed; ADR 0097 is superseded, ADR 0028 and 0042 in part, and 0061, 0085, 0087, 0088,
  0098 and 0102 carry amendment lines. `CONTEXT.md` drops Restricted member, Every project and Project access, and
  adds In this project, Kept out and Access page.

Calls made in the rewrite, recorded in the spec's owner decisions:
- the keys `projects:open`, `projects:create` and `projects:access`;
- the paths that asked a project-area action with no project (create, the wizard's repository list, detaching GitHub,
  installation frames) move to `projects:create`;
- a DM whose other participants are all Owners takes membership alone, so a person with no role can message one;
- ticket 04's zero-roles copy is adjusted to round 4's decision that public channels and DMs still work;
- no channel carry-over in the wipe, as ticket 06's settled question says;
- the instance areas stay in the role editor's Workspace section, an open point for the owner.

This directory stays until slices 06, 07, 08, 12, 13, 14 and 15 merge, since tickets 03, 04, 05 and 08 hold the
looks they build.
