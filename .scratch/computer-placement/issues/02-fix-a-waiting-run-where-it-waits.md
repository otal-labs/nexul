# 02 — Fix a waiting run where it waits

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, decision 6 and Rules.

A waiting run's `ambiguous_checkout` pass-over opens a checkout picker for that computer on the run itself (the queue
card and the trail), listing that computer's matching T3 projects. Saving writes the pick through the project link's
`checkouts`, which keeps a pick for a computer outside the project's list, and `SetProjectLink` already wakes the queue.
`no_checkout` and `clone_failed` open the existing Add picker for that computer the same way. `CHECKOUT_FIX_LINKS` in
`web/src/models/Pairing.tsx` stops linking to the Projects tab for these three.

Done when: a component test picks a checkout on a waiting run and asserts the saved link names that computer; a second
case covers a computer that is not in the project's own list. Seen working on a dev stack with a run held on an
ambiguous computer, which then starts. `plays.md`, the queue section, says where the fix is.
