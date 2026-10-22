# Runs go to the first ready computer in a person's order

Amends ADR 0102 and ADR 0132; supersedes ADR 0145's question for every case but one.

A person's runs had one place to go: the project's link, which pinned one computer, else the default computer. With
that computer offline a run waited, even with a second computer online and idle. A run could not fail over, and a
person with two computers could not spread work across them.

PR #838 gave every checkout T3 Code lists its normalized git remote, so Nexul can find a project's checkout on any
computer without being told. That removes the reason a link had to pin one computer.

Decision: a person keeps their own computers in one order, **Your computers**, and a run goes to the first one that can
take it.

- **Ready** means the computer is online with T3 Code answering, its session current, set up for the run's provider,
  offering the model, holding a checkout of the project, and, for a queued run, with a free slot (ADR 0166). The first
  gap is the reason the computer was passed over, and every passed-over computer is named on the waiting run and on
  the trail.
- **The checkout** on a computer is the one remembered for that person and project, else the one checkout whose git
  remote matches a repository of the project. Several matches are ambiguous and need a pick; none skips the computer
  unless the person's **Clone automatically** is on, which clones it under ADR 0162's folder rule first.
- **A project link** keeps provider, model, options and start-in for every computer, and either follows the person's
  order or names its own list. It no longer pins a computer.
- **Placement** is fill in order (first ready) or spread (least busy for its size, ties by order), set on the person
  and overridable on a project with its own list.
- **Before start, never during.** A run already going when its computer drops is not moved, since a second start could
  repeat half-done edits.
- **Own computers only.** A computer shared with the person is not in their order. Run on a chosen computer (ADR 0156)
  names an ordered list of its own, and the picker's rights are checked as before.
- **The pressed-play question.** ADR 0145 asked where to run in every unlinked project, because the fallback was one
  checkout that was usually wrong. Matching by remote finds the right checkout, so a pressed play asks only when no
  computer of the person's has the project at all. The answer is remembered as that computer's checkout. A pressed run
  never waits for room; with every ready computer full it starts on the first.
- **Chats** start a thread the same way, room ignored, and reply in an existing thread on that thread's computer.
  Outside a project they use the computer's chat folder, which replaces the one fallback T3 project.
- **Default computer** is gone; the top of the order is the default. On upgrade the old default goes first.

The trade-off: where a run lands now depends on live state (who is online, who is full), so the same press can land on
different machines on different days. The trail's computer and passed-over line are how that stays visible. Matching
by remote also trusts that a checkout with the right remote is a working copy of the project; a stale clone on a
second machine runs as it is.

Rejected: per-project failover lists only, which makes every project carry the same list; spreading by default, which
makes a person's main machine stop being the main one; moving a running run, which risks duplicated edits; and
counting shared computers into a person's order, which borrows someone else's capacity under rules ADR 0148 never
gave.

Decided 2026-10-22.
