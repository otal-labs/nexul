# An ended run continues on its own harness thread

A run that stopped short (a lost connection, a question answered elsewhere, a result that needs one more step) could
only be carried on in Nexul by pressing the play again. That starts a new trail and sends the play's full
instructions into the same harness thread, so the agent began the whole job over instead of taking the next step.

Decision: an ended trail takes a message that continues it.

- **What is sent.** Only the message, as the next turn on the trail's own harness thread, through the trail's
  computer, provider, and model and as its starter, like an answer that resumes a run. The play's instructions,
  links, and memories are not sent again: the thread already holds them. The message is posted in the target's
  thread as its sender's, and the trail shows it as a bubble.
- **Which trail.** The same one. It goes back to `running` and ends with the harness's outcome, announcing no second
  start. Continue is for an ended trail; a running or waiting one is stopped or answered instead, and a target with
  another run in progress refuses it.
- **A gone thread.** The turn asks the harness to keep the thread (`harness.Target.KeepSession`) rather than start a
  new one silently. When the thread was deleted in T3 Code, or the conversation never had one, the trail goes back to
  how it ended with the note "This run's thread is gone from T3 Code; the play started again in a new run.", and the
  play starts again on the same target with the trail's choices, its instructions followed by the message. Continue
  answers with that new trail. The new run passes the play's own checks again, so a ticket that moved stage since can
  refuse it.
- **Adapters.** `POST /api/plays/runs/{id}/continue` with `{message}`, `trail_update` with `continue`, and a "Continue
  this run" box under an ended trail's transcript. Continue waits up to 30 seconds to learn whether the harness took
  the turn, then answers with the trail as it is.

Rejected: a Continue that presses the play again with the message as extra instructions, which is the redo this
replaces; and creating a fresh thread when the old one is gone without saying so, which hides that the agent lost
everything it had done.

The trade-off: on a gone thread the message is posted twice in the target's thread, once as sent and once inside the
new run's start.

Builds on ADR 0055 and ADR 0127. Decided 2026-10-05.
