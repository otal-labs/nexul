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

## Stack events carry env values in plaintext

`service.created` and `service.updated` publish the whole stack, including its
env values and, since per-branch overrides, each branch rule's override
values. Webhook redaction only covers `deploy.requested`, so any integration
subscribed to the stack events receives secrets. Redacting them changes a
published event contract (ADR 0044), so the fix needs the owner's call: a new
schema version without values, or keys-only fields added beside the old ones.

## Ticket people cannot be edited on a phone

The ticket page's side panel is hidden below the `lg` breakpoint, so on
320–768px widths the developer and tester can be neither seen nor changed;
only the reporter shows, in the line under the title. The panel needs a
mobile placement (a sheet or an inline section) like the rest of the page.
