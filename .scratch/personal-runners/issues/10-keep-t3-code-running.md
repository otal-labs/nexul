# 10 — Keep T3 Code running

**Status:** resolved

**Blocked by:** 03, 20

Read first: `practices/go.md` (section 7), `practices/testing.md` (section 9), the spec (Finding T3 Code).

## What to build

The runner probes T3 Code every 30 seconds. A T3 Code its background service runs that misses two probes
is restarted with `t3 service restart`, at most once per 5 minutes, and the restart is reported in facts. A
closed desktop app is reported `not_running`; the row says "Open T3 Code". A state change sends a facts
report at once.

On Linux the runner is a system service running as the person (ticket 20), so it has no login session.
T3 Code's own background service is a user service, so the restart runs `t3 service restart` with
`XDG_RUNTIME_DIR=/run/user/<uid>` (and the matching bus address) set for that user's manager, which is up
because the installer turned on lingering for T3 Code (ticket 07). If that manager is not running, the runner
reports it instead of restarting.

## Acceptance criteria

- [x] `testing/synctest` tests: two missed probes restart once; a third within 5 minutes does not; a desktop
      app is never restarted.
- [x] Manual check on Linux, with the runner installed as ticket 20 does and no one logged in: `systemctl --user
      stop` on T3 Code's unit (as that user) comes back within a minute.

## Comments

Built: `internal/runner/t3code.go` (`t3Keeper`, `restartT3`), started by `Client.Run` for a personal runner and
kept across reconnects. No migration.

- Facts gain two fields on `t3`, beside the existing ones: `restarted_at` (time, when the runner last restarted the
  service) and `restart_error` (why it could not, cleared once T3 Code answers). `make event-schemas` regenerated
  `runner.facts_reported`. They live in the runner's memory, so a runner restart drops `restarted_at`.
- A miss is a probe that finds `not_running`; `answering` resets the count and records the runtime file's
  `serviceManaged`. Only a T3 Code last seen answering under its own service is restarted, so the desktop app and a
  hand-run `t3 serve` are only reported, and a runner that starts while T3 Code is already down restarts nothing
  (systemd's `Restart=always` in T3 Code's unit covers crashes and boot). The 5-minute gap counts from the last
  attempt, failed or not.
- `t3 service restart` runs with `T3CODE_HOME` set to the runner's home (T3 Code refuses to restart a unit whose
  home differs, which keeps a throwaway T3 Code's runner off the developer's own), and on Linux with
  `XDG_RUNTIME_DIR` (from the environment, else `/run/user/<uid>`) and `DBUS_SESSION_BUS_ADDRESS` at its `bus`. No
  `bus` means no user manager: the runner reports that, with the `loginctl enable-linger` fix, and runs nothing.
- A change in what facts report wakes the connection's facts loop at once; the first probe after start does not,
  since the report on connect carries it.
- With 30-second probes and two misses, a stop is noticed 30 to 60 seconds later, plus T3 Code's few seconds to start.
- The computer row's "Open T3 Code" for a closed desktop app is the row's work in 05 or 06: `not_running` is already
  what the facts say.
- Box check (nexul-box-keep from `nexul-box/clean`, v0.3.99-beta.8 from this branch, T3 Code 0.0.45 installed by the
  computer command for `alice`, no sessions, `Linger=yes`): `systemctl --user stop t3code` as alice at 21:40:09; facts
  `not_running` at 21:40:30, the restart and a facts report with `restarted_at` at 21:41:01, `answering` at 21:41:30.
  Stopped again at 21:43:54: `not_running` reported at 21:44:00 and no restart through 21:46:01; the next probe past
  the gap, 21:46:31, restarted it. With alice's user manager stopped (`systemctl stop user@1001.service`), the second
  miss reported `restart_error` "this account's user service manager isn't running (no /run/user/1001/bus) …" and
  ran nothing; starting the manager brought T3 Code back and the next report dropped the error. Facts frames were
  read off loopback on the box. No model turns were run.
