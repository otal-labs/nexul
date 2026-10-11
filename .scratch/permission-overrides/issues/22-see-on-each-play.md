# 22 — See on each play

**Status:** ready-for-agent

**Blocked by:** 14, 17

**Branch:** master, a normal PR, after the switch. A new ability: no play carries a See rule on upgrade.

Read first: `practices/go.md` (section 17), `practices/mcp.md` (sections 4, 7, 11), `practices/react-guide.md`
(Live events), `practices/testing.md` (section 10), ADRs 0057, 0119, 0145, 0148, the spec (Channels and plays stay
workspace-wide, Evaluation order), the research (sections 5 and 8).

## What to build

- `plays:read` set per play: play rules set See, Run, Edit and Delete (`plays:read`, `plays:run`, `plays:write`,
  `plays:delete`), written with `plays:write` on the play, under hold-what-you-give.
- Every path in research section 8 answers not found for a play the caller can't see: `plays.List`, `Get`,
  `BuiltinPlay`, `ListApplicable`, `checkPlay`, auto play matching and lists, queued runs, the clarify gate, and
  `play_list`, `play_run`, `trail_list`, `trail_update`.
- A run's starter reads their own trail, with or without `plays:read`; others read it when they can see the play.
- `playFrame`, `playRunFrame` and `autoPlayFrame` (`server/cmd/live_audience.go:427-480`) follow See, and the
  starter.
- The play panel gains a "See this play" row.
- The guide page `plays.md`.

## Acceptance criteria

- [ ] `TestPlays_SeeHidesItEverywhere`: a play denied to a role is gone from the list, run menus, auto plays and
      agents' play lists for its holders, and its link is not found, over HTTP and MCP.
- [ ] `TestPlays_TheStarterReadsTheirRun`.
- [ ] "a play's queued runs" and the plays list keep their counts with rules on every play; new
      `TestStatements_PlayRulesCostTheSameAtAnyLength`.
- [ ] Verified at 768, 1024 and 1440px.
