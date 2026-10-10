# 08 — A deny hides a project

**Status:** needs-info

**Blocked by:** 04, and the owner's answer to the spec's open question 2

Read first: `practices/go.md` (section 17), `practices/testing.md`, ADRs 0087, 0097, 0098, 0140, 0148.

## What to build, if the owner agrees

- A project is not found for a person when a rule on that project (for them, their role or Everyone) denies
  them `projects:read`; the Owner always sees it. A workspace-level deny of `projects:read` keeps its meaning
  today and hides nothing.
- Every list that filters by project (`ProjectsWith`, `CallerProjects`, `ProjectsAnywhere`) and every live
  frame follows it; the project list leaves it out.
- The project panel warns what a deny of "Read projects and board settings" does before it is saved.

## Acceptance criteria

- [ ] `TestPrivateProject_DenyHidesItEverywhere` over HTTP, MCP, lists, search and live frames.
- [ ] The parity and statement guards stay green.
