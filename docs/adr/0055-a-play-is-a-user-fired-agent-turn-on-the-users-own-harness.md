# 0055. A play is a user-fired Agent turn that runs on the user's own harness

Amended by ADR 0132: an auto play may start a play when a moment matches, still on one person's own harness.

Its move-on-success consequence is superseded by ADR 0112: a play never moves its ticket; footer memories tell the agent where it goes.

Superseded in part by ADR 0115: an interview run's thread is hidden plumbing holding its session; the run is read on the
project's Interview page, not as a conversation.

A play is a pre-configured Agent turn a user fires from a ticket, a doc, or
a project's interview with one button: "Fix with AI", "To tickets via AI",
"Run the interview". It is not an automation
(nothing fires it but a person) and not a chat mention (nothing is typed).
It runs on the clicking user's own paired harness with that user's
permissions, exactly as an `@Agent` mention does (ADR 0029), and it posts
into the target's thread so the run reads as a conversation.

The alternative was a workspace-level service harness so anyone with the
button could fire a play without pairing. Rejected: it would create the
shared identity ADR 0029 exists to avoid, and every trail would be
attributable to "the workspace" instead of a person. The cost is that the
button is disabled, with the reason, for anyone who has not paired a harness
or whose harness is offline.

Two consequences follow. A play's ticket move on success never moves the
ticket backwards, because the default automations already move it during
the run (PR opened, ticket finished) and a play must not undo them. And
every press leaves a persisted trail with the harness activity stream,
because a turn that acts on the codebase with no record of what it did
cannot be debugged, least of all when Nexul is being used to fix Nexul.

Decided 2026-09-16.
