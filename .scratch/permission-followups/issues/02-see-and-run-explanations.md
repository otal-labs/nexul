# 02: Explain denied See when Run is allowed

**Status:** done

**Blocked by:** None. Needs a behavior and wording decision before implementation.

## Problem

`internal/access/explain.go` answers each action separately. A target denied
See can still read as allowed Run, while `internal/plays/run_launch.go` refuses
the run on See first. The panel's Run answer does not explain that prerequisite.

## Scope

Decide how the permission panel distinguishes the Run rule's answer from the
ability to execute a play when See is denied. Keep the source of each answer
visible. Do not silently change rule evaluation or weaken the launch check.
The role-aware UI effort's single-item answers ticket 15 consumes explanations
but does not decide this See/Run mismatch.

## Acceptance criteria for triage

- Reproduce See denied and Run allowed, with the panel and launch result.
- Record the chosen wording and behavior in this ticket before implementation.
- The resulting explanation must make the See prerequisite clear while
  preserving the sources of the separate See and Run answers.

## Answer

Decided 2026-10-14.

- **Explain stays per action.** Every answer keeps its own source; rule
  evaluation and the launch check are unchanged, so a run on a play its starter
  can't see is still not found.
- **Run carries See as a prerequisite.** When the explanation's target (a person
  or a role) doesn't hold `plays:read` on the play, the `plays:run` answer adds
  `requires: {action: "plays:read", sentence: <See's source sentence>}`. The
  field is additive, over HTTP and in `permission_overwrite_list` with `user_id`.
- **The panel's wording.** The Run row shows the rule's own state as before, and
  under it, in the rules pane and in Check access, a muted line: "Can't run
  while See this play is denied · Denied on play Deploy prod for role Client",
  the second half being See's source sentence.
