# 07 — The button and the run dialog

**Type:** prototype
**Status:** resolved
**Blocked by:** 02

## Question

Raise the fidelity of the run flow with a throwaway UI the owner reacts to,
built with `design-mode` and verified at 320/375/414/768px:

- Where the play buttons sit on the ticket page and the doc page, with the
  doc header's "Thread" button beside them (the
  reference the owner gave is a top-right "Fix with AI" button beside
  Share), and how several plays stack on a phone.
- The three button states: hidden (no permission), disabled with reason
  ("Pair a harness in Settings", "Your harness is offline"), and enabled.
- The run dialog: memory multi-select, custom instructions, and for a
  ticket play the move-to column picked from the project's own columns,
  each pre-selected from the user's last run; confirm.
- The running state on the page and on the board card, the Stop control,
  and the thread showing the run start and the Agent's reply.
- Where run history lives: a runs list on the play in settings, a runs
  section on the ticket or doc, or both, each row opening the record with
  its activity stream.

Link the prototype from this ticket; do not merge it.

## Answer

Resolved 2026-09-17 with the owner, who flipped through the prototype in
the browser. Prototype: branch `proto/plays-ui`, route `/prototype/plays`
(dev builds only), three placements switchable via `?variant=`, state via
`?state=`, ticket or doc via `?target=`. Not merged; the branch is the
primary source.

- **Ticket page: variant B.** A "Plays" section in the right rail, placed
  **under Properties**, one ghost button per play. On narrow widths the
  rail is hidden and the plays become a sticky bottom bar.
- **Doc page: variant C.** One "Plays" menu button in the doc header beside
  the Thread button and the settings menu; each play is a row with its
  label and description; a disabled reason shows once at the top of the
  menu.
- **The run history is called the Trail** and lives under the ticket body,
  above the thread, full width: one hairline row per trail (spinner, tick,
  or cross; play, summary, starter, when), each opening the full trail: the
  choices made and every step the Agent took, ending in the outcome.
- **The run dialog ships as prototyped**: title and description, the
  memories list with always-included ones ticked and locked, an
  instructions box, a move-to select for ticket plays with the
  "never backwards" note, and a confirm button naming the destination.
- **Running state**: spinner in the play button with a Stop square, a
  spinner beside the id on the board card, and a spinner line in the
  thread. No banner; everything the Agent says or does lands in the thread.
- **Small-screen web verification is not required** for this effort; the
  owner plans a separate mobile app. The bottom bar exists so the ticket
  page stays usable when the rail collapses, not as a phone design.
