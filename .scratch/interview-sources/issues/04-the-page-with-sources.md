# 04: The Interview page with sources and drafts

Type: prototype
Status: resolved
Blocked by: 01, 03

## Question

What does the Interview page look like with sources and drafts? Prototype
it live on top of the built page at 768, 1024, and 1440px; the owner picks
by looking.

- Adding sources and setting each one's stance, including pasting text.
- A project with sources and no answers: drafting running, then the
  questions with each draft as the recommended pick and where it came
  from.
- A question no source spoke to, beside drafted ones.
- A source added later: the few redrafted answers coming back, beside an
  answer the person had already confirmed.
- A superseding project whose only source is its predecessor, under
  question.

## Answer

Picked from three live layouts: **the Sources section**. Prototype on branch
`proto/interview-sources` (commit 17b28cbf; `?variant=A&state=1..6` on the
Interview route), the primary source for the build.

- A collapsible "Sources" section at the top of the question column, above
  "Initial questions", with the same swimlane header look. The header
  holds the count ("8 sources · 7 follow · 1 question") and "Draft
  answers", which carries the warning dot when a source is new or changed
  since the last drafting run and is absent with no follow source.
- The drafting run's state sits as one line under that header with the
  trail icons ("Drafting · 3 of 12 drafted", "Drafts ready · 9 of 12
  drafted").
- One hairline row per source: kind icon, name (paths in mono, long names
  truncate), a Follow | Question segmented control, and a remove control.
  A gone ref reads "Memory · No longer there" with Remove; one the viewer
  cannot read reads "Doc · Not visible to you". "+ Add source" ends the
  list and opens an inline form: kind (Path, Doc, Memory, Project, Paste
  text), the matching field, the stance preselected per 01, Add and
  Cancel; pasting offers "or drop a text or markdown file".
- With only question sources, the section shows "Sources under question
  are asked about in the follow-ups, not drafted from." as its first line
  and no button.
- A drafted, unanswered row shows its number in a dashed ring and
  "Drafted · from <source>". Opened, the card preselects the drafted
  option tagged "Drafted", with a muted line under the question: From
  <source>: "<passage>". Next confirms it.
- A suggested change shows a warning dot and "Suggested change · from
  <source>" in the row; opened, it shows "Your answer" over the bordered
  suggestion with where it came from, and Dismiss / Accept. The section's
  count adds "· N suggested".
- Rejected: the sources in the memory column, which drops under all the
  questions at 768 and 1024px; a chip strip under the page header, which
  wraps into a grid past five sources.
