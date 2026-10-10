# 07 — `nexul install computer` installs T3 Code when it is missing

**Status:** resolved

**Blocked by:** 02

Read first: `practices/go.md`, `practices/testing.md`, ADR 0142, `website/public/tunnel.sh` (`ensure_t3`,
`install_t3`), `website/public/tunnel.ps1`, the spec (Installing a personal runner).

## What to build

Move `tunnel.sh`'s T3 Code logic into `internal/install` as the `computer` install's "T3 Code" step: reuse
a T3 Code that answers or is installed, else install T3 Code's command line with T3 Code's own installer and
`t3 service install` (Linux, macOS), `libatomic1` where missing, or the desktop app with winget (Windows).
Print what it installs; `--no-t3` skips it. Never prompt. The desktop app found closed prints "Open T3
Code".

The installer now runs as root for the person who typed `sudo` (ticket 20), so T3 Code's installer and its
`t3 service install` run as that user (`runuser -u`/`sudo -u`), never as root, and T3 Code lands under that
user's home. T3 Code's background service is a user service on Linux; confirm that against T3 Code's installer
and, if it is, turn on `loginctl enable-linger <user>` so it survives logout (the runner itself needs no
lingering).

## Acceptance criteria

- [x] Table tests over the found states (answering, desktop app, command line idle, none) with a fake
      commander, per OS.
- [x] On nexul-box from the `clean` snapshot: one command ends with T3 Code answering on loopback and the
      computer paired.
- [x] On nexul-box, after the install, logging out every session leaves T3 Code answering.
- [x] The guide's "What the command installs" section replaces the tunnel command's.

## Comments

Built: `internal/install/t3code.go`, the computer install's **T3 Code** step, run after **Command** and before
**Runner**, so a T3 Code that cannot be installed or started stops the install before the code is spent and the
same command works again. `computer.sh` passes what follows the token to `nexul install computer`. No migration.

- Confirmed in T3 Code's source: its Linux service is a systemd **user** unit, `~/.config/systemd/user/t3code.service`
  (`WantedBy=default.target`), and `t3 service install` refuses without a reachable user manager and lingering
  (`cloud/bootService.ts`, `requireSystemdPrerequisites`). So, as root, the step runs `loginctl enable-linger <person>`
  and `systemctl start user@<uid>.service`, then T3 Code's installer and `t3 service install` through
  `runuser -u <person> -- env HOME=<home> XDG_RUNTIME_DIR=/run/user/<uid>`. macOS uses `sudo -u` (ticket 08 has to
  run it); Windows installs the desktop app with winget (ticket 09).
- Found states, in the runner's order: the port T3 Code's `server-runtime.json` names, else 3773 (or `--t3-port`),
  answering on loopback; then the desktop launcher `~/.t3/bin/t3` (`t3.cmd`); then `t3` on root's PATH or the
  person's `~/.local/bin/t3`. Answering is reused as is, plus lingering when its own service runs it
  (`serviceManaged`). A closed desktop app prints "Open T3 Code". A command line that is not running gets its
  background service started (lingering, `t3 service install`) rather than ADR 0142's hint, so the one command
  still ends answering; on Windows it gets the hint. None: `libatomic1` where `ldconfig` lacks it, then T3 Code's
  installer and service. The wait for an answer is bounded by the install's health timeout.
- `--t3-port <port>` replaces the tunnel's `--port`: a drop-in `t3code.service.d/nexul-port.conf` written as the
  person; on macOS another port is refused, as before.
- Box check (nexul-box-t3 from `nexul-box/clean`, v0.3.99-beta.7 built from this branch): the command from
  `POST /api/pairing/computers/enrollments`, run by `alice` through `su -`, installed T3 Code 0.0.45 under
  `/home/alice` (all alice's, nothing under `/root`), `t3code.service` active and enabled in alice's manager,
  `Linger=yes`, `t3 serve` and the runner both running as alice; within a second `computer_list` read
  `paired: true`, `harness_version: 0.0.45`. After a reboot with nobody logged in, and after alice logged in and
  out, T3 Code still answered on 3773 (`State=lingering`, no sessions). Re-running after removing the runner
  printed `T3 Code ......... running on port 3773, its service kept running after you log out`; the binary's inode,
  the unit file's mtime and T3 Code's PID were unchanged. `--no-t3` for `bob` printed `skipped (--no-t3)`, left no
  `~/.t3` or `~/.config` and no lingering for bob, and installed the runner. No model turns were run.
- `loginctl terminate-user` is not a logout: it stops the lingering user manager until the next boot or login.
- For 08: the macOS branch is untested and assumes ticket 08 runs the install as root for the person; T3 Code's
  macOS service is a LaunchAgent that starts only in a GUI login. For 09: the Windows branch runs winget as the
  installing account. For 10: T3 Code's unit is `t3code.service` in the person's user manager, which lingering keeps
  running from boot; the runner's system service has no `XDG_RUNTIME_DIR`, so a restart needs
  `XDG_RUNTIME_DIR=/run/user/<uid>` (the bus is `$XDG_RUNTIME_DIR/bus`). A computer installed with `--no-t3` gets no
  lingering from Nexul; T3 Code's own `t3 service install` asks for it.
