# 10 — Stack rules

**Status:** ready-for-agent

**Blocked by:** 04

Read first: `practices/go.md` (section 17), `practices/mcp.md`, `practices/react-guide.md`,
`practices/testing.md`, ADRs 0087, 0091, 0140, 0148, the spec (Entities).

## What to build

- A project's stack takes Everyone, role and person rules for `stacks:*` and `deploys:*`, so for example only
  one role deploys the production stack. The evaluator walks project, then stack. Rules are written with
  `stacks:write` on the stack, under the hold-what-you-give rule.
- Every path that reads, deploys or streams logs for a stack answers through the same check: the web, MCP
  (`stack_*`, `deploy_*`), a branch-driven deploy's start, and live frames. `stack_list` and `deploy_list`
  filter in SQL before paging.
- `permission_overwrite_list` and `_update` accept `resource_type: stack`; explain covers stacks. The stack
  page gains the permissions panel limited to its actions.
- Instance stacks, which belong to no project (ADR 0079), are not covered here; they are checked as today.
- The guide page for stacks and deploys.

## Acceptance criteria

- [ ] `TestStackRules_OnlyTheRoleDeploys`: Everyone Deny `deploys:write`, one role Allow; others get a source
      line in the forbidden answer, over HTTP, MCP and a branch-driven deploy.
- [ ] `TestStackRules_ADeniedStackLeavesTheListAndFrames`.
- [ ] `list_paging_test.go` gains stack rules and stays green; a new
      `TestStatements_StackRulesCostTheSameAtAnyLength`.
- [ ] Verified at 768, 1024 and 1440px.
