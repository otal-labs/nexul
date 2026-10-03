# 05 — Run an @Agent or play turn to its reply on protocol 2

**What to build:** `t3clientv2.StartTurn` for real (spec decisions 9–11).
- Resolve an empty `Target.Model` to the provider's default (`t3rpc`). Mint the message id first.
- No session → `thread.create` with every required key:
  `{"type":"thread.create","createdBy":"user","creationSource":"web","commandId":…,"threadId":…,"projectId":…,"title":<non-empty>,"modelSelection":{"instanceId":…,"model":…,"options":[…]},"runtimeMode":"full-access","interactionMode":"default","branch":null,"worktreePath":null}`.
- Subscribe to `orchestration.subscribeThread` with `acceptBoundedSnapshot: true` and wait for the
  first snapshot before dispatching. The reused thread is gone when the initial subscribe fails with
  `OrchestrationV2GetThreadProjectionError`, or when the snapshot's `thread.deletedAt` is set or a
  `thread.deleted` event arrives before dispatch: recreate it once with the Full prompt. Dispatch on a
  soft-deleted thread succeeds (ticket 04 Findings 1), so a dispatch failure is a turn error, not a gone
  thread.
- Ticket 04 Findings: the `{"sequence":N}` reply to `message.dispatch` can arrive before the stream's
  `run.created`, so the watch must not assume order; optional keys (`historyOrigin`, `queueHeld`,
  `delegatedCompletion`, …) are absent rather than null, and absent `historyOrigin` means native; a
  failed run can carry an `assistant_message` (Claude's "Not logged in") whose text must not become
  the reply, the root error's message is the reply; the provider session's `lastError` is often null.
- `t3rpc.Stream.Next` reports a failed Exit only as text today: add a typed exit error carrying the
  cause's `_tag` values (ticket 01 Comments). An Exit whose cause is `Interrupt` after Nexul closed its
  own stream is a normal close.
- Prompt: `Full` + attachments for a new thread, or for `historyOrigin == "v1_import"` with no run in
  status `completed`; otherwise `Incremental`.
- `thread.runtime-mode.set` to `full-access` only when the snapshot has no queued, preparing, starting,
  running or waiting run; otherwise the note "this T3 thread is not in full access".
- `message.dispatch` with `dispatchMode: {type: "queue_after_active"}` and no `deliveryIntent`.
- The watch is a pure reducer (spec Code style): a snapshot always replaces state and the cursor; events
  at or below the cursor are dropped; unknown event types are skipped but advance the cursor. The run
  whose `userMessageId` is the message id is the turn; its `assistant_message` turn items →
  cumulative `Snapshot` per message id; Terminal exactly once (first `waiting`, else `completed` →
  done; interrupted, cancelled, rolled_back → interrupted; failed → error with the root error item's
  `failure.message` (status failed, nodeId = run.rootNodeId), a usage limit naming its reset time, then
  the provider session's last error, then a generic line; error items with status `running` are
  retries).
- While the run is `queued`, or the thread's queue is held: a note re-emitted every five minutes under
  one CallID.
- After the first snapshot, any stream Exit (a `Die`, or a `Fail` of any tag) or stream end
  resubscribes with `afterSequence = cursor`, up to three times with backoff, then ends the turn with an
  error. A reconnect whose handshake reports protocol 1 ends the turn with "T3 Code was updated during
  this turn; ask again".
- Runs with any other `userMessageId` produce nothing (ticket 11 adds the one exception).
- Add the protocol-2 turn semantics to ADR 0113 (ticket 03's; if it has not landed, add a section that
  ADR will reference) and a note in `docs/adr/0106-a-follow-up-turn-on-a-live-session-carries-only-what-is-new.md`
  that an imported thread's first protocol-2 turn gets the Full prompt.

**Blocked by:** 02, 04

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, ticket 04's Findings and the fixtures in `internal/t3clientv2/testdata`, and ticket 01's Comments.

- [ ] Error paths first: dispatch Exit failure → StartTurn error; gone reused thread (subscribe failure, `deletedAt`, `thread.deleted` before dispatch) recreated exactly once with Full; a failed run's assistant text is not the reply; failure-message precedence; running error item is not a failure; three failed resubscribes → TurnError; protocol change on reconnect → TurnError
- [ ] `thread.create` frame carries every required key (`branch` and `worktreePath` present as null)
- [ ] Reducer table over fixtures: waiting → done once, later completed/checkpoint/delivery updates emit nothing; completed without waiting → done; interrupted/cancelled/rolled_back → interrupted; queued → note repeating under one CallID; a resubscribe answered with a snapshot whose run is already waiting → done once; a Die mid-stream → resubscribe; foreign runs emit nothing
- [ ] Imported thread with no completed run → Full; with a completed run → Incremental; a reused thread with a running run gets no runtime-mode.set
- [ ] `TestStartTurn_WaitsForSnapshotBeforeDispatch`, `TestStartTurn_NeverSteers`; protocol-1 tests green; `make lint`, `make coverage` green
