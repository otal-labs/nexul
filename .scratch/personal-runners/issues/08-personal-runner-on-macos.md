# 08 — Personal runner on macOS

**Status:** resolved

**Blocked by:** 03, 07, 20

Read first: `practices/go.md`, `practices/testing.md`, ADR 0073, `internal/install/host.go`, the spec
(Installing a personal runner).

## What to build

`nexul install computer` on macOS, after Linux (owner, 2026-10-10), by `curl -fsSL <site>/computer.sh | sudo sh -s -- <token>`:
installs for `$SUDO_USER` and refuses a root login with no `SUDO_USER` (ticket 20's rules), into
`~/Library/Application Support/nexul`, a LaunchDaemon `nexul-computer` in `/Library/LaunchDaemons` with
`UserName` set to that user and `KeepAlive`, no Docker step, then T3 Code run as that user (ticket 07).
`computer.sh` handles Darwin. The removal follows ticket 20's design: a root-owned cleanup a request file triggers (a LaunchDaemon with `WatchPaths` on macOS), no sudo rule.

## Acceptance criteria

- [x] Unit tests for the plist (`UserName`, `KeepAlive`) and paths with the existing macOS fakes.
- [x] The daemon runs as that user, starts at boot and survives logout; root login without `SUDO_USER` is
      refused.
- [x] A run on a real Mac (a GitHub macOS runner is enough for install, connect and relay against a
      throwaway server), reported in the PR.

## Comments

Built: `internal/install/computer.go` (the macOS branch of the computer install, its cleanup daemon), `services.go`
(a unit with a `User` loads as a LaunchDaemon in launchd's `system` domain with `UserName`), `website/public/computer.sh`
(Darwin allowed) and `install.sh` (the person's home from `dscl` on macOS, which has no `getent`). No migration.

- Layout on a Mac: `~/.local/bin/nexul`, the runner and its credential (0600) under `~/Library/Application Support/nexul/computer`,
  its log in `~/Library/Logs/nexul/nexul-computer.log` (made the person's before launchd, as root, opens it), all the
  person's; `/Library/LaunchDaemons/io.nexul.nexul-computer.plist`, root's, with `UserName`, `RunAtLoad` and
  `KeepAlive {SuccessfulExit: false}`, the launchd form of Linux's `Restart=always` plus `RestartPreventExitStatus=0`, so
  a removed runner's clean exit is not restarted. launchd sets no `HOME` for a daemon, so the plist carries it; the
  runner finds T3 Code under it.
- Cleanup: root's copy at `/usr/local/libexec/nexul-computer-uninstall`, the request folder
  `/Library/Application Support/nexul-computer` (0700, the person's), and `io.nexul.nexul-computer-cleanup`, a root
  LaunchDaemon with `WatchPaths` on that folder. A watch fires for any change, and on the real Mac it fired once when
  the daemon loaded, so `--cleanup-for` now does nothing unless `remove-requested` exists (on Linux the path unit
  already guaranteed that). The removal deletes the person's files as them through `sudo -u` (macOS has no `runuser`)
  and unloads the cleanup daemon last, because that ends its own process.
- T3 Code: the install runs T3 Code's installer and `t3 service install` through `sudo -u <person> env HOME=<home>`.
  T3 Code's macOS service is a LaunchAgent in `gui/<uid>` (`cloud/bootService.ts`, `launchdManager`), which launchd
  runs only while the person is logged in at the screen; an SSH login has no `gui` domain and T3 Code's own bootstrap
  fails there. So a Mac's runner is connected from boot and after logout, but T3 Code, and with it the computer's runs,
  waits for that login; with FileVault on, nothing runs at all after a restart until someone logs in. The installer
  says so on its last line, and the guide, the spec's macOS row and ADR 0146's consequences say it too. `--t3-port` stays
  refused on macOS, since T3 Code's LaunchAgent always uses 3773.
- Restarting T3 Code (ticket 10) needs nothing extra on macOS: `t3 service restart` run by the runner from the
  `system` domain, as the person's uid, booted out and bootstrapped `gui/501` while the person was logged in. Logged
  out, it fails and the runner reports `restart_error`.
- Real Mac (`.github/workflows/computer-macos-smoke.yml`, `workflow_dispatch` only, `macos-latest`, arm64): the branch's
  server, `nexul` and runner built for darwin and served with `website/public/*.sh` by `python3 -m http.server`; a
  throwaway instance on the runner with dev login, `NEXUL_SITE_URL` and `NEXUL_RELEASE_URL` pointed at it; the command
  from `POST /api/pairing/computers/enrollments` run as `runner` (uid 501, in its GUI session). Green run
  https://github.com/otal-labs/nexul/actions/runs/38092425853 on the branch rebased over #564:
  `sudo env -u SUDO_USER sh` refused with "from your own account with sudo, not logged in as root" and nothing
  installed; the installer printed `Runner .......... nexul-computer, computer-j4mkohua`; the plist root-owned with
  `UserName` `runner` and `SuccessfulExit` false; the process `runner 1 …/nexul-runner` (parent launchd); `kill -KILL`
  on it and launchd started a new one (`killed 3635; launchd started it again as 3909`); the computer row
  `runner.connected: true`, `harness_version: 0.0.45`, `token_expires_at` 30 days on, `pair_error: null`, so the pairing
  exchange went through the relay; `GET …/providers` relayed to T3 Code answered `{"providers":[]}`; T3 Code's agent
  booted out with `launchctl bootout gui/501/com.t3tools.t3code.service` answered again 41 s later after the runner's
  `T3 Code's background service stopped answering; restarted it`; `DELETE /api/pairing/computers/{id}` and the runner's
  request had the cleanup leave no daemon loaded and none of the paths above. The real T3 Code ran headless there.
- Not run there: a reboot and a real logout (a hosted runner can do neither). Boot rests on `RunAtLoad` in the
  `system` domain, and logout on the daemon living outside the person's `gui` domain, which the restart from that
  domain into `gui/501` also shows. T3 Code's installer itself (it looks its release up on GitHub's API, which refuses
  the shared runners unsigned in): the workflow puts T3 Code's command line on the account with a pinned version first,
  and the command then starts its service (`t3 service install`), the "command line, not running" path. `upgrade` of a
  computer is untouched.
- To dispatch it, the workflow has to be on `master`: until this merges, `gh workflow run` answers 404, and the runs
  above came from a `push` trigger on this branch that the last commit removes.
- For 09: the install now branches on `darwin` in `computerPerson`, `computerPaths`, `computerService`, `asPersonArgs`,
  `installCleanup`/`removeCleanup` and `UninstallComputer`; Windows adds its own branch to each. `cleanUpComputer`'s
  check for the request file suits any trigger that can fire spuriously. The workflow's shape (build, serve the release
  and scripts locally, throwaway instance, enrollment, command, row, relay, removal) carries over to `windows-latest`,
  which also has to land on `master` before it can be dispatched.

