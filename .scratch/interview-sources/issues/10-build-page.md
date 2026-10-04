# 10 — Sources and drafts on the Interview page

**What to build:** The page decided in 04, built for real from the
prototype on `proto/interview-sources` (rewrite it properly, do not copy
the throwaway): the Sources section above "Initial questions" with its
header count, "Draft answers" with the warning dot (a source added after,
or a doc, memory, or text source changed after, the last finished
drafting trail started), and the drafting state line; source rows with
the stance control and remove, gone and not-visible rows; the inline add
form with every kind, the per-kind default stance, and a dropped text or
markdown file becoming pasted text (any other file refused with "paste
its text instead"); drafted rows and the preselected card with its From
line; suggested changes with Accept (saves the answer) and Dismiss
(dismisses the draft); the "· N suggested" count. Everything live-updates
from other people. Read-only viewers see the list and drafts without
controls. Built at 768px first, verified at 768, 1024, and 1440.

**Blocked by:** 07, 08

**Status:** resolved

- [ ] Component tests for the add form refusals and the default stance,
      draft confirm, accept, and dismiss
- [ ] Verified in a browser at 768, 1024, and 1440px

## Answer

Built in PR #437: the Sources section with the stance control, add form,
gone and not-visible rows, Draft answers with its warning dot and the
drafting state line, drafted rows, suggested changes with Accept and
Dismiss, and a growing text box for questions with no options.
