# A restart follows running play runs again

A play run's turn is watched only by the server process that started it. An instance upgrade restarts that process,
the turn carries on in the harness, and nothing in Nexul hears how it ends: the trail stays `running` and the ticket
card counts forever.

Decision: at boot, before the HTTP server accepts runs, every `running` trail is followed again. `harness.Client`
gains `Watch`, which subscribes to the trail's harness thread and streams the turn already in flight without sending
anything; a thread with nothing in flight ends at once with its last outcome and reply. The pipeline drains it like a
started turn, so the trail records the real steps, reply and outcome, and Stop and answers work as before.

- A `starting` trail has no harness thread yet, so it ends `interrupted` ("Nexul restarted before the run started").
- A `waiting` trail is left alone: its answer already resumes the run as a fresh turn on the same thread.
- A thread the harness refuses to follow (computer offline, thread deleted) ends the trail `failed` with the reason,
  and the thread says the reconnect failed.
- Steps the harness ran while Nexul was down are not replayed onto the trail; the reply and outcome are.
- Chat `@Agent` turns are not followed again: they keep no record of a turn in flight to resume from.
- Nexul keeps no id for the message it sent, so Watch follows the thread's newest turn (on protocol 2, the run whose
  hand-off led to it). A message typed into the T3 thread during the restart would be the turn followed.
- On protocol 1 a question already pending at the restart is history to the watch: the run follows on silently and
  the silence window ends it if nobody answers in T3.
