# 01: A play's own Run allow reaches its button

**Status:** done

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

## Answer

Decided 2026-10-14.

- **The server sends the answer.** The plays list and the applicable plays list
  carry `can_run` per play, decided by the same batched rule read that decides
  See (`PlaysWith`), with no explain per row and no extra statement. The
  applicable list still holds only plays the caller can see and run, and
  `play_list` gains `can_run` on each play. The field is additive.
- **The controls follow it.** `PlayButton`, `PlaysRailSection`, `PlaysMenu` and
  the queued run's Run it use `can_run` instead of workspace `plays:run`. The
  readiness note follows the plays it sits beside. A play you can see but not
  run stays hidden from the controls, as before.
- **Ownership.** Per-play Run on the play controls is done here; role-aware UI
  ticket 13 does not redo it.
