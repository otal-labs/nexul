# 09 — The follow-up run

**What to build:** The Interview play's instructions rewritten for the new
flow (code default and the instance template; existing workspaces keep
their copied instructions, so ship a forward migration updating the
built-in interview play's instructions where the workspace never edited
them): read the interview memory, which now carries the questions and the
answers; read the checkout; ask about skipped questions, gaps, and anything
the code contradicts, a round at a time with the question tool, each
question with its recommended option first and a one-line why; never record
something from the code the person did not confirm; no "scan the codebase?"
question; then write the memory as rules, keeping any existing rule no
answer contradicts, under the 8,000 cap. `Runner.Answer` writes each
follow-up and its answer into the stored answers as the next round when
the run's target is an interview (research in `research/`). The page shows
the live question of the active interview trail in the open follow-up row,
answered through the trail route, and the run state from the trail. The
interview thread stays as hidden plumbing.

`CONTEXT.md` (Interview, Interview memory, Play) and a new ADR recording
that the interview is questions answered on the page plus a follow-up run,
not a conversation, superseding what ADR 0065 and the plays ADRs say about
the interview thread.

**Blocked by:** 07, 08

**Status:** resolved

- [x] A run on a fake harness: rounds land as stored answers, the memory is
      written, existing rules survive
- [x] The page shows each round's live question and folds earlier sections
- [x] Instructions migration leaves an edited workspace play untouched
- [x] CONTEXT.md and the ADR updated

## Answer

Built in PR #362: the rewritten Interview play instructions with migration
0062, `Runner.Answer` storing each answered follow-up round, the live
question of a waiting run as the next follow-up section on the Interview
page (answered on the trail, replaced by the stored round), `CONTEXT.md`,
and ADR 0115.
