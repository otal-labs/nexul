# A queued mention can be steered into the turn under way

Amends ADR 0114. Amended by ADR 0172: a mention now steers by default, and the mark and its Steer show only for one
sent to queue or one T3 Code would not take.

A mention sent while the Agent is busy queues in T3 Code behind the run under way (ADR 0114), and the thread showed it
as sent: nothing said the Agent had not read it, and the only way to reach the Agent sooner was T3 Code's own Steer.

Decision: the thread marks a mention whose turn waits in the queue, and its **Steer** hands the message to the run
under way, whose reply then answers both.

- **The mark.** A harness update says when a turn starts or stops waiting in the queue. The pipeline passes it on as a
  `chat.agent.stream` frame naming the request message, which carries nothing else and leaves the bubble alone. The
  web app shows "Queued" under that message with Steer beside it; the phone app has no live stream and shows nothing.
- **Steer.** `POST /api/agent/conversations/{id}/steer` with the message id, for anyone who can read the
  conversation, like Stop. On `t3code-v2` it sends `queued-message.promote-to-steer` with the queued run and the
  thread's running run. T3 Code refuses when the run has started or nothing is running, and the refusal comes back as
  a conflict. Protocol 1 never queues, so it always refuses.
- **The steered turn ends done, with no reply of its own.** T3 Code cancels the queued run, which would end the turn
  interrupted; the turn knows it was steered, so it ends done instead, and the bubble of the turn under way stays.
  The dispatch still always queues: steering is a later act on a run that already carries the message's id.

The trade-off: one reply per mention gives way, since a steered mention is answered inside another turn's reply.

Rejected: dispatching every mention as a steer, which leaves no run carrying its id (ADR 0114); holding a mention in
Nexul until the Agent is free, which hides it from T3 Code's own queue and from Stop.

Not covered: a Steer pressed in T3 Code's own app still ends the Nexul turn interrupted, because the cancelled run
arrives before the message lands in the run that took it.

Decided 2026-10-24.
