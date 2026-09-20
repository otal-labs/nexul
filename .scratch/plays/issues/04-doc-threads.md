# 04 — Doc threads

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Docs have no thread today; only tickets do (`KindTicketThread`, one per
ticket by unique index). A doc play needs one, and the owner wants it to be
a real conversation humans use too.

1. One thread per doc, same as tickets? Created lazily on first message or
   first run, as the ticket thread is?
2. Where it renders: the doc page is the editor (ADR 0050) and chat is a
   dock over a live page (ADR 0051). Below the editor like the ticket page,
   in the dock only, or both?
3. What an `@Agent` turn in a doc thread carries as context: the doc's
   title and body, as a ticket thread carries the ticket. In full, or capped?
4. Thread indicators and notifications: the ticket-thread indicator and
   inbox rules apply unchanged?
5. Does a doc thread appear in the chat page's conversation list, and under
   what name?

## Answer

Resolved 2026-09-16 with the owner (one grilling round).

- **One thread per doc**, a new conversation kind `doc_thread`, created
  lazily on the first message or the first play run, mirroring the ticket
  thread's get-or-create and unique index.
- **Renders in the chat dock**, opened from a "Thread" button in the doc's
  header beside the play buttons. No section under the editor. The owner
  may redesign how threads and the chat Agent surface later, as a separate
  design effort; the dock is the answer for this spec.
- **An Agent turn in a doc thread carries the doc's title and body as
  markdown**, trimmed to fit the turn limit with a note when cut; block
  order and trim priority are ticket 05's.
- **Visibility**: superseded by ticket 08. A doc thread is gated by a new
  docs verb, `docs:thread`, so a client who can read the doc need not see
  the work behind it. The vocabulary was extended in ticket 08 to allow
  domain-declared verbs.
- **Listed in the chat page** as "Doc thread" with the doc's title; unread
  and mention behaviour as for channel threads. No thread marker on the
  docs list for now.
