# 03: The switch: See, Manage, Delete and Add machines enforced

Type: task
Status: open
Blocked by: 02

Read first: `AGENTS.md`, `CONTEXT.md`, `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2 to
8), `practices/mcp.md` (sections 4, 6, 7), `practices/react-guide.md` (The live topic contract), `practices/native.md`
(section 4), `practices/testing.md` (sections 3, 8, 10), ADRs 0087, 0140, 0146, 0148, 0150, the spec (Where each check
goes).

Changes access on upgrade: every machine starts with only the instance Owner on it.

## What to build

- **Lists filter in SQL** by the machines the caller Sees, before paging: `ListMachines`, `GET /api/machines`,
  `machine_list` (its runners, queue and automations hosts too), `ListRunners`, the queue, the topology canvas's
  machines and the setup pass's machine read. Join the parity and row guards in `server/cmd/list_paging_test.go`.
- **One machine:** `GetMachine` and anything that resolves one answers not found without See. Discover and
  `UpdateMachine` take Manage; runner removal takes Manage on its machine; the role-aware UI's machine delete takes
  Delete if it has landed.
- **Enrollment:** a code for a runner on an existing machine takes Manage on it; one for a new machine name takes Add
  machines. The code records who made it (a migration on the enrollment row), and enrolling a new machine writes a
  rule naming that person with all four actions in the same transaction. A code made for a new name that is taken by
  the time it's used is refused, so a code can't land a runner on someone else's machine.
- **Live:** `runner.connected` and `runner.disconnected` gain the machine id beside their fields (`make
  event-schemas`), and their audience is See on that machine. The web's and the phone's runner followers keep working.
- **The catalog:** `machines:read`, `runners:read`, `runners:write` and `runners:delete` go; the `runners` instance
  area becomes `machines` and lists `machines:create`. A numbered forward-only migration removes the gone actions from
  every role, token scope and instance-area rule, and rekeys area rules from `runners` to `machines`.
- The web's `RunnersPage` stops needing `machines:read` to load; it shows what the server returns. `client-core`
  and `web/src/models/Workspace.tsx` follow the catalog.
- Guide: `runners.md`.

## Acceptance criteria

- [ ] `TestMachines_NotOnTheMachineIsNotFound` over HTTP, MCP (`machine_list`, `machine_discover`) and a live frame.
- [ ] `TestEnrollment_ANewMachineGivesItsAdderEveryAction`, `TestEnrollment_AnExistingMachineTakesManage`,
      `TestEnrollment_ACodeCantLandOnAMachineTakenSince`.
- [ ] `TestMigration_RemovesTheGoneMachineBits`, upgrading from the previous schema with roles, tokens and area rules
      holding them.
- [ ] `TestStatements_MachineListCostsTheSameAtAnyLength`.
- [ ] Verified at 768, 1024 and 1440px as the instance Owner and as someone on one machine.
