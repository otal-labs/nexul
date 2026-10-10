# 20 — Install the computer runner as a system service that runs as its user

**Status:** ready-for-agent

**Blocked by:** 02

Read first: `practices/go.md`, `practices/testing.md`, ADRs 0052, 0073, 0074, 0142, 0146,
`internal/install/computer.go`, `internal/install/host.go`, `website/public/computer.sh`,
`website/public/install.sh`, the spec (Installing a personal runner, Security model).

## What to build

Ticket 02 shipped the Linux install as a systemd user unit with lingering, run without sudo. The owner decided
(2026-10-10) that it installs with sudo and runs as a system service instead. Change ticket 02's installer, its
script and the command it returns:

- The command `AddComputer` and `computer_create` return, and the dialog shows, becomes
  `curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`.
- `computer.sh`: runs as root under sudo and installs for `$SUDO_USER`. Run as root with no `SUDO_USER`, or
  `SUDO_USER=root`, it refuses and installs nothing. Run without root, it says to run it with sudo; with no
  `sudo` on the machine (`command -v sudo` fails), it says so plainly and what to do (install sudo and add
  the account to the sudo group). It reads no stdin.
- `install.sh` for the computer kind installs the `nexul` command into that user's `~/.local/bin`, owned by
  that user, and no longer refuses root; other kinds are unchanged.
- `nexul install computer --token`: writes `/etc/systemd/system/nexul-computer.service` with `User=<that
  user>`, `Restart=always`, `WantedBy=multi-user.target`, enables and starts it. The host credential and data
  stay in the user's `~/.local/share/nexul`, owned by them, mode 0600. The runner process never runs as root.
- Remove the lingering step. Remove the user unit: a computer installed by ticket 02's user unit has it
  stopped, disabled and deleted (as the user, best effort) when the new command runs on it, and keeps its
  computer record.
- Self-removal. A process running as the person cannot delete a root-owned unit, so the install also writes a
  root-owned helper that only removes the service, its files and its own sudoers entry
  (`/usr/local/libexec/nexul-computer-uninstall`), and a `/etc/sudoers.d/nexul-computer` entry letting only that
  user run only that helper without a password. A revoked runner and `nexul uninstall computer --detach` use it.
  `sudo nexul uninstall computer` removes the service, helper, sudoers entry and files.
- Self-update (ADR 0052) still works: the runner replaces the user-owned binary and exits for systemd to restart
  it.
- `website/.../guide/paired-computers.md` and `mcp-server.md` show the new command, say what it installs and for
  whom, and say what to do without sudo.

macOS (ticket 08) and Windows (ticket 09) follow this model and are not part of this ticket.

## Acceptance criteria

- [ ] On nexul-box, a normal user with sudo runs the one-liner and ends with an enabled `nexul-computer.service`
      system service, `User=` that user, with the runner process running as that user (`ps -o user`) and the
      computer row showing it connected.
- [ ] The service survives logout (every session closed, runner still connected) and reboot (box restarted,
      runner reconnects with nobody logged in), and the installer enables no lingering for the runner.
- [ ] A root login with no `SUDO_USER`, and `SUDO_USER=root`, are refused and install nothing.
- [ ] Without sudo on the machine, the script prints the no-sudo message; the guide matches.
- [ ] Files under the user's home are owned by the user, the unit file is root-owned, and the host credential
      is mode 0600.
- [ ] A computer installed by ticket 02's user unit moves to the system service by running the new command, with
      the old user unit gone and the same computer row.
- [ ] `nexul uninstall computer` removes the service, helper, sudoers entry and files; removing the computer in
      the app removes the service within a heartbeat through the helper; the sudoers entry passes `visudo -c`
      and allows nothing but the helper.
- [ ] Unit tests for the unit file text, the sudoers entry, the helper, and the `SUDO_USER` cases (set, unset,
      root, no sudo), following `internal/install`'s fakes.
