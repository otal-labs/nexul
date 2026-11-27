# An auto play can pick its computers, and the daily cap is gone

Supersedes ADR 0197's "auto plays name no computers" (its "No play and no auto play carries a computer list").
Amends ADR 0166.

ADR 0197 took the computer list off the auto play so that every run followed one order. That left an auto play unable to
say "only on a machine with a GPU" or "never on the laptop", a need that belongs to the auto play and not the play: the
same Fix with AI play may run anywhere when pressed and only on the build machine when it starts by itself. Separately,
the workspace's daily cap (auto runs per ticket per rolling day, with Resume) stopped loops that a computer's runs at
once now bounds anyway, and it paused tickets for a reason a person could not see on the Agent page.

Decision:

- **A computer rule on the auto play.** `auto_plays.computer_rule` is JSON `{"labels_in":[],"labels_out":[],"ids":[]}`,
  every part optional. A computer takes the run only when it carries at least one of `labels_in` (when set), none of
  `labels_out`, and is one of `ids` (when set). Labels are the same set a play's needs match: the owner's own labels plus
  the derived `docker` and OS ones. `ids` is accepted only when the auto play runs as a person, and each must be a
  computer that person owns or holds Run agents on.
- **Placement filters before it chooses.** The rule travels from the auto play, read at the front of the queue, to
  placement, which passes a computer it leaves out over as `rule_out` ("left out by the auto play's computer rule") and
  carries on down the person's order. A run someone presses is never filtered. A run that waits only on busy computers
  and left-out ones waits for a busy one to free up, as before.
- **The daily cap goes.** `workspaces.auto_play_daily_cap`, the limits routes, the cap check in the dispatcher, Resume
  (route, tool and `play_queue_resumes`) and the "Auto plays paused" state are removed. A computer's Runs at once is the
  only limit. The `auto_play.limits_updated` and `play.queue_resumed` topics, and the daily-cap fields of the first, stay
  in the published event contract (ADR 0044) but are never sent.

The trade-off: nothing now stops an auto play looping on one ticket other than the ticket's own conditions, its
once-within limit and the computers' room. An auto play that loops needs its conditions or once-within limit fixed
instead of a pause to lift.
