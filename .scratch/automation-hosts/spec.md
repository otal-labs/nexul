# Automation hosts follow their machine

**Status:** ready-for-agent

## Problem

An automations host runs workspace automations' code on a machine, yet it is checked as if it were an automation.

- `automations:read`, `automations:write` and `automations:delete` are instance-area actions held through a role in
  any workspace. The same bits let someone edit their own workspace's automations, so every role that may do that
  also lists every automations host (`GET /api/automation-hosts`), mints enrollment codes for new ones and removes any
  of them, the instance's own included.
- `SetHost` places an automation on any enrolled host with no machine check, so whoever edits automations in one
  workspace can run their code on a machine they can't see, or deploy to.
- The "automations" instance area ("automation hosts") hands out those same bits, which ADR 0151 left on roles because
  stripping them would stop people editing their own workspace's automations (#677).

## Decisions

ADR 0154 records them.

1. **Hosts follow their machine, as runners do** (ADR 0151, decision 7). Seeing a host takes See on its machine;
   minting a code for a host on an existing machine and removing a host take Manage there. A host on a machine Nexul
   doesn't know yet is the instance Owner's alone: a host is no way to add a machine, and a machine with no runner has
   no rules to give.
2. **Placing an automation on a host takes Deploy to on the host's machine,** beside `automations:write` in its
   workspace. The bundled instance host (no host id) stays open to every automation, as today. Taking someone off the
   machine leaves automations already placed there running, as a stack keeps its machine; their next move is checked.
3. **`automations:*` become workspace actions.** The catalog moves them from the instance area to the workspace area,
   the "automations" instance area goes, and its rules are deleted in the switch migration. Roles keep the bits.
4. **The enrollment code is checked again when it is used:** a code whose maker no longer Manages the machine, or that
   names a machine Nexul doesn't know and wasn't minted by the instance Owner, is refused, as `makerManages` does for
   runners. Only the instance Owner mints a code with no machine, since the machine it lands on isn't known until the
   host enrolls.

## Where each check goes

- `HostsService.CreateEnrollment`, `List`, `Remove`; `host_create` and `host_delete` with kind `automations`.
- `GET /api/automation-hosts` filters in SQL by the caller's machines (`CallerMachines`), as the machine list does.
- `Service.SetHost` and `automation_update`'s `host_id`; `resolveHost` resolves the machine and asks
  `RequireMachineNamed(…, machines:deploy)`.
- `server/cmd/route_permissions.go`: the host routes stop naming `automations:*`.
- `permissions.go`: the `automations` catalog entry's area, and the `automations` instance area removed.
- `internal/roles/template_test.go`: `TestShippedTemplates_HoldNoInstanceArea` no longer needs the automations carve-out.

## UI

- The automation's host picker lists only hosts on machines the viewer may Deploy to, plus the instance host.
- Add host and Remove host show where the viewer Manages the machine, following the machine's `actions`.
- Settings loses the "automation hosts" row on the Access card.

## Testing

- Use-case tests per check, table-driven over instance Owner, a Manage holder, a Deploy-to holder, See only, and no
  rule.
- `TestAutomationHosts_NotOnTheMachineIsNotListed` over HTTP and MCP.
- `TestAutomations_SetHostRefusesAMachineYouCantDeployTo`.
- A host's code used after its maker was taken off the machine is refused.
- The migration tested from the previous schema: rules on the `automations` instance area are gone, roles keep their
  `automations:*` bits.

## Out of scope

- The workspace-scoped automation surfaces spec's question of whether automations and integrations belong to a
  workspace.
- A machine created by a host alone.

## Commands

```
make sqlc && make sqlc-check
make event-schemas && make live-topics
make lint vet coverage
cd web && bun run lint && bun run typecheck && bun run test
```
