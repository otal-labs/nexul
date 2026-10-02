# Latent bugs

**Status:** needs-triage

Small defects seen in passing that have no effort of their own. One heading
each; delete the heading when it is fixed, delete the file when it is empty.

## A data race shows up intermittently in the DNS tests

One `make coverage` run on 2026-09-29 failed with a `-race` report in the
`internal/dns` package tests, then passed three focused `-race` reruns and a
second full coverage run. A race report is a real concurrent access, not
timing noise; find the shared state it names and guard it.

## Live push broadcasts every user's session and token events to every browser

The live hub sends `session.created`, `session.revoked`, and the personal
access token topics to every connected browser; the web filters on
`user_id`. Nothing secret travels (no token, no IP), but any signed-in user
can watch when someone else signs in and on what device. Route these topics
to the owning user's connections only.

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

## Direct messages are readable by any member who has the conversation id

`chat.Service.ListMessages` and `PostMessage` do not check that the caller is a
participant of a direct-message conversation, so anyone holding its id can read
and post, over HTTP and MCP alike. The doc-thread gate exists; direct messages
need the same kind of participation check in the use-case.

## Deleting a project that still has docs is an internal error

`workspace.Service.Delete` refuses a project with tickets, repositories, or
services, but `DeleteImpact` does not count docs, so a project holding a doc
passes the check and the delete then fails on the docs foreign key. The web
app gets a 500 and an MCP agent gets "internal error" instead of a conflict
that says to move or delete the docs first. Count docs (and any other row
that references the project) in the impact, or map the constraint failure to
a conflict naming what is left.

## An automations host's version never updates after enrollment

The host reports its version only when it enrolls, and the release build passes
no version to `build:binaries`, so the Automations hosts list keeps showing the
enrollment-time version (or `dev`) after `nexul upgrade`. Stamp the version into
the compiled binary and report it on each assignments poll.

## An automations host restarts its workers once after they first connect

The host restarts a worker whenever the automation's `updated_at` changes, and
the first connection itself updates it, so every worker starts twice about one
poll apart after the host boots. Harmless, but noisy in the logs; compare only
what a worker is built from (active version, config, token).

## Proxy-gateway exposures never get a route

An exposure on a reverse-proxy gateway expects Traefik's Docker provider to
route the target container by its labels, but nothing ever puts Traefik labels
on that container, and the runner's `docker run` cannot set labels. So a proxy
exposure records its hostname and DNS record while Traefik has no router for
it; only the instance's own domain routes, through the file provider (ADR 0077).
The fix is either label support in the runner (a stack field plus the assign
frame) or writing each exposure into the same file-provider config the gateway
container renders on start. Every router on the https entry point already
defaults to Let's Encrypt, so certificates follow once routers exist.

## A scoped token without docs:thread can open a doc's thread

The gateway derives a scoped token's scope from the path, so `POST /api/chat/docs/{docID}/thread` needs only `chat:write`, and the use-case then checks `docs:thread` against the token's creator, not the token. Add the route to `verbRouteScope` in `internal/integrations/gateway.go`, as the container logs routes are.

## Logs of a missing container loop as "Runner offline"

When a service's container does not exist, `docker logs` fails with a daemon error line that has no timestamp. The stream ends, and the web view and the phone reconnect. Each reconnect repeats the error lines, because deduplication keys on the timestamp. The marker also reads "Runner offline, reconnecting…", which is wrong: the runner is online and the container is missing. The server should send a distinct end reason for a missing container, and both clients should stop retrying on it and say so.

## A removed person's voice join token still works until it expires

Removal from a private voice channel disconnects the person, and they cannot get a new join token, but one they already hold stays valid for its six-hour lifetime, so a hand-built client could reconnect. LiveKit's RemoveParticipant accepts a token-revocation timestamp that would close this on servers new enough to support it; the other lever is a shorter token lifetime with refresh. Removal from the workspace and switching someone to restricted publish no channel membership change, so neither ends a call either.

## A doc's live replay skips updates that landed before another editor's snapshot

Replay sends only the updates after a room's newest snapshot. When one editor
commits a snapshot whose base predates another editor's updates, those
updates are not replayed to new joiners until their author commits again.
Seen while fixing the server-write reset (ADR 0109).

## Renaming a doc over MCP leaves the old title in open editors

A `doc_update` that changes only the title does not reset the live room (a
title-only edit is deliberately excluded from the reset), so editors already
open keep showing the old title until they reload.

## A doc's seed is not handed on when the seeder drops as a slow connection

Only the first editor of an empty room seeds it, and the seed passes to
another editor when the seeder leaves normally. When the hub drops the
seeder as a slow connection instead, nobody else is told to seed, so the
room can stay empty until someone reloads.

## A person's save changes the end of a note's file

Saving a note from the editor drops the file's final newline, or adds one
when the note ends in an image, so a note an agent wrote changes bytes on
its first human save even when nothing visible changed. Seen in the ticket
flow walkthrough; normalise the trailing newline in one direction.
