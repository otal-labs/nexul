# 02: Explain denied See when Run is allowed

**Status:** needs-triage

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
