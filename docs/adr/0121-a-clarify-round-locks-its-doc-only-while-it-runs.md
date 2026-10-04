# A Clarify round locks its doc only while it runs, and writes through its own lock

Amends ADR 0107 for one play: the built-in "Clarify via AI".

A clarification is answered over days in rounds, by the people who wrote the
doc, and its last round rewrites the doc. ADR 0107's lock never comes off, and
it refuses agents as it refuses people, so a Clarify run would lock the client
out of their own doc after the first round and be refused its own closing
write.

Decision: a Clarify run locks the doc when it starts, as every doc play does,
and the round records whether this run took the lock. Every end of the run
closes the round and, if this run took the lock, unlocks the doc; a doc
locked before the run stays locked. While the round runs, `doc_update` accepts
the doc's new body from the round's starter (the paired computer's token acts
as them) when it comes with the round's no-gaps verdict. People's edits stay
refused, because neither the editing session nor the plain update path
carries that verdict.

Every other doc play, built-in or custom, keeps ADR 0107's behaviour. The
round is the record of the lock, so docs still keep no `locked_by` column.

Rejected:
- Letting the agent unlock the doc itself, which works only when the starter
  holds `docs:lock` and depends on the agent obeying.
- Having the runner write a body the agent left on the round after unlocking,
  which stores a pending body and moves a doc write into the plays domain.
- No real lock, only a read-only page, which lets an edit made during the run
  be overwritten by the closing write.

Decided 2026-10-04.
