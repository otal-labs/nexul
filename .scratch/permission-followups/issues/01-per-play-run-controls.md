# 01: A play's own Run allow reaches its button

**Status:** needs-triage

**Blocked by:** None. Can be triaged independently.

## Problem

`web/src/components/play/PlayButton.tsx`, `PlaysRailSection.tsx` and
`PlaysMenu.tsx` check workspace `plays:run`. A person allowed Run by a play's
own rule can be offered that play by `internal/plays/usecase.go` but lose its
button to the workspace check.

## Scope

Align these per-play controls with the effective Run answer for that play.
Preserve server authorization and avoid fetching an explanation for every list
row. Coordinate with role-aware UI tickets 13 and 15; neither currently names
this allow-only regression. Do not duplicate the general action sweep.

## Acceptance criteria for triage

- Confirm the case with workspace Run absent, See allowed, and the play's own
  Run rule allowed. The offered play must have a usable Run control.
- Preserve denied Run behavior and the constant-request list contract.
- Agree ownership with the role-aware UI effort before implementation.
