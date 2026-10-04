# 13 — Where a draft came from, and the drafting line after a redraft

**What to build:** Two rough edges the walkthrough found once drafts met
real sources.

- The draft's "From" line names the source twice. The page shows
  `From <source>: <where>`, and the drafting run writes `where` with the
  source's name in it, because the instructions' example does
  (`practices/testing.md, Test error paths first`). On Clutch Hub every
  card read "From .NET backend standards: .NET backend standards, Stack".
  Change the Draft interview instructions so `where` says where inside the
  source (a heading, a file inside a folder source, a quoted phrase) and
  never repeats the source's name, and add a forward migration updating
  the built-in play's instructions where the workspace never edited them,
  as the earlier interview builds did. Check `migrations/` on master for
  the next free number first.
- After a redraft the Sources section's run line reads "Drafts ready · 3 of
  12 drafted" when the run only suggested 3 changes to answered
  questions, and "1 of 12 drafted" once one was accepted and two
  dismissed. Count suggested changes as suggested ("3 suggested"), and
  stop counting a draft once it is confirmed, so the line matches the
  question list's own count.

**Blocked by:** None — can start immediately

**Status:** resolved

- [x] The instructions migration leaves an edited workspace play untouched
- [x] A drafting run on a fake harness with a doc source writes a `where`
      without the doc's title, and the card's From line names it once
- [x] The run line after a redraft counts suggestions apart from drafts,
      with a component test

## Answer

Built in PR #PRNUM: migration 0073 rewords the Draft interview's `where` line, and the run line counts suggestions apart.
