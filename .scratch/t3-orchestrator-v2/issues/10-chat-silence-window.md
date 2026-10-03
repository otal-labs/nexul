# 10 — Chat replies end on silence, not a fixed ten minutes

**What to build:** Add `Silence time.Duration` to `agent.TurnRequest`. `HandleMessageCreated` and the
`AnswerFromChat` fallback set it to 15 minutes; plays leave it zero and keep their own timer. When it is
non-zero, the pipeline drops the fixed `maxTurnDuration` context deadline and instead resets a timer
on every update; when it fires, post the existing "no completion signal" message. While a Question is
pending the window is paused, as plays' `resetSilence` does; a pending question never posts the
no-signal failure. This lets ticket 11 wait for handed-off work without being cut off.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, the spec.

- [x] With `testing/synctest` and channel-backed fakes (`harnesstest`, never the WebSocket fake): chat updates keep arriving past ten minutes → not cut; 15 minutes of silence → the no-signal message; a chat turn with a pending question is not ended at 15 minutes
- [x] A play turn waiting on a question for 45 minutes is not ended by the pipeline and posts nothing
- [x] `make lint`, `make coverage` green

## Comments

- The window lives in `internal/agent/pipeline.go` as `silenceWindow`. Every update restarts it: snapshot, activity,
  approval or terminal. A Question stops it. It runs again only when `Service.Answer` has delivered an answer to that
  turn, as plays do. Updates that arrive while a question is pending do not restart it, so ticket 11's five-minute
  "waiting" step cannot restart the window under a pending question.
- The answer wake is `activeTurn.answered`, a 1-buffered channel. When ticket 11 keys the active map per turn, the
  channel moves with the entry and needs nothing else.
- Chat callers go through `runChatTurn`, which sets `Silence` and cancels the turn context on return. With the fixed
  deadline gone, that cancel is what releases the harness stream after the window ends a turn.
- `maxTurnDuration` is deleted. A cancelled context now ends a turn with "turn cancelled: <cause>". The no-signal
  text reads "after 15m0s of silence".
- Setup gets the same window. Target resolution, the version probe and `StartTurn` run before the stream exists, and
  their HTTP calls have no timeout, so `setupWindow` cancels them after 15 minutes. The turn then fails with "the
  harness did not answer within 15m0s". A play's setup is already bounded: its timer is armed before `RunTurn`, and
  `onSilence` cancels the run.
