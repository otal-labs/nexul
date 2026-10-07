# News on a harness thread brings Nexul back to it

Nexul followed a T3 thread only while one of its own turns was live. Once a turn ended, or gave up after a lost
connection, nothing watched the thread again: a run that carried on in T3 Code, or a person telling the agent to
continue there, never reached the trail or the conversation. T3 Code's own apps stay in sync by keeping
`orchestration.subscribeShell` open, which pushes a row per thread whenever its runs change.

Decision: Nexul keeps the shell open on every computer it holds, and follows a thread again when it has news.

- **Where it listens.** The presence connection Nexul already holds to a paired computer while its owner has Nexul
  open (a browser tab or the phone app) subscribes to the shell. A harness that can report sessions implements
  `harness.SessionWatcher`; presence uses it in place of `Hold`, so no second connection opens. Each thread is
  reported only when its newest run, whether a run is working, or its deletion changes.
- **What counts as news.** Every turn that hears its end from the harness records a marker on its conversation
  (`conversations.agent_seen`): the newest run it followed that T3 finished with. A turn that lost the harness
  records none. A thread linked to a conversation has news when no turn here runs on it and either a run is working
  (`activityRunStatus` preparing, starting or running) or its newest run is not the marker. The newest run counts by
  T3's own rule for the run that ran last: one still queued, or one cancelled before it started (a queued message T3
  steered into the run going instead), never ran and is no news, since a catch-up's snapshot leaves it out and could
  never move the marker to it. A conversation from
  before markers takes the thread's newest run as seen and follows only work under way, so its past never replays.
- **How it catches up.** A `Watch` with `Since` set to the marker adopts the first run after it, or the marker's own
  run while it still runs, and follows on from there with the runs typed in T3 and the work handed off (ADR 0116,
  ADR 0126). It shows the snapshot's steps, which a restart's watch leaves out; each step replaces one the trail
  already has by its call id, and the Agent's text is named by its message and offset, so a replay never duplicates.
- **Who takes it.** When the conversation's newest trail ran on that thread, as the computer's owner and on that
  computer, and has ended, or waits on a question whose turn has closed, its trail reopens as `running` with the note
  "New activity on this run's thread in T3 Code; following it again." and ends with the harness's outcome, announcing
  `play.run_finished` again. The catch-up reports a question answered in T3 Code with T3's recorded answer, which the
  waiting trail takes as its own. Otherwise a plain turn follows it as the computer's owner and posts the reply in the
  thread. One catch-up runs per conversation at a time, and none while a turn runs there.

Rejected: a "Follow again" button, which only helps once a person notices the trail stopped; polling every linked
thread, which T3's push already makes unnecessary; and holding every computer at all times, which would show Nexul as
connected in T3 Code even when nobody has Nexul open.

The trade-off: with Nexul closed everywhere nothing listens, so the catch-up happens the next time its owner opens
Nexul. A trail that ended done can move again when work continues on its thread. A protocol-1 computer is only held.

Amends ADR 0119 (a restart is no longer the only way back to a thread). Decided 2026-10-05.
