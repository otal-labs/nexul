# 19 — Stack rules

**Status:** ready-for-agent

**Blocked by:** 17

**Branch:** master, a normal PR, after the switch. A new ability; nobody's access changes on upgrade.

Read first: `practices/go.md` (section 17), `practices/mcp.md` (sections 4, 11), `practices/react-guide.md`,
`practices/testing.md` (section 10), ADRs 0079, 0087, 0091, 0140, 0148, the spec (Entities, Evaluation order).

## What to build

- A project's stack takes role and person rules for `stacks:*` and `deploys:*`, so for example only one role deploys
  the production stack. The chain is project, then stack. Rules are written with `stacks:write` on the stack, under
  hold-what-you-give.
- Every path that reads, deploys or streams logs for a stack answers through the same check: the web, MCP (`stack_*`,
  `deploy_*`), a branch-driven deploy's start, and live frames. A stack without `stacks:read` is not found.
  `stack_list` and `deploy_list` filter in SQL before paging.
- `permission_overwrite_list` and `_update` accept `resource_type: stack`; explain covers stacks. The stack page gains
  the panel limited to its actions.
- Instance stacks, which belong to no project (ADR 0079), are checked as today.
- The guide page `stacks-and-deploys.md`.

## Acceptance criteria

- [ ] `TestStackRules_OnlyTheRoleDeploys`: role Developer Deny `deploys:write`, role Frontend Allow, on one stack; a
      Developer without Frontend gets the source in the forbidden answer, over HTTP, MCP and a branch-driven deploy.
- [ ] `TestStackRules_ADeniedStackLeavesTheListAndFrames`.
- [ ] `list_paging_test.go` gains stack rules; new `TestStatements_StackRulesCostTheSameAtAnyLength`.
- [ ] Verified at 768, 1024 and 1440px.
