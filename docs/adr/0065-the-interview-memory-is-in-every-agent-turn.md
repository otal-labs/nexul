# The interview memory is in every agent turn

Superseded in part by ADR 0105: a play run names the interview memory for the agent to read first instead of
carrying it in full; an `@Agent` mention still carries it in full. Superseded in part by ADR 0106: a follow-up
mention on a live session does not repeat it, the session already holds it.

A project's interview memory holds its stack, paradigm, testing strategy,
principles, and vocabulary, written as rules. It is included in full in
every agent turn in that project, every play and every `@Agent` mention, and
cannot be switched off per play. The alternative, pulling it from the memory
index when an agent thinks it relevant, costs nothing per turn but relies on
the agent to know it needs the rules, which is exactly what a cheaper model
gets wrong. Carrying the rules every time is what lets a cheap model write
code the team can trust.

The cost is paid on every turn, so the memory is capped in length and the
agent that writes it must keep it to rules, not a transcript. The decisions
log, which grows with the project, is kept out of the every-turn slot and
pulled from the index instead. A project without an interview is not
blocked; it gets an "are you sure?" when the interview is skipped and a
banner until one exists.
