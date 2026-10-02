# A follow-up turn on a live session carries only what is new

Supersedes ADR 0065 in part: the interview memory and the other always-included memories reach every session in
full, not every turn.

An `@Agent` mention that starts a harness session sends the full prompt: the instructions, the memories index, the
always-included memories, the ticket or doc, the conversation so far, and the images those bodies embed. A later
mention in the same conversation runs on the same session, which already holds all of that, so it sends only the
messages posted since the last prompt, without the Agent's own replies, and the request. The full prompt goes again
only when the harness creates a session, because the conversation has none yet or the stored one is gone.

The owner saw every follow-up in a ticket thread repeat the memories, and a mention made while the previous turn was
still running repeat that turn's messages too. The always-included memories were deliberately re-sent on every turn
under ADR 0065, so a cheap model would not lose the rules; within one session the rules are already in its context,
and repeating them costs a large prompt per message for nothing the model did not have.

How far a conversation has been sent is its cursor, the time of the newest message the last prompt carried. It moves
when the harness accepts the prompt, not when the turn ends, so messages posted during a turn are not skipped and a
mention mid-turn does not repeat them. A fresh session after a lost one still gets only the messages after the
cursor, including the Agent's replies it never saw; it does not get the conversation's older history back.

Rejected: re-sending the always-included memories on every turn of a long session in case the model drifts. If that
shows up in practice, the answer is a fresh session, not a heavier follow-up.

Decided 2026-10-02.
