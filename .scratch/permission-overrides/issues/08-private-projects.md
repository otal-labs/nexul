# 08 — A deny hides a project

**Status:** ready-for-agent

**Blocked by:** 04

Read first: `practices/go.md` (section 17), `practices/testing.md`, ADRs 0087, 0097, 0098, 0140, 0148, the spec
(Evaluation order, step 7).

## What to build

- A project is not found for a person when the rules on that project itself (Everyone, their role, or them),
  taken in the usual order, end in Deny for `projects:read`. The Owner always sees it. A deny of `projects:read`
  anywhere else (the workspace rule, a role) keeps its meaning today and hides nothing.
- The check runs at the project before the chain walks inward: nothing inside a hidden project is reachable,
  and a doc's own rule for the person does not reopen it.
- Every list that filters by project (`ProjectsWith`, `CallerProjects`, `ProjectsAnywhere`), search, links and
  every live frame follow it; the project list leaves it out.
- The project panel warns what a deny of "Read projects and board settings" does before it is saved.

## Acceptance criteria

- [ ] `TestPrivateProject_DenyHidesItEverywhere` over HTTP, MCP, lists, search and live frames.
- [ ] `TestPrivateProject_OnlyADenyOnTheProjectHides`: a workspace or role deny elsewhere hides nothing, and
      the Owner still sees the project.
- [ ] `TestPrivateProject_ADocRuleDoesNotReopenIt`.
- [ ] A Restricted member's project is hidden by the same deny, and a hidden project never explains itself.
- [ ] The parity and statement guards stay green.
