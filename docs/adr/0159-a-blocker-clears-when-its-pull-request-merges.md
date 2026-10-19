# A blocker clears when its pull request merges, not only when it reaches done

A blocked-by link held its ticket until the blocker entered a done-stage column. A blocker whose pull request had merged
but which sat in a testing column for QA still showed "Blocked by" on every ticket waiting on it, though the code those
tickets need was already on the default branch.

Decision: a blocker is cleared once it is **finished** (a linked pull request merged and none is still open, ADR 0021) or
in a done stage, whichever comes first. Finishing publishes `ticket.unblocked` with the new cause `blocker_finished`, in
the same transaction as `ticket.finished`. `finished_at` is set once and never cleared, so a finished blocker stays
cleared when QA sends it back: the fix ships as another pull request on the same ticket, and the original code is still
merged. A blocker with no pull request (a design task, a decision) still clears only at done.

Rejected: clearing at the testing stage, which would unblock a ticket whose pull request is still open; and re-blocking
when a finished blocker leaves done or gets a new open pull request, which would make `ticket.unblocked` fire again for
the same code and start the same auto play twice.

Decided: 2026-10-19
