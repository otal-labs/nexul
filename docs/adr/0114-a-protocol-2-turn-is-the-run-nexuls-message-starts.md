# A protocol-2 turn is the run Nexul's message starts

On T3 Code's orchestrator V2 (orchestration protocol 2) a thread no longer has one session with one active turn.
Every message becomes a run, runs queue behind each other, and T3 starts runs of its own on the same thread: wakes
for delegated work, pull-request watches, scheduled tasks, resumes after a usage limit. Protocol 1's "the session went
idle and the reply closed" has no equivalent, so Nexul needs its own definition of where an `@Agent` or play turn
starts and ends.

Decision: Nexul mints the message id, and the turn is the run whose `userMessageId` is that id.

- **The message.** `message.dispatch` with `dispatchMode: queue_after_active` and no `deliveryIntent`, always. A
  steered or restarting dispatch merges the message into another run, so no run would carry the id and the watch
  would never end. On a busy thread the run queues; while it is queued, or held by T3 after a restart, the turn
  shows a note again every five minutes under one call id, so the callers' silence windows wait as long as T3 holds
  it. Runs with any other `userMessageId` are T3's own and produce nothing.
- **Before dispatching.** The turn subscribes with a bounded snapshot and waits for it. A reused thread that is
  missing (the first subscribe fails with `OrchestrationV2GetThreadProjectionError`) or deleted (`deletedAt` in the
  snapshot, or a `thread.deleted` with it) is recreated once. T3 still runs a message sent to a deleted thread, so a
  failed dispatch is a turn error, never a sign the thread is gone. A thread outside full access is set to it only
  when no run is queued or live, because changing the mode detaches provider sessions; otherwise the turn notes it.
- **The prompt.** The full prompt goes to a new thread, and to a thread T3 imported from protocol 1
  (`historyOrigin: v1_import`) until one of its runs completes, since T3 hands such a thread only an excerpt of its
  old history. Anything else gets the incremental prompt (ADR 0106).
- **The images.** A full prompt's images are uploaded first with `assets.persistChatAttachments`, under the thread
  and message ids, as gif, jpeg, png or webp data URLs, and `message.dispatch` carries the references T3 returns.
  Any other type is left out with a note, since T3's providers take only those four. A refused upload fails the
  turn before anything is sent, because the prompt points at images the agent would not see. An incremental prompt
  carries none: the thread already holds them.
- **The end.** The turn is done at its run's first `waiting`, or at `completed` if `waiting` was missed. T3 persists
  a finished provider turn as `waiting` and moves it to `completed` once its checkpoint is captured; the reply is
  final at `waiting`, and the checkpoint is T3's own rollback bookkeeping, which can lag or stall. `interrupted`,
  `cancelled` and `rolled_back` end it interrupted. `failed` ends it in error, and the message is the run's root error
  (an `error` item with status `failed` on the run's root node), naming the reset time of a usage limit, then the
  provider session's last error, then a generic line. A failed run can stream assistant text first (Claude's "Not
  logged in"), and that text is not the reply. Error items still `running` are retries. The terminal result is
  emitted exactly once, because a stopped run reports its end twice and delivery bookkeeping re-sends `run.updated`.
- **The stream.** A pure reducer folds it: a snapshot replaces the state and the cursor, events at or below the
  cursor are dropped, and unknown event types are skipped but still move the cursor. Once the first snapshot is in,
  any end of the stream, a defect such as `LiveStreamBufferError` included, resubscribes after the cursor, up to three
  times in a row with backoff, and then the turn ends in error. A reconnect that finds T3 Code on another protocol
  ends the turn at once: "T3 Code was updated during this turn; ask again".
- **Stop.** Stop acts on the run of the turn watching the thread. Only when no turn is watching, or T3 has no run for
  its message, does it fall back to the thread's newest unfinished run. A queued run is cancelled. A live one gets
  `run.interrupt` without `holdQueue`: T3's own Stop sends it, and it would hold every later Nexul message until
  someone resumes the queue in T3. A `waiting` run has already replied; only the thread's latest one can still have
  background work to stop, and T3 calling it not interruptible means nothing is left. With nothing to stop, Stop is a
  conflict. A turn that stops watching while its run is still queued cancels that run, since nobody would read its
  reply.
- **Answers.** A question is answered with `runtime-request.respond`, keyed by question id. A request T3 resumes by
  dispatching the answer as a message of its own (`responseCapability` `message`) takes one non-empty string per
  question, with multi-select choices joined by ", ".
- **An answer after the turn ended.** The next turn carries it, and reads the request in its snapshot before sending
  anything. A request still pending gets `runtime-request.respond`, not a message: a live question keeps its run
  `running`, and a message would queue behind it forever. That turn then follows the run the question held open, or
  for a `message` request the `async-answer:<requestId>` run T3 starts, or the live run T3 steered the answer into;
  it sends no prompt, so the conversation's unsent messages wait for the next one. A request T3 already resolved ends
  the turn with "Already answered in T3 Code". An expired, cancelled, unknown or unresumable one goes as an ordinary
  message. T3's refusal of the respond decides only when the request changed after the snapshot.

Rejected: ending the turn at `completed`, which adds checkpoint latency to every reply and hangs when the capture
target is incomplete; following every run on the thread, which would reply to T3's own wakes, watches and schedules
as if Nexul had asked; and treating a failed dispatch as a missing thread, which the captures disproved.

A turn that hands work to another agent waits for it past `waiting`; that is a later decision, not this one.

Decided 2026-10-03.
