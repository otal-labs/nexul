# 09 — Deliver an answer when the turn's watch is gone

**What to build:** Spec decision 14.
- `harness`: `PendingAnswer{RequestID string; Answer QuestionAnswer}`, `TurnPrompts.Answer
  *PendingAnswer`, and `StartResult.PromptSent bool`.
- `agent.TurnRequest` gains `Answer *harness.PendingAnswer`; `buildTurnPrompts` copies it into
  `TurnPrompts.Answer`. `AnswerFromChat`'s fallback sets `{RequestID: requestID, Answer: answer}`;
  plays `Runner.Answer`'s fallback sets `{RequestID: trail.Question.RequestID, Answer: answer}`.
  `RunTurn` calls `markSent` only when `PromptSent` is true, so messages posted between the question
  and the answer still reach a later Incremental prompt.
- `t3clientv2.StartTurn` with `Answer` set: subscribe, read `projection.runtimeRequests` for the
  request id. Resolved → Terminal done with the note "Already answered in T3 Code". Pending with
  `responseCapability` live or message → `runtime-request.respond`, then follow the request's run
  (live), the `async-answer:<requestId>` run if one appears (message), or the live run the answer was
  steered into. Not found, expired, cancelled or not resumable → dispatch the answer text as today. Use
  the respond error's text only as a fallback when the request changed between snapshot and send.
- `t3client` (protocol 1) ignores `Answer` and sets `PromptSent: true`.

**Blocked by:** 08

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] `TestStartTurn_AnswerToPendingLiveRequest_SendsOnlyRespond` (frames sent are exactly one `runtime-request.respond`, `PromptSent` false)
- [ ] Expired → text dispatched; resolved → done with note; no answer path produces two runs (dispatch counter)
- [ ] Chat and plays fallbacks pass the answer (a test on each); the protocol-1 fake still gets a plain turn and `markSent` runs
