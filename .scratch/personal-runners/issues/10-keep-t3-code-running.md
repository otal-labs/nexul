# 10 — Keep T3 Code running

**Status:** ready-for-agent

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

- [ ] `testing/synctest` tests: two missed probes restart once; a third within 5 minutes does not; a desktop
      app is never restarted.
- [ ] Manual check on Linux, with the runner installed as ticket 20 does and no one logged in: `systemctl --user
      stop` on T3 Code's unit (as that user) comes back within a minute.
