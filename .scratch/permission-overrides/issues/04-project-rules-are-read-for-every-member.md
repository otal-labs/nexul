# 04 — Project rules are read for every member

**Status:** done

**Blocked by:** 02, 03

**Branch:** master, a normal PR, before the switch. The rules are read for everyone and still applied only to
Restricted members.

Read first: `practices/go.md` (section 17), `practices/testing.md` (sections 3, 8, 10), ADRs 0097, 0135, 0140, 0148,
the spec (Projects are opt-in, Performance plan), the research (sections 1.1 to 1.4 and 9).

The read half of opt-in projects, landed early so production runs it for days before anything is wiped.

## What to build

- `layers()`, `ProjectsWith` and the memo read the project rules for the person and their roles, for every member, in
  one statement per workspace per request. They still apply only to Restricted members.
- `ListProjectPeople` (`internal/tenancy/project_access.go:197-216`) becomes one SQL query, and the workspace project
  list (`internal/workspace/usecase.go:144-166`) uses the `ProjectsWith` batch instead of a check per row.
- The fixtures of `access_memo_test.go`, `list_paging_test.go` (`listParityFixture`) and
  `permissions_integration_test.go` (`seedPermFixture`) list every non-Owner reader in their projects with a project
  rule. The answers don't change yet, so after the flip no guard passes for the wrong reason.

## Acceptance criteria

- [x] An empty `access.Matrix` diff, master against the branch, in the PR. A fixture replaces the production copy, as for
      slice 02: a schema-0098 database holding every member and rule kind (person and role rules on docs, projects and
      plays, a workspace-wide allow and deny, several roles on one member, a Restricted member with Project access and
      role rules on its project), 192 rows on both sides.
- [x] `TestProjectsWith_ReadsARestrictedMembersAccessOnce` becomes `…ReadsEveryMembersProjectRulesOnce`.
- [x] New `TestStatements_ProjectRulesCostTheSameAtAnyLength`; the project list and the project-people query join
      `TestStatements_ListsCostTheSameAtAnyLength`.
- [x] `TestRows_ListToolsReadOnlyTheirPage` gains a non-Owner viewer for `doc_list`.
- [x] `ticket_list` and `doc_list` statements and median time on the heavy copy, before and after, in the PR. A
      single check or live frame reads only its project's rules; lists read the workspace's in one batch. Production-
      shaped, every path stays inside 10%; with the access grid filled `doc_list` is 11% slower, and with every member
      named on every project 16%, from decoding one rule row per listed project: raised with the owner before merging.

- Owner, 2026-10-11: doc_list's +11% (grid filled) and +16% (every member named on every project) are accepted; cheaper rule decoding is held for the monthly performance audit.
