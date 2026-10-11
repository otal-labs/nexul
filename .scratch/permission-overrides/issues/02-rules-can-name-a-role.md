# 02 — Rules can name a role

**Status:** done

**Blocked by:** 01

**Branch:** master, a normal PR, before the switch. Built and guarded; no writer creates a role rule yet.

Read first: `practices/go.md` (sections 14, 16, 17), `practices/architecture.md` (sections 2, 3, 8),
`practices/testing.md` (sections 3, 8, 10), `practices/react-guide.md` (Live events), `practices/native.md`
(section 4), ADRs 0010, 0042, 0044, 0135, 0137, 0140, 0148, the spec (Targets, Evaluation order, Performance plan),
the research (sections 1.1, 1.3, 3 and 5).

## What to build

- A forward-only migration (next free number) rebuilds `permission_overwrites` with `target` (`person`, `role`; no
  Everyone), a nullable `user_id` cascading on account delete and a nullable `role_id` cascading on role delete, a
  check that exactly one is set, and a unique index over `(resource_type, resource_id, target, coalesce(user_id,
  role_id))`. Every row is copied as `person`. Each index names the query it serves. sqlc regenerated.
- The evaluator reads a person's rows and their held roles' rows in one memoised statement. On each entity the role
  rules merge with Allow winning, then the person rule beats them, and the nearer entity beats the farther one.
- `DocsWith` stays bounded by the number of rules, and `ListUsers` (`internal/access/usecase.go:447-469`) counts role
  rows that allow `permissions:write`.
- `access.grant.changed` gains `target` and `role_id` beside `user_id`, never in its place; `make event-schemas`. A
  role row's audience is the role's holders (`grantFrame`, `server/cmd/live_audience.go:227-237`), and the web and
  phone followers refetch `/me` on a frame that names no person, the way `role.updated` does. `make live-topics`.
- No HTTP route or MCP field writes a role row yet.

## Acceptance criteria

- [x] An empty `access.Matrix` diff, master against the branch, with row counts, in the PR. The owner chose a prod-shaped fixture over a production copy (2026-10-11): a schema-0096 database holding every rule kind.
- [x] Table-driven `TestDecide_RoleRules`: two roles on one entity in every combination of states, a person rule over
      them, and a nearer entity over a farther one, each with its source.
- [x] Deleting a role deletes its rows; deleting an account deletes its rows.
- [x] `list_paging_test.go` gains role rows on docs and stays green; new
      `TestStatements_RoleRulesCostTheSameAtAnyLength`, rules on every row against none.
- [x] `liveRequests.test.tsx` and the phone's `requestCounts.test.tsx` gain a grant frame that names no person.
- [x] `make event-schemas` and `make live-topics` are clean, and the published contract only grows.
- [x] `doc_list`, `ticket_list` and one live frame: statements and median time on the heavy copy, production build,
      before and after, in the PR.
