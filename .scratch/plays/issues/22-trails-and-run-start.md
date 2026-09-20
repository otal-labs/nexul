# 22 — Trails and the run start

**What to build:** Running a play through MCP or the gateway creates a trail and starts the turn. The trail is created at the call in state `starting`; the request is composed from the play's name and base instructions, the selected memories inlined in full (always-included ones first, refused up front when over the per-run ceiling), and the caller's custom instructions, all inside the request block; the caller's "Started <play>" message is posted in the target's thread and is the request; the turn runs on the caller's harness through the use-case from ticket 13; the harness activity stream is captured on the trail, capped; the trail reaches `running` and then `done`, `failed`, or `interrupted` with the Agent's reply message id. One active trail per target; a second run is refused with "a run is in progress". `play_run`, `play_run_get`, `play_list_runs` and their routes exist. Outcomes (move-to, events, notifications, timeout, Stop) are ticket 23.

**Blocked by:** 13, 16, 20

**Status:** done

- [ ] `play_run` on a ticket in the play's show-when stage creates a trail, posts the Started message, and the fake harness receives a prompt whose request block contains the play instructions, the inlined memories, and the custom instructions in that order
- [ ] A harness that refuses to start leaves a `failed` trail with the reason; a caller without a ready harness gets the readiness reason
- [ ] The trail records play, target, starter, selected memories, custom instructions, chosen move-to, session id, timestamps, outcome, reply id, and the activity lines
- [ ] Over-ceiling memory selections are refused with the totals; a second run on the same target while one is active is refused
- [ ] A play excluded from the project, disabled, or denied to the caller cannot be run
