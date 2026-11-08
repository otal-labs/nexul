# An auto play may sweep its tickets on a schedule

Amends ADR 0132.

Every auto play waited for something to happen to one ticket, doc or stack. Teams want the other shape too: "overnight,
take every hands-off ticket in this project and leave anything that needs me". Done by hand, a night like that needed
the shared machine guarded from a second agent, tickets already running left alone, a stop before the morning, and a
written summary at the end.

Decision: a ticket play's auto play may wait for a **schedule** moment, a five-field cron expression read in an IANA
timezone, at most once an hour. Each firing is a **sweep**: every ticket of the workspace that the play takes and the
auto play's conditions hold for, highest priority first and board order within a level, is queued as one batch, up to
the schedule's cap (1 to 100, default 10). The queue, its re-check at the front, the daily cap per ticket and the person
a run lands on are the ones every auto play uses.

- **Needs a human is stop-and-ask.** Every matching ticket is tried; there is no hands-off label and no triage pass. A
  run that meets a decision it cannot make hands the ticket back with a question, as the play's instructions say.
- **No double runs.** A ticket with a run going or waiting, or an item already queued, is skipped by the sweep with a
  line saying so. While a sweep's run is starting or working on a ticket, other auto plays' moments on it are skipped
  ("a scheduled sweep was working on it"), so the run moving its card does not set off the stage's own auto play.
- **A run window.** A schedule may name a stop time; a sweep starts nothing after the first such time after it fired,
  and skips what is left.
- **One at a time.** Any auto play may let one of its runs go at a time, or, naming a resource such as `windows-vm`, one
  at a time across every auto play that names it. A held run waits as "one at a time". A run stopped on a question lets
  go, since it is not using the resource.
- **Setup and teardown are automations.** A sweep publishes `auto_play.sweep_started` and `auto_play.sweep_finished`; an
  automation starts and stops what the runs need. Runs queued while a machine boots wait for its computer to come
  online, as any run does.
- **The digest is a notification.** When nothing of a sweep waits or works any more, it is stamped finished with its
  counts, and the people its runs went out as get one notification that opens the sweep: each ticket, whether it
  finished, asked, failed or was skipped and why, where it stands now, and the bugs found in it during the sweep.
- **Firing.** A sweep's id comes from its auto play and minute, so a firing sweeps once. Editing the auto play starts
  its schedule over from the edit. After downtime only the latest missed firing sweeps, and only while it can still
  start something: before its stop time, or within the hour when it names none.
- **Agents set it with `play_update`.** The schedule, one at a time and the resource are fields of `add_auto_plays` and
  `update_auto_plays`, no new tool; the advertised tool size rises from 188,000 to 189,500 characters for them.

Rejected: a hands-off label or a triage pass to decide what a sweep may take, when asking at the point of doubt already sorts them;
setup and teardown plays run before and after, which spend an agent run to start a machine; a doc or chat note per
sweep for the digest, which piles up where people work; schedules on doc and stack plays, which nobody has asked for.

Decided 2026-11-08.
