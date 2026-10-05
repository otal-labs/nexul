# A message typed in T3 joins the turn and shows in the thread as the person's own

People carry a run on in T3 Code itself: they answer the agent there, or tell it to continue from their phone. Under
ADR 0116 a run typed in T3 had no causal link to Nexul's run, so the turn ended on Nexul's own reply, the typed
message never reached the Nexul thread, and the trail stopped short of the work that followed.

Decision: while a turn is open, a message a person types in T3 on its thread belongs to the turn.

- **Which messages.** A user message T3 records as written by the user (`createdBy` `user`) in its web or mobile app
  (`creationSource` `web` or `mobile`). Nexul sends every message of its own with an id starting `nexul-`, and T3 makes
  an answer given in message mode `async-answer:<request>`; neither is ever a typed message, whoever T3 names as its
  author, since Nexul's dispatch also says `user` and `web`.
- **What the turn follows.** A typed message steered into a followed run shows where it landed. A run a typed message
  started after the turn's own run joins the followed set like a wake does (ADR 0116), so its steps stream into the
  trail, its reply becomes the turn's reply, and the turn stays open while it is queued or working, under the same
  60-minute cap. Only handed-off work earns the "Waiting for work handed off" step, since a typed run shows its own
  steps. Stop reaches a typed run as it reaches a wake. A run typed after the turn ended is not followed.
- **Where it shows.** The message lands in the conversation as its author's own, at the time it was written, marked
  with the harness it was written in (`messages.via`, `T3`), and the trail keeps it as a step of kind `user_message`
  between what the Agent said before and after it. Both read "via T3". Its id derives from the conversation and T3's
  item id, so relaying it again (a reconnect, a replayed snapshot) changes nothing. Its mentions are not parsed: it
  already reached the harness, and an `@Agent` in it must not start a second turn.
- **The next prompt.** A relayed message is left out of the next incremental prompt, because the harness session
  already holds it, and kept in a full prompt, which builds a new session from the conversation.

Rejected: mirroring every run of the thread, which ADR 0116 rejected for posting replies nobody asked for; and telling
typed messages apart by their text, which a long prompt makes unreliable and a short one makes ambiguous.

The trade-off: a protocol-1 computer relays nothing, and a run typed while no turn is open on the thread stays in
T3 Code.

Amends ADR 0116. Decided 2026-10-05.
