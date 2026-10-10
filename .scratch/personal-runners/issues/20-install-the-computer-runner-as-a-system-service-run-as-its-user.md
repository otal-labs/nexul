# 20 — Install the computer runner as a system service that runs as its user

**Status:** resolved

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
- Self-removal, with no sudo rule (owner, 2026-10-10). A process running as the person cannot delete a root-owned
  unit, so the install also leaves a root cleanup: a root-owned copy of `nexul`
  (`/usr/local/libexec/nexul-computer-uninstall`), a request folder only that user may write in
  (`/var/lib/nexul-computer`), and a path unit `nexul-computer-cleanup.path` that starts the oneshot
  `nexul-computer-cleanup.service` once `remove-requested` exists there. The cleanup does one fixed removal, takes
  no input from the request, and is idempotent. A revoked runner and `nexul uninstall computer --detach` write the
  request. `sudo nexul uninstall computer` removes the service, the cleanup and the files at once.
- Self-update (ADR 0052) still works: the runner replaces the user-owned binary and exits for systemd to restart
  it.
- `website/.../guide/paired-computers.md` and `mcp-server.md` show the new command, say what it installs and for
  whom, and say what to do without sudo.

macOS (ticket 08) and Windows (ticket 09) follow this model and are not part of this ticket.

## Acceptance criteria

- [x] On nexul-box, a normal user with sudo runs the one-liner and ends with an enabled `nexul-computer.service`
      system service, `User=` that user, with the runner process running as that user (`ps -o user`) and the
      computer row showing it connected.
- [x] The service survives logout (every session closed, runner still connected) and reboot (box restarted,
      runner reconnects with nobody logged in), and the installer enables no lingering for the runner.
- [x] A root login with no `SUDO_USER`, and `SUDO_USER=root`, are refused and install nothing.
- [x] Without sudo on the machine, the script prints the no-sudo message; the guide matches.
- [x] Files under the user's home are owned by the user, the unit file is root-owned, and the host credential
      is mode 0600.
- [x] A computer installed by ticket 02's user unit moves to the system service by running the new command, with
      the old user unit gone and the same computer row.
- [x] `nexul uninstall computer` removes the service, the cleanup and the files; a revoked runner removes the
      service within a heartbeat through the cleanup; the cleanup ignores what the request file says.
- [x] Unit tests for the unit file text, the cleanup units, the cleanup ignoring the request, and the `SUDO_USER`
      cases (set, unset, root, no sudo), following `internal/install`'s fakes.

## Comments

Built: `internal/install/computer.go` (install for `$SUDO_USER`, the system unit, the cleanup, the move from the user
unit, the shared removal), `website/public/computer.sh` and `install.sh`, `hostcred.ComputerCommand` (`| sudo … sh`).

- The command is `curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`; a release or site override renders as
  `| sudo NEXUL_VERSION=… sh`, which sudo accepts for a member of the `sudo` group.
- The unit carries `User=<person>` and no `Group=` (a person's primary group need not share their name), and
  `RestartPreventExitStatus=0`: a removed runner exits 0, so a failed cleanup never turns into a restart loop.
- The cleanup's service names the person in its own root-owned unit (`--cleanup-for alice`). It reads neither the
  request nor the runner's `unit.json`, sends nothing to the instance, and deletes the person's files through
  `runuser -u <person> -- rm -rf`, so a link in their home leads root nowhere.
- `nexul uninstall computer` run as the person tells the instance and writes the request; under sudo it removes
  everything at once. sudo's `secure_path` skips `~/.local/bin`, so the sudo form is
  `sudo ~/.local/bin/nexul uninstall computer`; the guide and the installer's output name the form without sudo.
- The unit name is fixed, so one person per machine for now: a second account gets "installed for another account".
- Moving a ticket 02 install: the new command finds the record and the user unit, stops and deletes the user unit
  as the person, and keeps the credential, so the token goes unused. No release carried the user unit when this
  was built (v0.3.33-beta predates #555), but tonight's beta may. Lingering is left as it was.
- Self-update: the runner's directory and binary are the person's, so ADR 0052's rename and re-exec need nothing
  new. Not run end to end: the server resolves updates through GitHub's release API, which knows no local tag.
- For 04: the runner already calls `nexul uninstall computer --detach` on an `uninstall` frame or a
  `runner_removed` refusal, and that now writes the request. Tombstoning the credential when a computer is removed
  in the app (and sending the frame) is all 04 adds; a credential revoked through `/api/runners/self/remove`
  removed the install within 2 seconds on the box.
- For 07: the installer runs as root for the person; run T3 Code's installer as them (`runuser -u`). For 10: the
  runner's system service has `HOME` but no `XDG_RUNTIME_DIR`; restarting T3 Code's user service needs
  `XDG_RUNTIME_DIR=/run/user/<uid>` and lingering on.
