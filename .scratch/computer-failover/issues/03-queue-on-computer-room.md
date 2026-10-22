# 03 — The queue waits on computer room, not a person's slots

**Status:** ready-for-agent
**Blocked by:** 02

Spec: `../spec.md`, Callers (queued auto play, waking), Surfaces (events, MCP). ADR 0166.

The dispatcher resolves each due item through 02 with room counting on `play_trails.computer_id`; nothing ready holds
it as `no_computer_ready` with `computers` beside the reason. A person's pass no longer stops at one offline computer.
Remove the workspace runs at once (panel line, route field, `workspace_update`/`workspace_list`), `PickComputer` and its
route, and `trail_update`'s pick fields. Wake on order, runs at once, project list, checkout, clone switch and runner
connect. `make event-schemas`, `make permission-catalog`.

Done when: two computers at runs at once 2 start four queued runs, the fifth waits naming both as full, and taking one
computer offline sends the next run to the other.
