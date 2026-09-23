# Ticket workflow depth: dependencies, richer types, comments, and activity

**Status:** needs-triage

> Dependencies and the bug type (its template and required relation) moved to
> `.scratch/llm-setup/` on 2026-09-23 and were decided there with a lighter
> rule: a blocked ticket shows an icon and never refuses to move. Only
> comments and activity, and custom fields, remain for triage here.

Carried over from the retired tickets requirements document. Everything here
was specified before the board was built and none of it shipped; it needs
re-confirming against the product as it stands today before any of it is
sliced into tickets.

## Problem Statement

The board tracks what a ticket *is* and where it sits, but not the things a
team needs once more than one person works off it:

- Nothing records that one ticket cannot start until another finishes. The
  ordering lives in somebody's head, a blocked ticket gets pulled into
  progress anyway, and the person who pulled it finds out when they open the
  code.
- Ticket types exist as a name and a color only. A bug filed by one person
  and a bug filed by another share nothing — no prompt for reproduction
  steps, no requirement to point at what it regressed, no field that can be
  filtered or automated on.
- A ticket has no conversation and no history. There is nowhere to ask a
  question about a ticket that stays attached to it, and nothing shows who
  moved it, when, or from what — so "why is this back in review" has no
  answer the ticket itself can give.

## Solution

Three additions to the ticket, all visible on the ticket page and reflected
on the board:

**Dependencies.** A ticket can declare that it is blocked by one or more
other tickets. A blocked ticket is still visible on the board and can still
be groomed, but cannot leave the backlog stage until every blocker sits in a
status whose stage is `done`. The card and the ticket page say plainly that
it is blocked and link to each blocker. Completing the last blocker makes it
eligible immediately. Cycles are rejected at the point somebody tries to
create one.

**Richer ticket types.** A type can carry a body template (a bug type opens
with Steps to reproduce / Expected / Actual already in the body), can require
a relation to another ticket (a bug must point at what it regressed), and can
define structured custom fields where a value needs to be filtered,
validated, or automated on. Guidance that is only guidance stays prose in the
template; a custom field is for values the board or an automation reads.

**Comments and activity.** A comment thread on the ticket, and a timeline
recording the changes that matter — status moves, assignee changes, type and
category changes, links added or removed — interleaved with the comments so
one column tells the whole story.

## User Stories

1. As a developer, I want to mark my ticket as blocked by another ticket, so that nobody starts it before the work it depends on exists.
2. As a developer, I want a blocked ticket to refuse to leave the backlog, so that the rule holds even when somebody drags the card without reading it.
3. As a developer, I want a blocked ticket's card to say what is blocking it and link there, so that I can see the reason without opening anything.
4. As a developer, I want the dependency to clear the moment the last blocker reaches a done-stage status, so that I never have to remember to unblock things by hand.
5. As a developer, I want a dependency that would create a cycle to be rejected when I create it, so that the board can never reach a state where nothing can start.
6. As a project owner, I want an automation to be able to react to blockers completing, so that a freed ticket can be announced, but never to be able to start a blocked ticket, so that the rule has exactly one meaning.
7. As a project owner, I want to give a ticket type a body template, so that everyone filing that kind of work answers the same questions.
8. As a person filing a bug, I want the body pre-filled with the sections the type expects, so that I do not have to remember the format.
9. As a project owner, I want a ticket type to require a link to another ticket, so that a regression always points at what it regressed.
10. As a project owner, I want to define a structured field on a type, so that I can filter the board on a value rather than reading it out of prose.
11. As a project owner, I want structured fields to be validated on save, so that a filter over them is trustworthy.
12. As a developer, I want to comment on a ticket, so that a question about the work stays attached to the work.
13. As a developer, I want to edit and delete my own comments, so that I can fix what I wrote.
14. As a developer, I want to be notified when something happens on a ticket I am involved in, so that a comment addressed to me is not missed.
15. As a reviewer, I want a timeline of a ticket's changes, so that I can see who moved it, when, and from what status.
16. As a reviewer, I want comments and changes in one interleaved timeline, so that a comment sits next to the change it was about.
17. As a project owner, I want relations between tickets to be explicit records, so that nothing is inferred from what somebody typed in a title.
18. As a person reading a blocked ticket, I want its blockers listed even when I cannot open some of them, so that the reason is never invisible.

