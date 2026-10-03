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

- [ ] With `testing/synctest` and channel-backed fakes (`harnesstest`, never the WebSocket fake): chat updates keep arriving past ten minutes → not cut; 15 minutes of silence → the no-signal message; a chat turn with a pending question is not ended at 15 minutes
- [ ] A play turn waiting on a question for 45 minutes is not ended by the pipeline and posts nothing
- [ ] `make lint`, `make coverage` green
