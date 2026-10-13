# 10: Requests declare their permission, and a test checks them against the server

Type: task
Status: open
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/typescript.md`, `practices/testing.md`, the spec.

## What to build

- `make permission-catalog` generates `client-core/routePermissions.json` from the router's permission table.
  CI checks it is current.
- Every web and phone mutation hook declares its permission as a `PermissionAction`, the typed union from ticket
  09, so a typo fails `typecheck` before the test runs.
- `client-core/routePermissions.test.ts` fails on a missing or different permission.
- Fix the 8 mismatches in research 01 in this change.
- Add a row to AGENTS.md's enforcement table.

## Acceptance criteria

- [ ] Removing a hook's permission, or changing it, fails the test.
- [ ] Adding a route without regenerating fails CI.
