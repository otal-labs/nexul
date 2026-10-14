# Machine permissions: each machine says who sees it and what they may do on it

**Status:** ready-for-agent. The permission rework (ADR 0148) has merged; this builds on its rules.

## Problem

A machine is root on a server: whatever deploys there runs in its Docker. Yet every machine is one switch.

- `ListMachines` checks the instance bit `machines:read` and returns every machine. A role holding that bit in any
  workspace sees every server on the instance, the instance's own host included.
- A stack's machine is a free string. `CreateStack`, `UpdateStack`, branch deployments, gateways and the proxy and
  tunnel steps check only `stacks:write` (or `dns:write`), never the machine, so anyone who may edit stacks in one
  project can aim a compose file at any machine, even one they can't list, by typing its name into `stack_create`.
- Import and Discover check nothing about the machine, and Discover lists every container on it.
- An instance-area rule (ADR 0148) can hand out "runners and machines" as a whole, never one machine.
- An Owner of any workspace holds every instance area and can't be denied, so whoever creates a workspace holds the
  instance: its machines, connectors, accounts and upgrades.

## Decisions

From the owner, 2026-10-14. ADR 0151 records the ones that are hard to reverse.

1. **Each machine carries rules**, for a role (of any workspace) or a person, in Allow, Fallback or Deny, through the
   same permissions panel as stacks and computers. Its actions are **See**, **Deploy to**, **Manage** and **Delete**.
   This replaces the earlier idea of machines belonging to workspaces.
2. **Machines are opt-in, like projects.** See is held only through a machine's rules, never by a role alone. A
   machine someone can't See is not found everywhere: lists, pickers, MCP, live frames, topology. Being on a machine
   (See), their roles' machine bits decide the other three, and the machine's rules Allow or Deny on top: the person's
   roles' rules merged with Allow winning, then a rule naming the person, which beats them.
3. **The instance Owner** (the Owner of the default workspace) sees and holds everything on every machine and is never
   denied. Personal runners stay the one exception: only the person who installed one shares it (ADR 0146, 0148).
4. **Owners of other workspaces hold no instance area by right.** Inside their workspace they stay full Owner. A
   machine, a connector, accounts, upgrades or any other instance area reaches them only through a rule, like anyone
   else. This narrows ADR 0148's "an Owner of any workspace is never denied an area".
5. **Who writes a machine's rules:** the instance Owner, and anyone holding Manage on that machine, who can't allow or
   lift a Deny on what they don't hold there (ADR 0088). Adding a Deny is never limited.
6. **Deploy to is checked when something is aimed at a machine** (creating a stack, moving one, importing, creating a
   gateway or a proxy or tunnel stack) **and on every deploy a person starts.** A deploy the server starts, after a
   push or a merge, passes as the server. Taking someone off a machine stops them deploying there on the next check.
7. **Runners follow their machine.** Permissions are per machine, never per runner: every runner on a box shares its
   Docker. Seeing a runner, the queue's jobs for it, and its live frames take See on its machine; adding a runner to an
   existing machine or removing one takes Manage. Adding a **new** machine (a runner under a machine name not yet
   known) takes **Add machines**, an instance-area permission, and whoever adds it gets a rule naming them with all
   four actions. They hold no special status: Manage holders and the instance Owner can take them off.