## Implementation Decisions

- Dependencies are a record in the tickets domain, not a ticket field, and
  they are directed: blocked-by and its inverse blocks are the same row read
  from either end.
- Eligibility is a tickets-domain invariant enforced in the use-case layer,
  not in the adapters and not in an automation. Both adapters and every
  automation path go through the same transition call, so the rule cannot be
  bypassed by choosing a different entry point.
- Eligibility reads a status's **stage**, never its name (ADR 0022): a
  blocker is complete when its status is in the `done` stage, whatever that
  project chose to call the column.
- Cycle rejection happens at write time on the dependency record, so the
  board never has to defend against a cycle at read time.
- Ticket types are already project-scoped workspace-domain records
  (ADR 0007); templates, required relations, and custom field definitions
  extend that record rather than introducing a new owner.
- A body template is applied at creation time only — it seeds the body and is
  never re-applied, so editing a template does not rewrite existing tickets.
- Custom field values are stored per ticket against the type's field
  definition. Changing a type on an existing ticket must have a defined
  answer for values that no longer have a definition; losing them silently is
  not acceptable.
- Comments belong to the tickets domain, alongside labels and links, and are
  authored, edited, and deleted by their own author, matching the rule chat
  messages already use.
- Activity entries are derived from the events the domain already publishes
  rather than written by hand at each call site, so a new transition path
  cannot forget to record itself.
- Notifications for ticket activity go through the existing workspace
  notification fan-out; this spec adds no second notification mechanism.
- Permissions stay workspace-level throughout (ADR 0023): a dependency, a
  comment, or a relation is a data relationship and never grants access to
  anything.
- Real-time collaborative editing of a ticket body is explicitly not part of
  this — ordinary save semantics are enough.

## Testing Decisions

Tests assert externally visible behaviour: what a transition does, what an
API call returns, what the timeline contains — never how the eligibility
check is structured internally.

- Tickets use-case tests, table-driven, are the primary seam: blocked
  transition refused, allowed once the last blocker reaches a done-stage
  status, cycle creation rejected, self-dependency rejected, an automation
  path refused exactly as a user path is. Prior art: the existing transition
  and label tests, which already drive the service against fake repos and a
  fake status store.
- The same seam covers type templates and custom fields: a created ticket
  carries the template body, a missing required relation is rejected, an
  invalid custom field value is rejected.
- Comments reuse the authored-content test shape chat messages already have:
  author may edit and delete, a non-author may not, deleting twice is a
  no-op.
- An integration test against real SQLite covers the timeline: a sequence of
  real transitions produces the expected ordered entries, and entries survive
  a status being renamed or deleted. Prior art: the existing docs/tickets
  storage integration tests.
- Board rendering of the blocked state is covered at component level with a
  staged blocked ticket, not by driving a whole board.

## Out of Scope

- Subtasks. Documents carry the higher-level context; tickets stay the
  actionable unit.
- Milestones. If they return, they join as another board filter dimension,
  not a separate mechanism.
- Per-ticket permissions of any kind.
- Real-time collaborative editing inside a ticket.
- Requiring a GitHub Issue anywhere in the flow.
- Changing how a ticket finishes (ADR 0021) — dependencies gate starting, not
  finishing.

## Further Notes

The board already assumes this is coming: the transition check allows any
configured status pair today and is the place the eligibility rule lands, and
status stages were made a fixed ordered enum partly so a dependency rule
could read them.

Sequencing, if the whole thing is confirmed: dependencies first (they are the
one piece with an invariant attached), then comments and activity (they share
a surface on the ticket page), then type templates and custom fields (the
largest, and the least blocked by anything else).
