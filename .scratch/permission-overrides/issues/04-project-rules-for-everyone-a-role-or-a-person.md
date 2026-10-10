# 04 — Project rules for everyone, a role or a person

**Status:** ready-for-agent

**Blocked by:** 02

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 8),
`practices/mcp.md` (sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0087, 0088, 0097, 0135,
0140, 0148, the spec (Evaluation order, Migration plan).

Project access becomes person rules on the project, read for every member, not only Restricted ones.

## What to build

- The evaluator walks the project before the doc: Everyone, role, person on the project, then the doc's rules.
  A Restricted member's project-area base is nothing and only person rules Allow; a project where they hold
  nothing is not found (ADR 0097, unchanged).
- Migration: for every member not restricted today, trim each `project` row's allow to what they already hold
  in that project from their role and workspace rule, so no answer changes.
- `ProjectsWith` reads every project rule that can touch the person in a workspace in one statement.
- `SetEveryProject` and `SetProjectAccess` (`internal/tenancy/project_access.go`) write the same person rows;
  Project access is accepted for any member. Rules for a role or Everyone on a project take `members:write` in
  its workspace and the hold-what-you-give rule on that project.
- `permission_overwrite_list` and `_update` accept `resource_type: project`; `account_update`'s
  `project_access` is accepted for any member.
- `access.grant.changed` for a project role or Everyone row reaches that role's members or the workspace, and
  its managers as today.
- Explain covers projects: every project-area action.

## Acceptance criteria

- [ ] `TestMigration_EffectiveAccessUnchanged` gains From role members with dormant Project access rows; the
      production-copy matrix diff is empty and stated in the PR.
- [ ] `TestDecide_ProjectThenDoc`: the spec's worked example, row by row, with its sources.
- [ ] `TestRestricted_RoleAndEveryoneRulesOnlyTakeAway`, and a Restricted member's new project still reads as
      not found.
- [ ] `list_paging_test.go` gains project rules for each viewer kind and stays green; a new
      `TestStatements_ProjectRulesCostTheSameAtAnyLength`.
- [ ] `ticket_list` and `doc_list` before and after on the heavy copy, in the PR.
- [ ] The tool count is unchanged.
