# Latent bugs

**Status:** needs-triage

Small defects seen in passing that have no effort of their own. One heading
each; delete the heading when it is fixed, delete the file when it is empty.

## The outbox crash-redelivery test is timing-sensitive under the race run

`TestRelay_CrashRedelivery_DedupeEndToEnd` in `internal/platform/eventbus/outbox`
failed once inside a full `make coverage` run with "row must be marked
published after the successful re-publish, actual: 0", then passed three
consecutive `-race -count=3` runs of the package alone, on the same commit and
on master. The relay's second boot polls every 5ms and the assertion waits on
`require.Eventually`; under the load of the whole suite the mark-published
write appears to land after the wait gives up. Worth a look at the wait budget
or at asserting on the bus delivery instead of the row flag.
