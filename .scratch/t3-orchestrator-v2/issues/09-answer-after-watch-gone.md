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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] `TestStartTurn_AnswerToPendingLiveRequest_SendsOnlyRespond` (frames sent are exactly one `runtime-request.respond`, `PromptSent` false)
- [ ] Expired → text dispatched; resolved → done with note; no answer path produces two runs (dispatch counter)
- [x] Chat and plays fallbacks pass the answer (a test on each)
- [ ] The protocol-1 fake still gets a plain turn and `markSent` runs
