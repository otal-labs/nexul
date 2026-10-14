# 20: Delete a machine, refused while stacks deploy to it

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- `DELETE /api/machines/{id}` and MCP `machine_delete`: refused while stacks target the machine, naming them.
- It takes Delete (`machines:delete`) on the machine (ADR 0151).
- Publishes `machine.deleted` with a live follower.
- The web adds Delete on the machine card.
- The ADR for the refusing deletes.
- Update `runners.md`.

## Acceptance criteria

- [x] Use-case and integration tests for the refusal and the delete.
- [x] The offline card goes.

## Answer

- `DELETE /api/machines/{id}` takes `machines:delete` on the machine (a machine the caller can't see is not found) and
  answers 204. It is refused with `apperrs.InUseError` (409; the same text over MCP) naming each stack aimed at the
  machine (branch deployments too), each gateway on it, and each runner on it that is connected.
- A free machine goes with its offline runners and their credentials, its rules (the existing trigger), and the unused
  codes for its name, in one transaction with the `machine.deleted` outbox row. A retired runner that returns uninstalls
  itself, as a removed computer's does. Automations hosts are only labelled with a machine name and stay.
- MCP: `host_delete` with `kind: "machine"` (the tool budget has no room for `machine_delete`, and the host family
  already removes by id from `machine_list`). Its `host_create` counterpart refuses with a pointer to `kind: "runner"`.
- `machine.deleted` carries the machine's id and name and the ids of the people who saw it; the audience rule delivers
  it to them only, since the rules are gone by delivery. The web follower drops the card and refetches runners, queue,
  codes and the deploy picker; the phone's runners query refreshes on it.
- The card's ⋯ and right-click menus get Delete for a Delete holder; the confirm names the machine, its rules and its
  offline runners, and a refusal shows as the toast. ADR 0153 records the refusing deletes; `runners.md` and
  CONTEXT.md carry the behaviour.
