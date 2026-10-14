# 14: Progress on Team and accepted invitations

Type: task
Status: resolved
Blocked by: 09

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/go.md` (section 17), `practices/react-guide.md`, `practices/design-language.md`,
`practices/testing.md` (section 10), ticket 06's answer, the branch `proto/team-progress` (read it; never
merge it).

## What to build

- **Team people list:** `TeamPerson` gains `welcome` and `joined`.
- **Invitations:** the list returns the ones accepted in the last 30 days.
- **The gap line** under each person, the person dialog's Welcome block, and the Invitations tab's "Accepted in
  the last 30 days", all as ticket 06 locked them.
- **"Not in a project yet"** shows on Team whatever the dot says.

## Acceptance criteria

- [ ] The gap line wording and states follow ticket 06, including all done (nothing) and disabled accounts.
- [ ] Statement-count guard: the Team list reads progress in a fixed number of statements for any length.
- [ ] The line updates live from `welcome.progress_changed`. Verified at 768, 1024, 1440.

## Answer

Built in #643: `welcome` and `joined` on Team people, the gap line, the person dialog's Welcome block, and accepted and ended invitations on the Invitations tab.
