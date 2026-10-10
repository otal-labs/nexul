# 10 — Keep T3 Code running

**Status:** ready-for-agent

**Blocked by:** 03

Read first: `practices/go.md` (section 7), `practices/testing.md` (section 9), the spec (Finding T3 Code).

## What to build

The runner probes T3 Code every 30 seconds. A T3 Code its background service runs that misses two probes
is restarted with `t3 service restart`, at most once per 5 minutes, and the restart is reported in facts. A
closed desktop app is reported `not_running`; the row says "Open T3 Code". A state change sends a facts
report at once.

## Acceptance criteria

- [ ] `testing/synctest` tests: two missed probes restart once; a third within 5 minutes does not; a desktop
      app is never restarted.
- [ ] Manual check on Linux: `systemctl --user stop` on T3 Code's unit comes back within a minute.
