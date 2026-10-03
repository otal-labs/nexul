# 08 — The Interview page

**What to build:** The page decided in 05, built for real from the
prototype on `proto/interview-page` (rewrite it properly, don't copy the
throwaway): the checklist beside the memory column, "Initial questions"
and one collapsible section per follow-up round with the swimlane header
look, the open row holding the question card with Back, Skip, and Next,
answered and skipped rows reopening on click, every Next and Skip saving
through 07, live updates from other people. The memory column shows the
memory, its empty state, and the run-state header line with the trail
icons. The Conversation section and the "Start from the template" empty
state go. A project with a memory but no answers shows the state from 05.
Finishing the last unanswered initial question shows a Done that starts
the follow-up run (wired in 09; until then it starts the existing play).
Built at 768px first, verified at 768, 1024, and 1440.

**Blocked by:** 06, 07

**Status:** resolved

- [x] Every state in 05 renders from real data at the three widths
- [x] Answers save per step and survive a reload; two browsers see each other
- [x] Conversation section and "Start from the template" removed

## Answer

Built in PR_URL.