8. **Seeing** a stack that belongs to no project (a gateway, the instance's tunnel agent) takes See on its machine and
   `dns:read`; editing or deploying one takes Deploy to and `dns:write`. Without rules, only the instance Owner.
9. **A stack shows its machine's name** to someone who can't See the machine, as plain text: no link, no details.
10. **On upgrade every machine starts with only the instance Owner on it.** No rule exists, the role bits that meant
    "every machine" are removed, and deploys the server starts keep working. The instance Owner adds people back from
    each machine's panel.

## The actions

| On screen | Action | Held through | Area |
|---|---|---|---|
| See | `machines:see` | the machine's rules only, like `projects:open` | off the grid; `ParseAction` refuses it |
| Deploy to | `machines:deploy` | roles, then the machine's rules | machine |
| Manage | `machines:write` | roles, then the machine's rules | machine |
| Delete | `machines:delete` | roles, then the machine's rules | machine |
| Add machines | `machines:create` | roles, then the "machines" instance area's rules | instance area |

`machines:read`, `runners:read`, `runners:write` and `runners:delete` leave the catalog. The instance area keyed
`runners` ("runners and machines") becomes `machines` ("adding machines") and lists `machines:create` only. Role
permission sets, token scopes and instance-area rules drop the removed actions in the switch migration.

## Evaluation

For person `P`, action `A`, machine `M`:

1. `P` is not signed in or not active: refused.
2. `P` is the instance Owner: allowed, never denied. An Owner of another workspace is not special here.
3. See: `M`'s rules for the roles `P` holds in any workspace, Allow winning, then the rule naming `P`. Not held: `M` is
   not found, and every other action answers not found too.
4. Any other action: the union of `P`'s roles in every workspace they're in holds `A` or not; then `M`'s rules for
   those roles, Allow winning; then the rule naming `P`. The answer names its source, as every ADR 0148 answer does.

The server's own calls pass, as `RequireArea` lets them today. The check resolves a machine by id, or by name where a
stack or gateway carries one; an unknown name is invalid input, never a free string.

## Where each check goes

- **Lists:** `ListMachines`, `machine_list`, `GET /api/machines`, `ListRunners`, the queue, the topology canvas's
  machines, the automations hosts filed under a machine, the setup pass's machine read. Filtered in SQL by the
  caller's machines before paging (practices/go.md section 17, ADR 0140), joining the parity and row guards in
  `server/cmd/list_paging_test.go`.
- **One machine:** `GetMachine`, Discover (Manage), `UpdateMachine` (Manage), machine delete (Delete), enrollment for a
  runner on an existing machine (Manage) or a new one (Add machines), runner removal (Manage).
- **Aiming at a machine** (Deploy to): `CreateStack`, `UpdateStack` when the machine changes, branch deployments,
  Import, `CreateGateway`, the proxy and tunnel provisioning paths, `stack_create`, `stack_update`, `machine_import`,
  `gateway_create`.
- **Deploying** (Deploy to, plus the stack's own checks): `Deploy`, retry, `stack_deploy`, Cancel stays with the stack.
- **Live:** `runner.connected` and `runner.disconnected` carry the machine id (a new field beside the old ones,
  `make event-schemas`) and reach only who Sees the machine; a new rule on a machine reaches its holders through
  `access.grant.changed`. `make live-topics`.
- **Pickers:** `MachinePicker` lists machines the viewer may Deploy to.
- **Catalogs:** `client-core/permissions.ts`, `web/src/models/Workspace.tsx` and, once the role-aware UI guard has
  landed, its generated catalog follow the table above.

## UI

- Runners page: each machine card gets **People on this machine**, the permissions panel limited to See, Deploy to,
  Manage and Delete, targeting a role of any workspace or a person. Shown to the instance Owner and Manage holders.
- Settings: Add machines is handed out on the "machines" instance area's Access card.
- Stack pages: the machine as plain text when the viewer can't See it.
- The phone's Runners screen lists what the server returns, refreshed by the same topics.

Verified at 768, 1024 and 1440px.

## Out of scope

- A remote shell on a machine, which becomes its own action when it's built.
- Per-runner permissions (decision 7).
- Machines belonging to workspaces, dropped in favour of rules.

## Overlaps

- `.scratch/role-aware-ui/` ticket 20 (machine delete) takes Delete on the machine once ticket 03 here lands; whichever
  lands second adapts. Ticket 12's runners and machines sweep reads the actions above.
- `.scratch/workspace-scoped-automation-surfaces/` no longer covers runners; this effort does.

## Testing

- Use-case tests per check, table-driven over instance Owner, another workspace's Owner, a role rule, a person rule,
  Deny beating a role, no rule.
- `TestMachines_NotOnTheMachineIsNotFound` over HTTP, MCP and a live frame.
- `TestMachines_StackCreateRefusesAMachineYouCantDeployTo`, by name over MCP included.
- `TestMachines_TakenOffStopsTheNextDeploy`; `TestMachines_APushDeployStillRuns`.
- `TestInstanceAreas_AnotherWorkspacesOwnerHoldsNoArea`.
- The switch migration tested by upgrading from the previous schema with prod-shaped fixtures (roles holding the
  removed bits, instance-area rules on `runners`).
- E2E on `nexul-box` with two people: the instance Owner and a workspace Owner of another workspace.

## Commands

```
make sqlc && make sqlc-check
make event-schemas && make live-topics
make lint vet coverage
cd web && bun run lint && bun run typecheck && bun run test
cd native && bun run lint && bun run typecheck && bun run test
```
