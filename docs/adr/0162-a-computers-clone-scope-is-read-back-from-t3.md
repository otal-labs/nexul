# A computer's clone permission is read back from T3, never stored

Cloning a repository onto a computer needs the `source-control:write` scope on the computer's T3 session. A computer
paired before Nexul asked for that scope holds the older set, so its clone failed first and only then said to pair it
again, and the project picker offered a Clone button that could not work.

Decision: Nexul asks T3 what the session may do instead of recording it. T3's `GET /api/auth/session`, called with the
session's bearer token through the computer's runner like every other request to T3, answers `permissions`, the full
list, and `scopes`, the subset with the older names. The list is read when the picker loads the computer's projects and
when a clone starts. The picker says to pair again, with a button, before any click, and a clone without the scope is
refused before T3 is asked to start one.

- **Unknown is not lacking.** A T3 that has no such endpoint, answers neither field, or cannot be asked counts as able:
  T3 itself refuses a clone it will not take, so a guess against a working computer would be the only wrong answer.
  `permissions` wins over `scopes` when both are present, and an empty list is a grant of nothing.
- **Harness-neutral.** The harness seam gains `CanClone`; the scope's name stays inside the T3 packages.

Rejected: storing the granted scopes on the computer when it pairs. That needs a migration, leaves every computer
paired before it unanswered until it re-pairs, and goes stale when a session is revoked or re-granted inside T3.

Decided 2026-10-21.
