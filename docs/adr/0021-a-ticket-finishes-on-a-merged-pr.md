# A ticket finishes on a merged PR, not on all its PRs closing

A ticket is **finished** when it has at least one linked PR, none of its
linked PRs are still open, and at least one of them was merged. PRs closed
unmerged are counted as no-longer-open but never as completion, so a dead
PR neither blocks the ticket forever nor finishes it by accident — a ticket
whose PRs were all closed unmerged simply never finishes on its own and
waits for a real one.

The tickets domain publishes `ticket.finished` once this becomes true, and
the default automation that moves the ticket to the project's done column
consumes that event rather than raw `git.pr_merged`. That indirection is
the point: the "is this work done" rule lives in the domain, and automations
only react to the answer, so a second automation can never disagree about
what finished means. Re-evaluation runs on both merge and close events,
because closing the last open PR is as much a state change as merging one.
