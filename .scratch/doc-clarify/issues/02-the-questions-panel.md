# 02: The Questions panel on the doc page

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

What does a clarification look like on the doc page, through the whole
loop? Prototype it live at 768, 1024, and 1440px; the owner picks by
looking.

- The client's view: a round of questions waiting, stepping through the
  cards, Skip, the "Anything else?" box, the reply line under an earlier
  "Anything else?", and earlier rounds collapsed. No AI or Agent wording
  anywhere.
- The developer's view: the same panel plus "Clarify via AI", a round
  fully answered, the "no gaps left" signal, Close, and "To tickets via
  AI" after closing.
- A doc locked while a round runs, as the client sees it.
- A closed clarification, and reopening it with a new round.
- Where the panel sits beside the doc body and the thread pane, and
  whether the docs list or the header needs a marker for "waiting on your
  answers".

## Answer

Picked from three live variants: the tab. Prototype on branch
`proto/doc-clarify-panel` (commit 5f9fad49; `?variant=C&state=1..7&as=client|dev`
on the doc route), the primary source for the build.

- A "Doc | Questions N" segmented switch at the top of the doc page; the
  count is the questions waiting. The Questions view replaces the article
  at full width; the Doc view is the page as it is today.
- Inside it, the Interview page's checklist: a "Questions" heading with the
  state line ("2 questions waiting · Round 1"), then one foldable section
  per round with its count ("2 answered · 1 skipped"). Numbered rows on
  hairlines; only the current row is open, holding the question with
  "Question 4 of 5", the "Why we're asking:" line, the options with
  "(Suggested)" first and never pre-picked, "Something else…", and Back,
  Skip, Next. Answered rows show a tick and a muted one-line answer,
  skipped rows a dash and "Skipped"; both reopen on click.
- Each round ends with the optional "Anything else?" box. New rounds are
  added below the earlier ones, which fold; a folded round keeps its
  "Anything else?" text and the one-line reply visible under it.
- While a round runs: the client sees the muted line "More questions are
  on the way" and the plain locked state; people who can run plays see the
  trail spinner, "Round N running", and that the doc is locked until it
  ends.
- A round fully answered: the client sees "All answered"; the developer
  sees "Round N answered" and "Clarify via AI" as the primary button (it is
  secondary while questions are still waiting).
- No gaps left (developer only) with Close; closed shows every round folded
  to its count, "Closed · N rounds · N answered", with "Clarify via AI" to
  reopen and "To tickets via AI"; the client sees "All answered · N rounds".
- No separate header or docs-list marker: the tab's count does that job.
