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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [ ] Fixture-shaped reducer tests: async delegate wake joins and the turn ends on the wake reply; a queued wake whose `run.created` precedes its message is adopted; a provider-native background subagent's notification wake joins; PR-watch wake, scheduled run and another thread's send are ignored; a steered result ends with the "later reply" note
- [ ] Under `testing/synctest` with channel-backed fakes: a running parent with a running app_owned subagent and no events for 40 minutes is not ended by either silence window; the waiting step reuses its CallID; the cap ends done and the stored reply ends with the exact line
- [ ] Two mentions in one conversation, the second ends first: Stop still interrupts the first (fake harness records both)
- [ ] A turn with no hand-offs ends exactly as in ticket 05
