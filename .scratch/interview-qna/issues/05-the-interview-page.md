# 05: The Interview page

Type: prototype
Status: resolved
Blocked by: 01, 04

## Question

What does the Interview page look like through the whole flow? Prototype
it live at 768, 1024, and 1440px; the owner picks by looking.

- A project with no answers: the empty state and the first question card.
- Stepping through the template's questions, and the moment the last one
  is answered and the follow-up run starts.
- Waiting on the run, then its follow-ups in the same card.
- The finished memory, and the answers alongside it.
- Coming back later: changing one answer and generating again.
- A project that already has a memory but no answers.

## Answer

Picked from three live variants: the memory column. Prototype on branch
`proto/interview-page` (commit 6513b0c9; `?variant=B&state=1..7` on the
Interview route), the primary source for the build.

- Two columns at wide widths: the question checklist on the left, the
  interview memory on the right as its own column, with the run state in
  that column's header line ("Agent asking · Follow-up 1 of 2", "Memory
  ready · Written 2 min ago" and Regenerate). Where the content area is too
  narrow for both (about 1024px with the sidebar open), the memory column
  drops under the checklist.
- The checklist: numbered rows separated by hairlines, no card per row.
  Only the current row is open and holds the question card, with Back,
  Skip, and Next. Answered rows collapse to a success tick, the question,
  and a one-line muted answer, and reopen on click; skipped rows show a
  muted dash and "Skipped"; pending rows show their number in a ring.
- Rows sit in collapsible sections with the board swimlane's header look:
  "Initial questions" with a count ("10 answered · 2 skipped"), then one
  section per round of follow-ups, "Follow-ups from the agent 1", "… 2",
  each with its own count ("0 of 2 answered"). When a new round arrives the
  sections above it fold, so the open question sits near the top; any
  section opens again on a click.
- A follow-up row carries "Follow-up 1 of 2", a "Why I'm asking" line, and
  its recommended option first, preselected and labelled (Recommended).
- Run-state icons are the trail's: a spinner while the agent reads or
  writes, the blue waiting icon while it asks, the success tick when the
  memory is ready. After an answer changes under an existing memory, the
  header shows the warning dot and a Regenerate the memory action.
- A project with a memory but no answers shows the memory and all twelve
  questions pending, with one line saying answering them lets the agent
  update it.
- The memory column with no memory shows an empty state saying the agent
  writes it once its follow-ups are answered.
