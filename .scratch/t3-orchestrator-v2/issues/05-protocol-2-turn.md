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

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, ticket 04's Findings and the fixtures in `internal/t3clientv2/testdata`, and ticket 01's Comments.

- [x] Error paths first: dispatch Exit failure → StartTurn error; gone reused thread (subscribe failure, `deletedAt`, `thread.deleted` before dispatch) recreated exactly once with Full; a failed run's assistant text is not the reply; failure-message precedence; running error item is not a failure; three failed resubscribes → TurnError; protocol change on reconnect → TurnError
- [x] `thread.create` frame carries every required key (`branch` and `worktreePath` present as null)
- [x] Reducer table over fixtures: waiting → done once, later completed/checkpoint/delivery updates emit nothing; completed without waiting → done; interrupted/cancelled/rolled_back → interrupted; queued → note repeating under one CallID; a resubscribe answered with a snapshot whose run is already waiting → done once; a Die mid-stream → resubscribe; foreign runs emit nothing
- [x] Imported thread with no completed run → Full; with a completed run → Incremental; a reused thread with a running run gets no runtime-mode.set
- [x] `TestStartTurn_WaitsForSnapshotBeforeDispatch`, `TestStartTurn_NeverSteers`; protocol-1 tests green; `make lint`, `make coverage` green

## Comments

- **Where it lives.** `turn.go` is StartTurn (create, subscribe to the first snapshot, runtime mode, dispatch) and the
  turn's connection; `watch.go` is the pure reducer; `pump.go` is the stream loop (the queued note's timer and the
  resubscribes). The spec's Structure put the reducer in `turn.go`; it is its own file to keep the I/O out of it.
- **`t3rpc.ExitError`** `{Method, Causes []ExitCause}` with `Failed(errorTag)`. Its `Error()` text is the old one and it
  still matches `ErrInvalid`, so protocol 1 sees no change. An Interrupt exit after Nexul closes its own stream never
  surfaces: `Stream.Close` unregisters before it sends the Interrupt. Any stream end the pump does read, an Interrupt
  exit included, resubscribes, so it needs no case of its own.
- **`refused(what, err)`** in `turn.go` turns a failed command into `invalid: <what>: <T3's message>`. Ticket 08 should
  use it for `queued-run.cancel`, `run.interrupt` and `runtime-request.respond`.
- **`t3rpctest.Server.CommandCauses`** fails only the command types it names, beside `DispatchCause` for all of them.
- **Images (ticket 06).** `message.dispatch` always sends `attachments: []`; a Full prompt's images are dropped until
  06 persists them in `turn.dispatch`, before the `message.dispatch` call. `Attachments` is `[]json.RawMessage`, so the
  persisted references go in verbatim.
- **Stop and answer (ticket 08).** `Interrupt` and `Answer` still return `ErrInvalid` (reworded to "stopping or
  answering a turn on this T3 Code version needs a newer Nexul"), so the setup turn's interrupt-on-question fails on
  `t3code-v2` until 08. The turn's run is `watch.run`, its message id `turn.messageID`. A turn whose context ends while
  its run is queued does not cancel it yet.
- **Pending answers (ticket 09).** `StartTurn` ignores `TurnPrompts.Answer`, which landed with #355, and sends the
  answer text as an ordinary message. Ticket 09 reads `runtimeRequests` from the snapshot `turn.subscribe` already
  waits for, before `turn.dispatch`.
- **Items (ticket 07).** `watch.item` receives the turn's own run's items only. Today it maps `assistant_message` to a
  `Snapshot` and records failed `error` items; the mapper plugs in there.
- **Hand-offs (ticket 11).** `watch.end` decides the turn is over at the run's terminal status, and the pump stops at
  the first Terminal. Ticket 11 extends `end` and keeps the pump reading past `waiting`.
- **Model (ticket 18).** `create` resolves an empty model to the provider's default; `message.dispatch` sends no
  `modelSelection`.
- **Judgment calls.**
  - Resubscribe backoff is 1s, 5s, 25s. The count resets once a resubscribed stream delivers anything, so it is three
    in a row, not three per turn. A dropped connection is a stream end too: `turn.open` redials, and a refused redial
    counts as a try. Only `ErrProtocol` ends the turn at once, with "T3 Code was updated during this turn; ask again".
  - The queued note is "Queued in T3 Code behind another run on this thread", or "Held in T3 Code's queue; resume it
    in T3 Code or stop it here" when `queueHeld`, under call id `queue:<runId>`. It shows at once, again every five
    minutes, and at once when it changes.
  - A usage limit reads "<message> (the limit resets at 2006-01-02 15:04 UTC)", taking the time from the root error,
    else from `thread.limitRecovery` when it names the same run. With no root error and no session error the line is
    "T3 Code reported the run failed without saying why."
  - "Imported with no completed run" looks at the bounded snapshot's runs: every live run plus the recent window. A
    completed run older than the window counts as none, so such a thread gets the Full prompt, the safe side.
  - A failed `thread.runtime-mode.set` fails StartTurn. The busy note reads "This T3 thread is not in full access".
  - An empty title becomes "Nexul chat", since `thread.create` refuses an empty one.
  - A reused thread found gone is logged at Info and recreated once; a replacement that is gone too fails the turn.
- **ADR.** ADR 0113 was not on `origin/master` and ticket 03 is writing it, so the protocol-2 turn is its own ADR,
  `docs/adr/0114-a-protocol-2-turn-is-the-run-nexuls-message-starts.md`. Ticket 03's 0113 should reference it, and
  ticket 11's ADR moves to the next free number. ADR 0106 carries the imported-thread amendment.
- **Fixture from source.** `run-waits-then-completes.source-ce90eec1ff.ndjson` is built by hand from T3 at
  `ce90eec1ff` (RunExecutionService's finalization, CheckpointCaptureService, ProviderFailure's retry item), with
  Codex-style streamed text: waiting, then the checkpoint, completed, a delivery `run.updated`, and another run. Ticket
  16 should replace it with a capture from a logged-in provider.
