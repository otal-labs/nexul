# Crash intake runs on the server, behind one route

A crash reporter posts each crash to a channel through a bot (crash-report/1: a `Crash fingerprint: <hash>` line and
one embed). The Crash intake default automation turns those posts into bug tickets: one ticket per fingerprint, a
running note for its repeats, and a new high-priority bug found in a done ticket when the crash comes back in a build
that holds every merged fix (ADR 0064).

Decision: the rules live in `internal/crashintake`, a use-case behind `POST /api/crash-reports`. The default automation
only picks out posts in its configured channel and hands each one to that route with its project, bug type and label.

- **A default automation has no creator.** Its token passes the gateway's scope check but names no person, and a
  ticket's note needs one to post as. The intake posts notes for the calling person, or, for a creator-less default,
  for the workspace's longest-standing Owner, the same person a stack's crash ticket comments for.
- **One record per crash, and it is the ticket.** A crash is found by searching the project's tickets for its
  fingerprint line, never by a table of its own. Each running note (seen again, old build, can't tell) is read back
  from the note to count, so the thread a person reads is the whole record.
- **Did the build get the fix?** The git provider's `MergedInto` asks GitHub whether each merged pull request's merge
  commit is the build commit or one of its ancestors. When GitHub can't answer, the intake notes why and files nothing,
  so an unknown never becomes a regression. There is no fallback on commit dates: the report carries no build date, and
  a date misjudges a build made from an older branch.
- **Two posts at once file one ticket.** The intake handles one report at a time, so the second post finds the ticket
  the first one filed.

The trade-off: a custom automation can call the same route, so the rules cannot be read or changed in the automation's
code the way a default's usually can.

Rejected: the rules in the automation, calling the existing ticket and chat routes, since the note routes refuse a
caller with no person; a table of fingerprints, which would be a second record that drifts from the tickets people edit.

Decided 2026-10-24.
