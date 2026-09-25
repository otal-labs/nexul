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

## Cloudflare record writes report proxied as false

`decodeRecord` in `internal/dns/cloudflare/client.go`, used by the create and
update record calls, never reads `proxied`, so `dns_record_create`,
`dns_record_update`, and the matching HTTP responses say `proxied: false` for a
proxied record such as a tunnel CNAME. The write itself is right: reading the
record back through the list call shows the true value. Decode the field.

## A non-admin can tell whether an account id exists

`auth.Service.UpdateAccountStatus` looks the target up before the instance-admin
check, so a non-admin gets 404 for an unknown id and 403 for a real one, and a
call that would change nothing (an already-active account) succeeds without the
check. Both the web app and `account_update` behave this way. Checking the admin
bit first closes it.

## Direct messages are readable by any member who has the conversation id

`chat.Service.ListMessages` and `PostMessage` do not check that the caller is a
participant of a direct-message conversation, so anyone holding its id can read
and post, over HTTP and MCP alike. The doc-thread gate exists; direct messages
need the same kind of participation check in the use-case.
