# 11 — Keep a turn open for the work it handed off

**What to build:** Spec decision 15, in the protocol-2 watch and the agent pipeline.
- Decode what carries the links: `message.updated` with role `user` and `projection.messages`
  (`delegatedCompletion.parentRunId`, `notification`), `subagent.updated` and `projection.subagents`
  (id, origin, runId, status, childThreadId, completionDelivery.state), and `run.created`/`run.updated`.
  An absent `delegatedCompletion` on `run.updated` keeps the previous value; record every
  `delegatedCompletion.delivery.messageId` seen.
- The followed set starts with Nexul's run and grows only through: a user message whose
  `delegatedCompletion.parentRunId` is followed; a run whose `userMessageId` is a recorded delivery
  message id; a user message whose `notification` decodes to `{kind: "background_task", work:
  "subagent", childThreadId: X}` with X a followed subagent's child thread; `restartContinuationOfRunId`
  in the set. A run whose message is not known yet waits in a pending map and is adopted when its
  message links it. Never followed: `message:pr-watch:` ids, `scheduledTaskId`, `senderThreadId`,
  command and monitor notifications, runs from T3's own UI.
- Pending = a followed run that is live, a subagent (either origin) of a followed run that is pending,
  running or waiting, or an `app_owned` one whose `completionDelivery.state` is `pending` or `claimed`.
  Wake runs stream into the same turn (their assistant text continues the reply); Terminal comes when
  nothing is pending.
- "Waiting for work handed off in T3 Code": a step re-emitted every five minutes under one CallID while
  anything is pending, whatever the parent run's status (wait-mode delegation keeps the parent running
  with no events).
- A linked user message that lands in a run outside the set (T3 steered it into a later turn) ends the
  wait done with the note "The handed-off result went to a later reply in T3 Code".
- Cap: 60 minutes after Nexul's own run reached `waiting` → Terminal done and a result flag the pipeline
  reads to append this exact line to the stored reply: "Part of this work is still running in T3 Code."
- Pipeline: the in-flight turn map becomes per turn (each turn gets an id; `clearActive` deletes only
  its own entry; `Interrupt` and `Answer` reach every live turn of the conversation, newest first for
  answers).
- ADR 0114 (re-check the number): a turn on T3 Code waits for the work it handed off, and its reply
  carries it.

**Blocked by:** 05, 10

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [x] Fixture-shaped reducer tests: async delegate wake joins and the turn ends on the wake reply; a queued wake whose `run.created` precedes its message is adopted; a provider-native background subagent's notification wake joins; PR-watch wake, scheduled run and another thread's send are ignored; a steered result ends with the "later reply" note
- [x] Under `testing/synctest` with channel-backed fakes: a running parent with a running app_owned subagent and no events for 40 minutes is not ended by either silence window; the waiting step reuses its CallID; the cap ends done and the stored reply ends with the exact line
- [x] Two mentions in one conversation, the second ends first: Stop still interrupts the first (fake harness records both)
- [x] A turn with no hand-offs ends exactly as in ticket 05

## Comments

