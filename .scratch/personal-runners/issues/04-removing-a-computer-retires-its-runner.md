# 04 — Removing a computer, or disabling its owner, retires the runner

**Status:** resolved

**Blocked by:** 02, 20

Read first: `practices/go.md`, `practices/architecture.md` (Principles, section 6), `practices/testing.md`,
ADRs 0074, 0088, the spec (Access and privacy, rule 4).

## What to build

- Removing a computer with a runner tombstones the runner's credential and sends `uninstall`, so the
  runner removes its own service through the root cleanup path unit ticket 20 installs (offline ones on their
  next connect, ADR 0074). Its relay streams close.
- `RevokePersonalRunners(userID)` on the runner domain, called when an account is disabled or removed. It
  returns nothing about the computers; the admin sees "Their computers were disconnected".
- On uninstall, the runner revokes T3 Code sessions labelled `Nexul` with `t3 auth session list --json`
  and `revoke`, best effort, so removing a computer ends Nexul's access inside T3 Code too.
- `website/.../guide/paired-computers.md`: Remove says what it now undoes.

## Acceptance criteria

- [x] `TestRemoveComputer_RevokesItsRunner`, `TestAccountDisabled_RevokesTheirPersonalRunners`,
      `TestAccountRemoved_RevokesTheirPersonalRunners`.
- [x] The admin's response and events carry no computer id, name or count beyond the account itself.
- [x] Reactivating the account restores no runner; adding the computer again adopts the old row (covered
      with ticket 12's adoption; until then the row stays and shows "Add this computer again").
- [x] Manual check on nexul-box: removing the computer in the UI removes the `nexul-computer` system service within a heartbeat.

## Comments

Built: `RetireComputerRunner` and `RevokePersonalRunners` in `internal/runner/enrollment.go`, the
`account.disabled` / `account.removed` consumer `HandleAccountClosed` (`runner.revoke_personal`), the T3 Code
session revoke in `internal/runner/t3sessions.go`, and `DeleteComputer` retiring the runner first. No migration.

- Both paths go through the runner domain's existing removal: the credential is tombstoned, the row deleted, relay
  streams closed, `uninstall` sent to a connected runner, and an offline one is refused as removed on its next
  connect. They also delete the unused enrollment codes for the computer, or for every computer of the account, so a
  command minted earlier and run afterwards enrolls nothing (`invalid_code`).
- A failed retire keeps the computer row, so removing it again retries.
- The account path is a bus consumer on the outbox events the account change already writes; the HTTP and MCP
  answers are unchanged (204, `{id, deleted}`) and name no computer. The web toasts say "Account disabled. Their
  computers were disconnected." and "Account removed. Their computers were disconnected.", the same whatever the
  person had; the confirmations and the account tools' descriptions say so beforehand.
- T3 Code sessions: before `nexul uninstall computer --detach`, the runner runs `t3 auth session list --json` and
  `t3 auth session revoke <id>` for each session whose `client.label` is `Nexul`, with a 30-second cap; `t3` is
  found on `PATH`, then `$T3CODE_HOME/bin/t3`, then `~/.local/bin/t3`. A `nexul uninstall computer` the person
  runs themselves reaches it only through the `uninstall` frame, so the root cleanup may stop the runner first.
- Box check (nexul-box-retire, v0.99.4-beta.1 built from this branch, a fake `t3` under alice's `~/.t3/bin`):
  `DELETE /api/pairing/computers/{id}` at 20:47:09.34, unit file gone by 20:47:09.86 and the cleanup finished at
  20:47:10 (service, cleanup units, root helper, request folder and `~/.local/bin/nexul` gone; `t3` saw
  `auth session list --json` then `auth session revoke s-nexul`). With the service stopped, the removal left the
  unit in place; starting it was refused as removed and the unit was gone 0.5s later. The Owner disabling alice got
  `204` with an empty body, her runner uninstalled within a second, her unused command answered `invalid_code`, the
  audit log named neither computer, and after reactivating her both rows stayed with no runner and a fresh command
  for the old row was `201`.
- For 06: the list cannot tell a revoked runner from one never installed (both have no `runner`); either row offers
  "Add this computer again", which is `POST /api/pairing/computers/enrollments {id}`. The server keeps the row and
  mints the command; ticket 12 adds adoption on top.
- For 12: adoption must not resurrect a revoked runner row; the revoked credential stays a tombstone.
