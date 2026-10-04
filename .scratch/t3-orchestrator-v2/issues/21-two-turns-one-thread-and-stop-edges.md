# 21 — Two turns on one T3 thread, and Stop at the edges

**What to build:** The server-side edge cases the reviews of tickets 11–14 left open. Re-read the code
first; line numbers drift.

1. **Two turns on one T3 thread.** `internal/t3clientv2` keeps the watching turn per thread
   (`Harness.messages` / `Harness.turns`), so when a second @Agent mention queues behind a running turn on
   the same T3 thread, the older turn's entry is overwritten. Its Stop then falls back to the thread's newest
   unfinished run (the queued one) and misses its own running run, and Stop reaches only the newer turn's
   hand-offs. Key the watch per turn (the turn's message id), so each turn's Interrupt stops its own run and
   its own hand-offs. Test: two turns on one thread, the second queued; Stop on the first interrupts the
   first's running run and cancels nothing of the second's; Stop on the second cancels its queued run.
2. **Answers wake only their own turn.** `internal/agent/pipeline.go` `wakeAll` restarts every live turn's
   silence window after any answer. Route the answer and the wake by the question's request id, so
   answering one turn's question does not restart another turn's window while its question is still open.
3. **Stop during a resubscribe.** In `internal/t3clientv2/pump.go`, `resubscribe` waits on `ctx`, not on the
   turn having been stopped, and its two Terminal sends bypass `finish`, so a Stop during the backoff ends
   late or as "Lost the connection…", and a note left by Stop is lost. Make the backoff end at once on Stop
   (Terminal interrupted) and send every Terminal through the same path that carries Stop's notes.
4. **Images the prompt names but never sends.** `internal/agent` names every `image/*` attachment as
   "attached to this turn", but protocol 2 sends only gif, jpeg, png and webp (protocol 1 has its own
   limits). Name as attached only the types the harness takes, and list the others as links the agent can
   open, so the prompt never claims an image the agent did not get.
5. **Hand-off code where the practices put it.** Move `storedHandoffs` and `cutReplies` out of
   `internal/chat/model.go` into the use-case layer (practices/architecture.md: models import nothing).
   Fix the "at most 160 runes" wording (a long title is 160 runes plus the ellipsis).
6. **Tracker.** In the spec, the phone line now says hand-off pills ship on the phone (ticket 19). Ticket 16
   gains the phone check from ticket 19's Comments: scroll a hand-off with close to 200 steps.

**Blocked by:** None — can start immediately

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, tickets 11–13's Comments.

- [x] Two-turns-one-thread Stop test passes both ways; protocol-1 behavior unchanged
- [x] Answering one turn's question leaves another turn's paused window paused (synctest, channel fakes)
- [x] Stop during a resubscribe backoff ends interrupted at once and keeps its note (synctest)
- [x] A Full prompt never names an svg or bmp as attached on protocol 2
- [x] `make lint`, `make vet`, `make coverage` green

## Comments

- **Where it lives.**
  - 1: `harness.StartResult.TurnID` names a turn and `harness.Target.TurnID` carries it back to `Interrupt`. Protocol 2
    names a turn by its message id; protocol 1 leaves it empty. `t3clientv2.Harness.turns` is keyed by `turnKey{thread,
    message}`. The pipeline sets the id in `announceTurnStarted`, the setup turn right after `StartTurn`.
  - 2: `activeTurn.asked` (under `Service.mu`) records each question's request id in `drainTurn` before the card is
    posted. `Service.answerers` puts the asking turn first, and `wake` restarts only the asking turns' windows.
  - 3: `pump.stopped` is the one "Stop halted the turn" check, used by `drain` and `resubscribe`; the backoff waits on
    `runningTurn.halted`. The protocol-change and "Lost the connection" ends go through `finish`.
  - 4: `harness.ImageTaker` is an optional `Client` capability; `t3clientv2.Harness.TakesImage` answers it from
    `supportedImages`, and `persistImages` uses the same check as its backstop. `agent.bodyImages` replaces
    `ExtractAttachments`; `targetSection` lists each image not attached as `- name: /api/attachments/<id>` under
    `linksLine`, then "The other images…" when some were attached.
  - 5: `storedHandoffs`, `cutReplies`, `jsonLen` and `maxHandoffBytes` are in `chat/usecase.go`. The "at most n runes"
    wording was `harness.Preview`'s doc.
  - 6: the spec's phone line and ticket 16's phone step.
- **Judgment calls.**
  - The pipeline's Stop still calls `Interrupt` once per live turn of the conversation, as before; each call now
    reaches its own turn. A Stop that names no turn, or one whose watch has ended, keeps the newest-unfinished
    fallback. A Stop pressed while a turn is still inside `StartTurn` has no name yet, so it takes that fallback.
  - An answer to a question no live turn raised (one an ended turn left) is still tried on every live turn, newest
    first, and wakes no window: none is paused on it. `ErrConflict` wakes nothing, as before.
  - Every Terminal goes through `finish`, so a note left by a refused Stop also shows before "Lost the connection…" and
    before the protocol-change end. A Stop that lands during a redial rather than the backoff is caught at the next
    backoff or by `drain`; on the third and last redial the turn still ends "Lost the connection…", with Stop's notes.
  - Images are named as attached only when every image in the body was attached, with the line unchanged, so a
    protocol-1 prompt with images that fit reads exactly as before. Images over the size caps (10 MiB each, 25 MiB a
    turn) are now listed by link as well instead of being silently claimed. A non-image or an attachment that cannot
    be read stays out of the list; the agent sees its link when it reads the body.
  - The markdown rewriting ("[image: name, attached to this turn]", "[attachment omitted: name]") had no reader since
    ADR 0111 (the body never enters the prompt), so it went with `AttachmentBudget`; one turn reads one body.
  - `chat/model.go` keeps the `Handoff` types and `NewHandoff`, a conversion, so it still imports `harness`, as
    `plays/model.go` does for `ActivityEntry`. Only the storage rules moved, as the ticket names.
- **Known limits.**
  - A `t3code` computer runs through `Forward`, which is not an `ImageTaker`. The one turn that moves a computer
    forward builds its prompt for protocol 1, so an svg there is still named as attached, then skipped with the
    "Not sent to T3 Code" note.
  - A play-start Incremental prompt repeats the ticket section with its images line, though neither protocol sends
    images with an Incremental prompt (the thread got them with its Full prompt). Unchanged here.
  - After a Stop cancels a queued run, `turn.end` still sees the watch's run as queued and sends a second
    `queued-run.cancel`; T3 answers "is not queued" and it is logged at Warn. Harmless, unchanged here.
- **Tests.** Each fails on the code before this change:
  - `TestInterrupt_TwoTurnsOnOneThread_EachStopsOnlyItsOwnRunAndHandOffs` (t3clientv2, the fake server; no timers).
    With turns keyed by thread, Stop on the first cancels the second's queued run and the first never ends.
  - `TestAnswer_TwoTurnsPausedOnTheirOwnQuestions_WakesOnlyTheTurnThatAsked` (agent, synctest, channel fakes). Before,
    the answer went to the newer turn and both windows woke: two no-signal posts.
  - `TestPump_StopDuringAResubscribeBackoff_EndsInterruptedAtOnceWithItsNote` (synctest). Before, nothing came until
    the backoff ran out, then "Lost the connection…" without the note. `TestPump_StreamEndsMidTurn` now leaves a note
    first and checks every end shows it; the "three failed" and "changed protocol" rows failed before.
  - `TestRunTurn_TicketImagesTheHarnessDoesNotTake_AreLinkedNotNamedAsAttached` (agent). Before, svg and bmp were
    attached and the images line claimed them. `TestBodyImages*` replace the `ExtractAttachments` tests for the new
    contract (oversized images by link, the turn's total, one read per id).
  - Existing Stop tests now name their turn; `TestWatch_RunAlreadyOver_EndsAtOnceWithItsFinalReply` counts every
    registered turn instead of loading by thread id, which the new key would have made vacuous.
- **Ticket 16.** Check live: two mentions on one T3 thread, the second queued; Stop each and confirm only its own run
  and hand-offs stop.
