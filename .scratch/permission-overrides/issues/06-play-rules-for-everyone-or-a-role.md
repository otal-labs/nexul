# 06 — Play rules for everyone or a role

**Status:** ready-for-agent

**Blocked by:** 03

Read first: `practices/go.md` (section 17), `practices/react-guide.md`, `practices/mcp.md` (section 4),
`practices/testing.md`, ADRs 0057, 0148, the spec.

## What to build

- Plays take Everyone and role rules for `plays:run` in all three states, beside the person exclusion that
  exists (`internal/access/usecase.go:381-390`); managed with `plays:write` as now. Everyone Deny with a role
  Allow is how a play is kept to one role.
- The play's settings page uses the permissions panel limited to `plays:run`, replacing the exclusion dialog.
- Every path that lists or runs plays (the play menus, auto plays, `play_list`, `play_run`) answers through
  the same check; `play_list` filters before paging.
- The guide page `plays.md`.

## Acceptance criteria

- [ ] `TestPlays_EveryoneDenyRoleAllow`: only the role's members see and run the play, on HTTP, MCP, an auto
      play's match, and the play menu's live refresh.
- [ ] An existing exclusion still excludes after the upgrade (matrix diff empty).
- [ ] Verified at 768, 1024 and 1440px.
