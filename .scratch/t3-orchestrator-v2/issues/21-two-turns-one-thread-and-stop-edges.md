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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, tickets 11–13's Comments.

- [ ] Two-turns-one-thread Stop test passes both ways; protocol-1 behavior unchanged
- [ ] Answering one turn's question leaves another turn's paused window paused (synctest, channel fakes)
- [ ] Stop during a resubscribe backoff ends interrupted at once and keeps its note (synctest)
- [ ] A Full prompt never names an svg or bmp as attached on protocol 2
- [ ] `make lint`, `make vet`, `make coverage` green
