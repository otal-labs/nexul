# 09 — Deliver an answer when the turn's watch is gone

**What to build:** Spec decision 14.
- Already in place: `harness.PendingAnswer`, `TurnPrompts.Answer`, `agent.TurnRequest.Answer`, and
  both fallbacks (chat and plays) passing it.
- `harness`: `StartResult.PromptSent bool`. `RunTurn` calls `markSent` only when `PromptSent` is true, so messages posted between the question
  and the answer still reach a later Incremental prompt.
- `t3clientv2.StartTurn` with `Answer` set: subscribe, read `projection.runtimeRequests` for the
  request id. Resolved → Terminal done with the note "Already answered in T3 Code". Pending with
  `responseCapability` live or message → `runtime-request.respond`, then follow the request's run
  (live), the `async-answer:<requestId>` run if one appears (message), or the live run the answer was
  steered into. Not found, expired, cancelled or not resumable → dispatch the answer text as today. Use
  the respond error's text only as a fallback when the request changed between snapshot and send.
- `t3client` (protocol 1) already dismisses the question before the prompt and ends with the note when T3
  holds an answer; it sets `PromptSent: true`.

**Blocked by:** 08

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [x] `TestStartTurn_AnswerToPendingLiveRequest_SendsOnlyRespond` (frames sent are exactly one `runtime-request.respond`, `PromptSent` false)
- [x] Expired → text dispatched; resolved → done with note; no answer path produces two runs (dispatch counter)
- [x] Chat and plays fallbacks pass the answer (a test on each)
- [x] The protocol-1 fake still gets a plain turn and `markSent` runs

## Comments

- **Where it lives.** `answer.go`: `turn.deliver` reads the request in `turn.snapshot` (the first snapshot `subscribe`
  saw) and responds, `answeredTurn` is the note-and-done result, `projection.request` and `projection.runOf` look
  things up. `turn.start` calls `deliver` between the snapshot and `dispatch`, only on a reused thread.
  `StartResult.PromptSent` is `turn.prompted`, set once `message.dispatch` succeeds.
- **The watch can change what it follows.** `watch.runs` now holds whole runs. `watch.follow(messageID)` rekeys the
  watch: a live answer follows the asking run's own `userMessageId`, a `message` answer `async-answer:<requestId>`.
  `track` keeps updating a followed run by id once it is known, and `item` takes a `user_message` item carrying the
  message id as naming the run it was steered into. Ticket 07's mapper plugs in after that check; ticket 11's causal
  links can reuse `follow` and the by-id tracking.
- **Stop after an answer.** `turn.messageID` is rekeyed with the watch, so `Harness.messages` holds the asking run's
  message id (live) or `async-answer:<requestId>` (queued). A steered answer has no run of its own, so Stop falls back
  to the newest unfinished run, which is the run it was steered into.
- **Judgment calls.**
  - The request's run comes from the snapshot's `nodes` (`request.nodeId` to `node.runId`): T3 always keeps a pending
    request's node in a bounded snapshot, while turn items are only the recent window.
  - A respond refused with "Runtime request <id> is resolved." ends with the note; "is expired.", "is cancelled." or
    "was not found." sends the message instead. Any other refusal fails the turn with T3's message, including a
    `not_resumable` that appeared after the snapshot, since T3 words that reason freely.
  - `PromptSent` is false both after a respond and when T3 already answered; neither sent the conversation's messages.
  - Protocol 1 sets `PromptSent: true` on every result, its already-answered one included, so its cursor moves exactly
    as before.
  - A new or recreated thread never responds: the old question is not on it, so it gets the Full prompt as today.
  - A live answer's turn picks the asking run up from the respond on; that run's items already in the snapshot are not
    sent again. A resubscribe answered with a snapshot (a gap past 128 events) re-emits its earlier assistant text.
- **ADR.** ADR 0114 gained an "An answer after the turn ended" bullet.
