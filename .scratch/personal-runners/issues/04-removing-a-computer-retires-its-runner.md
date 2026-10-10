# 04 — Removing a computer, or disabling its owner, retires the runner

**Status:** ready-for-agent

**Blocked by:** 02, 20

Read first: `practices/go.md`, `practices/architecture.md` (Principles, section 6), `practices/testing.md`,
ADRs 0074, 0088, the spec (Access and privacy, rule 4).

## What to build

- Removing a computer with a runner tombstones the runner's credential and sends `uninstall`, so the
  runner removes its own service through the root-owned removal helper ticket 20 installs (offline ones on their
  next connect, ADR 0074). Its relay streams close.
- `RevokePersonalRunners(userID)` on the runner domain, called when an account is disabled or removed. It
  returns nothing about the computers; the admin sees "Their computers were disconnected".
- On uninstall, the runner revokes T3 Code sessions labelled `Nexul` with `t3 auth session list --json`
  and `revoke`, best effort, so removing a computer ends Nexul's access inside T3 Code too.
- `website/.../guide/paired-computers.md`: Remove says what it now undoes.

## Acceptance criteria

- [ ] `TestRemoveComputer_RevokesItsRunner`, `TestAccountDisabled_RevokesTheirPersonalRunners`,
      `TestAccountRemoved_RevokesTheirPersonalRunners`.
- [ ] The admin's response and events carry no computer id, name or count beyond the account itself.
- [ ] Reactivating the account restores no runner; adding the computer again adopts the old row (covered
      with ticket 12's adoption; until then the row stays and shows "Add this computer again").
- [ ] Manual check on nexul-box: removing the computer in the UI removes the `nexul-computer` system service within a heartbeat.
