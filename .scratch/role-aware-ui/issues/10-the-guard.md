# 10: Requests declare their permission, and a test checks them against the server

Type: task
Status: resolved
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

- [x] Removing a hook's permission, or changing it, fails the test.
- [x] Adding a route without regenerating fails CI.

## Answer

- The server had no route to permission table, only checks inside the use-cases, so `server/cmd/route_permissions.go`
  is one: every writing route of the gateway, keyed `METHOD /path`, names the catalog action its use-case checks.
  `none` marks a route no role permission governs (a person's own records, public sign-in and enrollment); several
  actions mean the target (a doc or a play, a channel or a DM) decides.
  `TestRoutePermissions_NameEveryWritingRouteAndOnlyCatalogActions` walks the real router and fails on a route with no
  row, a row with no route, or an action off the catalog. `make permission-catalog` also writes
  `client-core/routePermissions.json`, and `TestRoutePermissionsFile_MatchesTheRoutes` fails when it is stale.
- The declaration is `requires(permission, request)` from `client-core/requires.ts`, an identity wrapper around the
  function that sends a request, or around the call itself. Hooks keep calling `api.post(url, body)`, so no existing
  mock changes. The permission is a `PermissionAction` or `"none"`, so a typo fails `typecheck`.
- `client-core/routePermissions.test.ts` parses every web and phone source, finds each `api.post/put/patch/delete`
  and `requestJSON` write, resolves its URL (strings, templates, same-file constants, top-level path helpers,
  either branch of a ternary) and fails on a write with no `requires`, a permission its route doesn't take, a route
  that doesn't exist, a `requires` around no request, or a URL it can't read.
- The mismatches (all the client; the server's checks stand):
  - Board New category took tickets:write; it needs projects:write. The Create menu rows now each carry theirs.
  - The wizard rows (tests location, stack, env and first deploy, branch rules, import, attach) were gated on
    projects:create. A door now needs what its first steps write (`canRunWizard`): a new project, projects:create and
    projects:write (a project can finish with no service); adding a service or attaching a repository, projects:write,
    stacks:write and deploys:write. The route opens for either door, Import needs Manage on the machine and
    stacks:write, and Expose service needs dns:write.
  - New direct message hid the case the server allows: anyone may message an Owner.
  - Pass and Fail were open to anyone who sees the ticket; the server needs tickets:write.
  - The stack's Exposures tab read the project's workspace permissions for DNS, which is held anywhere.
  - Delete role asked for roles:delete, which no use-case checks; it needs roles:write. Clone to workspace takes its
    place as the role menu's gated row.
- Left for the sweeps: per-control locking inside the wizard's steps (ticket 11), the role-wide against per-item
  answers (ticket 15) and clone destinations.
