# A computer can refuse auto plays, and an unfinished setup is not a computer

Extends ADR 0211 (its `plays_off`) and ADR 0165 (the run order).

`plays_off` keeps every run off a computer, so a person who wanted a laptop to stay theirs for pressed runs but out of
the unattended ones had no switch. And adding a computer saves a row before its runner enrols: an install nobody
finished left a nameless row that sat in the run order and showed in every refusal's passed-over list.

Decision:

- **`auto_plays_off`** is a second owner switch on the computer, set beside `plays_off` (`PATCH
  /api/pairing/computers/{id}`, `computer_pair`). A run an auto play started (a queue item with an `auto_play_id`) passes
  over such a computer with the reason `no_auto_plays` ("<name> takes no auto plays"); a run someone presses, an
  @Agent turn and a run an automation queued still land there. `plays_off` still passes it over for every run.
- **An unfinished setup** is a computer with no name and no session, which a runner names on enrolling. It is in no run
  order and no placement candidate, so it never appears in `passed_over` or `waiting_on`. `GET /api/pairing/computers`
  and `computer_list` still return it, marked `"unfinished": true`, so a client can offer to clear it with the
  existing delete route; the enrolment code keeps pointing at its id.
- A migration deletes the unfinished setups older than a day, with no runner row pointing at them.

Rejected: deleting unfinished setups on a timer in the server (a migration clears today's backlog and the clear action
covers new ones), and one `plays` setting with three values (the two switches are independent: `plays_off` wins).

Decided: 2026-11-27.