- **Where it lives.** `t3clientv2/watch.go`: `followed` (Nexul's run plus the runs its hand-offs woke), `deliveries`
  (wake message id to the run that named it), `messages` (user messages carrying `delegatedCompletion` or a subagent
  notice) and `subagents` (rows by id). `link` grows the set to a fixpoint from `caused` and `linked`, `pending` says
  whether anything still runs, `steered` spots a result T3 put into a run outside the set, and `end` now returns
  `([]Update, *TurnResult)` so the steered note rides with the Terminal. `waiting` and `leave` serve the cap.
  `t3clientv2/pump.go`: `standingNote` (the queued note, else the waiting step, `handoff:<run id>`) and `capped`.
  `agent/pipeline.go`: `active` is `map[conversationID][]*activeTurn` (the pointer is the turn's id), `liveTurns`
  copies them newest first; `TurnResult.LeftRunning` makes `finishTurn` append the line.
- **Ticket 12.** `Interrupt` runs on its own connection and only knows `Harness.messages` (thread to message id); the
  followed set lives in the turn's watch, which the pump goroutine owns. Either rebuild it in `Interrupt` by applying
  `getThreadProjection` to a fresh `newWatch(messageID)` as a snapshot (the projection holds every live run and
  working subagent, but only a window of messages, so an old wake message may be missing), or publish a copy from the
  pump under a mutex. The watch already ends interrupted when Nexul's own run turns `interrupted`, even mid-wait, and
  emits nothing after its Terminal, so "no wake run is followed afterwards" holds once Stop interrupts that run.
- **Ticket 13.** `w.subagents` holds every row seen, replaced by each snapshot (a bounded snapshot keeps every
  working subagent); the children of the turn are the rows whose `runId` is in `w.followed`. `subagent` decodes only
  id, runId, origin, status, childThreadId and completionDelivery today: add driver, model, title, prompt and result.
  At the cap `leave` ends the turn with `LeftRunning`, which is where a still-running child becomes `left_running`.
- **Ticket 16.** Check live: the order of a Claude background subagent's `subagent.updated` (completed) against its
  wake's `run.created` (read from source: the adapter buffers the notice and replays it inside the wake run, so the
  row stays `running` until then; `ClaudeAdapterV2.ts` around `bufferWakeMessage`), and a wake reporting several
  subagents at once, whose notice carries no `childThreadId` (`Notification.ts:120`) and so links nothing.
- **Tests.** The silence windows themselves are not re-tested: the pump test shows the step every five minutes with
  no Terminal for 40 minutes, and `TestHandleMessageCreated_UpdatesKeepArriving_TurnRunsPastTenMinutes` (chat) and
  `TestSilence_ContinuousActivity_KeepsTheRunAlive` (plays) already show any step keeps each window open. The paused
  guard's test is a step added to `TestHandleMessageCreated_PendingQuestion_PausesTheWindowUntilAnswered`; it fails
  with the guard removed. "No hand-offs ends as in ticket 05" is the ticket 05 and 07 reducer and pump tests, unchanged.
- **Judgment calls.**
  - Only a done run waits. An interrupted or failed Nexul run ends the turn at once, as before: Stop must end the
    turn, and T3 disposes the parent's delegated cohort on interrupt.
  - The Terminal's state is Nexul's run's. A wake run adds its text and steps; its own failure is not the turn's.
  - A followed wake run is pending while queued, preparing, starting or running; at `waiting` its reply is final.
  - The waiting step is an `ActivityNote` under `handoff:<Nexul's run id>`. It shows at once when something becomes
    pending, also while Nexul's run still runs (a wait-mode delegation, a foreground subagent), then every five
    minutes; the queued note wins while Nexul's run is queued.
  - The cap's clock starts when the pump first sees Nexul's run done and survives resubscribes. The cap is checked
    before the step, so the last update is the Terminal.
  - A steered result ends the wait even if other hand-offs are still pending, as the ticket says. Steered means a
    linked message whose `runId` names a known run that is neither its own nor followed.
  - A snapshot replaces the subagent rows but keeps the link messages and the followed set, which a window can drop.
  - Only T3's wire form of a subagent notice links (`{kind: "background_task", work: "subagent"}`); T3 always encodes
    its `subagent` source that way.
  - A restart continuation joins only while the turn is still open (a wake interrupted by a T3 restart). Nexul's own
    run interrupted by a restart ends the turn interrupted, as before.
  - The line follows the reply after a blank line; with no reply text it is the whole reply.
  - `Answer` tries each live turn newest first and moves on after any refusal except `ErrConflict` (answered), then
    wakes every live turn's silence window, since the asking turn is the paused one. `Interrupt` calls every live
    turn and succeeds if any harness stopped something, else returns the first error.
  - The reducer emits nothing after its Terminal; the ticket 05 fixture's post-terminal delivery would otherwise
    adopt a wake run once the turn was over.
- **ADR.** 0114 and 0115 were taken on `origin/master`, so this is
  `docs/adr/0116-a-turn-waits-for-the-work-it-handed-off.md`; ADR 0114 points at it and the spec's Docs list says so.
