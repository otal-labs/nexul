# 10 — The Questions panel on the doc page

**What to build:** The tab picked in 02 ("Doc | Questions N" on the doc page), built for real from the
prototype on `proto/doc-clarify-panel` (rewrite it properly, don't copy the
throwaway). The interview's reusable rows, sections and option steps move to
a shared folder rather than being imported from the memory components. Every
answer, skip, clear, and "Anything else?" saves through 07, with live
updates from other people. People who can run plays see the run state, the
"no gaps left" and "answers changed since the doc was written" signals,
Close, and "To tickets via AI" after closing; nobody else sees agent
wording. Built at 768px first, verified at 768, 1024, and 1440.

**Blocked by:** 02, 07

**Status:** resolved

- [x] Component tests: client view with no agent wording, permission-gated
      developer controls, a locked doc still answerable
- [x] The F1–F7 self-review in `practices/react-guide.md`
