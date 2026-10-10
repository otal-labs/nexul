# 02 — Doc rules for a role or everyone

**Status:** ready-for-agent

**Blocked by:** 01

Read first: `practices/go.md` (sections 14, 16, 17), `practices/architecture.md` (sections 2, 3, 8),
`practices/mcp.md` (sections 4, 6, 7, 11), `practices/testing.md` (sections 3, 8, 10), ADRs 0010, 0042, 0088,
0135, 0137, 0140, 0148, the spec (Migration plan, Performance plan).

## What to build

- Migration (next free number): rebuild `permission_overwrites` with `target` (`person`, `role`,
  `everyone`), nullable `user_id` (cascading on account delete) and `role_id` (cascading on role delete), the
  check that exactly the right one is set, and the unique index over `(resource_type, resource_id, target,
  coalesce(user_id, role_id, ''))`. Copy every row as `person`. Each index names the query it serves. sqlc
  regenerated.
- The evaluator reads, for a doc, Everyone, then the person's role, then the person, in one statement with the
  person's own rows, memoised as today. A Restricted member's project-area actions take Deny only from
  Everyone and role rules.
- `DocsWith` adds the docs whose role or Everyone rows turn the project's answer, from one read bounded by the
  number of rules.
- Setting rules: `SetGrants` takes a target and a state (Allow, Fallback, Deny); the giver holds what an Allow
  adds or a lifted Deny hands back, judged on the doc (ADR 0088); adding a Deny is never limited. HTTP
  `PUT /api/permissions` gains `target`, `role_ids` and `state` beside its fields. `permission_overwrite_update`
  gains the same; `grant` keeps its meaning when `state` is absent.
- `ListUsers` (`internal/access/usecase.go:446-469`) also admits someone who holds `permissions:write` on a doc
  through a role or Everyone row.
- `access.grant.changed` gains `target` and `role_id` beside its fields; `make event-schemas`. Its audience
  (`grantFrame`, `server/cmd/live_audience.go:225-235`) reaches the role's members for a role row and the
  workspace's members for an Everyone row; the web and phone followers refetch the doc and the caller's
  permissions.
- Explain names role and Everyone sources.

## Acceptance criteria

- [ ] `TestMigration_EffectiveAccessUnchanged`: a fixture at the previous schema with every row kind, upgraded,
      matches the golden matrix from ticket 01.
- [ ] The matrix from a copy of the production database, taken at master and at the branch, has an empty diff
      (stated in the PR with the row counts compared).
- [ ] Table-driven `TestDecide_DocRuleOrder`: Everyone, role, person on one doc, every combination of
      states, and a Restricted member never gaining from a role or Everyone Allow.
- [ ] `TestGrants_NobodyAllowsWhatTheyLack` for a role and an Everyone row; a Deny is always accepted.
- [ ] `list_paging_test.go` gains role and Everyone rows on docs and stays green; a new
      `TestStatements_DocRulesCostTheSameAtAnyLength`.
- [ ] Deleting a role deletes its rows; deleting an account deletes its rows.
- [ ] `doc_list`, `ticket_list` and one live frame: statement count and median time on the heavy copy,
      production build, before and after, in the PR.
- [ ] `make event-schemas` and `make live-topics` are clean; the published contract only grows.
