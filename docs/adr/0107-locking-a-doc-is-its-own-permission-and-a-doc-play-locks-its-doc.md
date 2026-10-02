# Locking a doc is its own permission, and a doc play locks its doc

Supersedes ADR 0092 in part: locking and unlocking now take a permission bit of their own instead of `docs:write`.

ADR 0092 let anyone who could write a doc lock or unlock it, so editing and locking could not be granted apart. The
owner wants them apart, and wants a doc a play runs on to be locked from the run's start, so it stops changing under
the run.

Decision: `docs:lock` ("Lock and unlock docs") is a verb on the docs domain beside `docs:thread` and `docs:clone`
(ADR 0057), so the role editor shows it as one more toggle on the Docs row. The docs use-case checks it on the doc
for locking and unlocking, so the row menu, the doc's own menu and its Unlock button, `POST /api/docs/{id}/lock` and
`/unlock` (a scoped token needs `docs:lock` for both), and `locked` on `doc_update` all answer the same. What a locked
doc refuses is unchanged.

- Migration 0057 adds `docs:lock` wherever `docs:write` was held: every role, every permission overwrite's allow list
  (a doc creator's grant and Project access included), every pending invitation's grant and per-project levels,
  every integration install, and every automation. A deny on `docs:write` gains a deny on `docs:lock`, so nobody who
  could not lock a doc before the upgrade can after it. A doc's creator receives `docs:lock` on it with the rest of
  the creator grant. The Owner holds it through the bypass.
- Starting a run of a doc play locks the doc if it is not locked, and nothing unlocks it when the run ends. The lock
  comes with the start, not with the starter: starting a doc play needs no `docs:lock`, and a failed lock is logged
  and never fails the run. A doc already locked is left as it is.
- The lock is the ordinary one: it travels as `doc.updated` with `locked` set and the starter as the actor, so open
  pages and the doc list show it at once, and the run's trail and the doc's thread carry the line "Locked the doc
  because the run started; it stays locked after the run ends." Docs keep no record of who locked them; the trail is
  where a play's lock is explained.

The trade-offs: after the upgrade every writer can still lock, until an owner trims their role, because the backfill
errs toward keeping what people could do. A doc play now leaves its doc locked, and someone holding `docs:lock`
unlocks it when the doc should change again. A custom doc play meant to edit its own doc is refused by that lock,
because the lock refuses agents as it refuses people; the built-in "To tickets via AI" only reads its doc.

Rejected: unlocking the doc when the run ends, which the owner declined; locking only when the starter holds
`docs:lock`, which would make the same play behave differently per person; and a `locked_by` column on docs, which
the trail note already covers.

Amends ADR 0057, whose declared verbs gain `docs:lock`. Decided 2026-10-02.
