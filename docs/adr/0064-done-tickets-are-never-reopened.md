# Done tickets are never reopened; a bug after done is a new ticket found in it

A ticket in a done column carries its why: its doc, its merged PR, its
thread, and possibly a decisions-log entry. Reopening it would rewrite that
record and make the merged PR stop being the implementation record. So a bug
found before done moves the card back to progress, because the work is still
in flight, while a bug found after done becomes a new bug ticket found in the
done one. The done ticket lists the bugs found after it, and an agent fixing
a bug receives one hop of context: the origin ticket, its doc, and its PRs,
never the origin's origin.

Because reopening is ruled out, the bug type requires its found-in link,
unless the reporter marks the origin unknown, which is recorded as such
instead of guessed. The normal create dialog offers only feature and task;
bugs come from Report a bug, a failed test, or the board's own report
action.
