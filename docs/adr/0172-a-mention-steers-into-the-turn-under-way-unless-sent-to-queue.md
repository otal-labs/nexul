# A mention steers into the turn under way unless sent to queue

Amends ADR 0114 and ADR 0169.

A mention sent while the Agent was busy waited in T3 Code's queue until someone pressed Steer (ADR 0169). A nudge
in the middle of a turn, such as a correction typed while a Fix with AI run works on the ticket, so sat unread
until the turn under way ended, which is when it was no longer useful. T3 Code itself made steering the default
for a follow-up and kept queueing behind a modifier.

Decision: a mention steers into the conversation's turn under way by default, and the person chooses Queue to make
it wait its place.

- **The dispatch still queues.** Every mention still goes out as its own `queue_after_active` run carrying its
  message id (ADR 0114), so Stop and the watch keep their run. Steering stays a later act on that run: when the
  harness reports the turn waiting in its queue, the pipeline calls the same Steer that ADR 0169's button calls,
  before it marks the message.
- **Only into a turn of Nexul's.** It steers only when another Nexul turn is under way on the same conversation, a
  chat turn or a play run. With none, the run ahead is one of T3 Code's own (a wake, a pull request watch, a
  schedule), whose reply Nexul never posts, so the mention stays queued and marked.
- **A refusal falls back to the mark.** When T3 Code refuses, because the queued run started or nothing is running,
  or the call fails, the message shows "Queued" with Steer beside it, as before.
- **Queue.** A message carries `queue` when its author sends it to wait: Ctrl Enter, or ⌘ Enter on Apple keyboards,
  in the web composer, holding Send in the phone app, and `queue` on the HTTP post and `message_post`.
  `chat.message.created` carries the flag beside the message; it is not stored, since it only decides the delivery
  of the turn that message starts. While the Agent works in a thread, the web composer says that Enter steers and
  the shortcut queues.
- **Nothing else changes.** An idle thread starts a turn as before, and Stop, answers and the auto-declined
  approvals of an unattended turn are untouched: only when the next message reaches the Agent moves.

The trade-off: a nudge is answered inside the reply of the turn it joined, never by a reply of its own, and the
person has to ask for that one when they want it.

Rejected: dispatching the mention as a steer, which leaves no run carrying its id (ADR 0114); steering into any
running run on the thread, which loses the reply when that run is T3 Code's own; storing the choice on the message,
which would grow the messages table for a value nothing reads after the turn starts.

Decided 2026-10-27.
