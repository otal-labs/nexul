# 04 findings: runPlay in the SDK

## How the SDK reaches the server today

Two channels, and only one of them carries actions.

- The dial-in WebSocket carries event delivery down and run reports up,
  nothing else. Frame types are `announce`, `hello`, `event`,
  `run_started`, `run_log`, `run_finished`, `run_crashed`
  (`sdk/src/protocol.ts:6-13`, mirrored by hand in
  `internal/automations/protocol.go:13-27`; one flat `Frame` struct,
  `protocol.go:36-51`). `DialinClient.runHandler` builds the ctx and calls
  the handler (`sdk/src/client.ts:153-184`, `buildCtx` at 169).
- Every action goes over HTTP through `ctx.api`, an `ApiClient` holding the
  automation's token (`client.ts:82`, `sdk/src/api-client.ts:41-58`). The
  shipped default `ticket-finished` already acts this way:
  `ctx.api.request("PATCH", "/api/tickets/${id}/status", …)`
  (`automations/defaults/ticket-finished/index.ts:27`).
- On the server, `RequireAutomation` accepts any `dat_` token on `/api`,
  checks the route against the token's scopes, and sets the actor to the
  automation's creator with an `AutomationRef` (`internal/automations/gateway.go:41-67`).
  A token acts as its creator, held to both its scopes and the creator's
  grant; a shipped default has no creator (ADR 0087, lines 32-33).
- Host workers use a host-scoped `dat_h.` token (`host_token.go:11`,
  checked in `hosts_usecase.go:292-326`) that passes the same gate. The host
  only spawns workers with it (`automations/src/supervisor.ts:80-97`,
  `worker-entry.ts:39-45`) and never sees what a handler calls.

So a new call is added end to end as: an `/api` route and use case on the
server, a scope mapping in the gateway, a method in the SDK. No frame, no
host change. The header comment in `api-client.ts:1-6` (the `dat_` token
"only ever authenticates the WS dial-in endpoint") is stale and contradicts
`gateway.go:41`; fix it in the build.

## Where the play-name union comes from

- `events.generated.ts` is generated at SDK build time from the Go catalog
  (`sdk/tools/generate-events.ts:12`, `Topic` at `events.generated.ts:138`).
  That works because topics are code. Play labels are workspace data, so
  nothing built into the SDK package can know them.
- There is no automation code editor in the web app. `rg` for monaco,
  codemirror, `addExtraLib` or `.d.ts` loading in `web/` finds nothing; the
  automation page shows code read-only in a `<pre>`
  (`web/src/components/automation/AutomationVersionDiff.tsx:51`). Authors
  write code in their own editor in a project made by `nexul init` and
  upload it with `nexul push` (`sdk/bin/cli.ts:79-91`, `141-162`). "The
  editor" is therefore the author's local TypeScript, and the union has to
  land in their project as a file.
- `nexul push` bundles with `Bun.build` (`cli.ts:148`), which strips types
  and never type-checks. A typo is caught in the editor or by `tsc`, not by
  push, unless push is taught to check (see risks).
- Play labels are not unique in a workspace. `Play.Validate` only requires a
  non-empty label (`internal/plays/model.go:196-199`), and the only unique
  index is on `builtin_key` (`migrations/0055_instance_templates.sql:21`).
  A rename is an ordinary `Update` that emits `play.updated`
  (`internal/plays/usecase.go:215-237`, `events.go:15-17`).

## What a run checks today, and whose permission

- `POST /api/plays/{id}/run` (`internal/plays/run_handler.go:37`, `113-129`)
  calls `Runner.Run`, whose starter is the context actor
  (`run.go:321-341`). For an automation that is the creator; a default
  automation has no creator and is refused as unauthenticated (`run.go:323-325`).
- `checkPlay` requires the play in the target's workspace, enabled, project
  membership, not excluded, the starter's `plays:run` on that play, and for
  a ticket play the ticket in the play's one stage (`run.go:761-785`).
- The gateway maps that route to `plays:write`, not `plays:run`: there is no
  `verbRouteScope` entry for it (`internal/integrations/gateway.go:78-94`),
  so it falls through to `<domain>:<method action>` (`gateway.go:106-124`).
  A token granted only `plays:run` (a real scope, `permissions.go:71,136`)
  cannot run a play. Worth fixing alongside.
- The run lands on the starter's own harness (`run.go:353`); there is no way
  today to run on someone else. Trails record `Via` web or mcp only
  (`model.go:84-85`).

## Recommendation

### Signature

`runPlay` lives on the handler context, ticket plays only, ticket by id
(event payloads type `ticket` loosely, so authors already narrow the id the
way `ticket-finished` does).

```ts
// sdk/src/context.ts
// Filled per workspace by `nexul types`; empty means "not generated yet".
export interface PlayNames {}

type Known = [keyof PlayNames] extends [never] ? Record<string, "ticket"> : PlayNames;
export type TicketPlay = { [K in keyof Known]: Known[K] extends "ticket" ? K : never }[keyof Known] & string;

export interface RunPlayOptions {
  runOn?: "developer" | "tester"; // default developer
  priority?: "high" | "normal" | "low"; // default normal
}

export interface QueuedRun {
  id: string;
  state: "queued" | "didnt_run" | "paused";
  reason?: string;
}

export interface Ctx<S extends ConfigSchema = ConfigSchema> {
  // api, config, secrets, log as today
  runPlay: (play: TicketPlay, ticketId: string, opts?: RunPlayOptions) => Promise<QueuedRun>;
}
```

`buildCtx` adds `runPlay` as a thin wrapper over
`api.request("POST", "/api/plays/queue", { play, ticket_id, run_on, priority })`.
On a non-2xx it throws `ApiError` with the server's error text in the
message (today's message is only the status, `api-client.ts:55`, which would
leave run history reading "failed: 404").

Before any declaration exists, `PlayNames` is empty and the parameter falls
back to `string`, so a fresh project compiles. Once generated, a typo, a
renamed play, or a doc play fails the type check. The value carries the
play's type so only ticket plays are accepted.

### Producing and refreshing the union

A per-workspace declaration, merged by module augmentation, written by the
CLI into the author's project:

```ts
// src/plays.generated.d.ts — generated by `nexul types`, do not edit
export {};
declare module "@nexul/sdk/context" {
  interface PlayNames {
    "Fix with AI": "ticket";
    "To tickets via AI": "doc";
  }
}
```

- New command `nexul types <automation-id>`: `GET /api/automations/{id}`
  for its `workspace_id`, then `GET /api/workspaces/{ws}/plays`
  (`internal/plays/handler.go:35`) with the PAT from `nexul.config.json`;
  writes every play's label and type, keys through `JSON.stringify`. A label
  held by two plays is left out with a printed warning (it would be a
  duplicate property, and the server refuses it anyway).
- `nexul push <id>` runs the same refresh before bundling, so every push
  picks up renames. `nexul dev` takes no id and stays as it is.
- The `init` scaffold starts with
  `/// <reference path="./plays.generated.d.ts" />` so the file is part of
  the program even without a tsconfig; `Bun.build` ignores the line. The
  file must be a module (`export {}`) or `declare module` would shadow the
  SDK module instead of augmenting it.
- Refreshing is pull-only. A rename after a push cannot reach code already
  deployed; that is the stale case below.

### Stale or ambiguous name at run time

The server resolves the label at call time in the ticket's workspace (and,
for an automation caller, refuses a ticket outside the automation's own
workspace as not found):

- no play with that label: `ErrNotFound`, "no play named "X" in this
  workspace";
- two plays with that label: `ErrInvalid`, "two plays are named "X"; rename
  one".

`runPlay` throws, the handler crashes unless the author catches it, and the
run's history shows the message (`client.ts:182`). Nothing is queued. The
host needs no change.

### Protocol

No new frame. `runPlay` is an API call like every other action, through the
existing gateway, so it inherits revocation, host-token checks, scopes and
audit attribution for free. `protocol.ts` and `protocol.go` stay as they are.

One new route, mounted with the run routes under `/api/plays`:
`POST /api/plays/queue`, body `{ play, ticket_id, run_on?, priority? }`,
answering 202 with the queue entry. Registered in `server/cmd/routes.go`
for the OpenAPI document (ADR 0045), with a `verbRouteScope` entry mapping
it to `plays:run`. Add the same entry for `POST /api/plays/{id}/run`.

### Permissions

1. The token's scopes include `plays:run` (gateway).
2. The automation's creator holds `plays:run` on that play and can see the
   ticket's project: the existing `checkPlay` with the creator as starter,
   at enqueue time. A token never does more than its creator could.
3. The person the run lands on holds `plays:run` and project access,
   checked when the entry reaches the front, as for an auto play (excluded
   gives "didn't run").

A default automation has no creator and cannot call it; no default needs to.

### Queue and run on

`runPlay` is an automatic run: it enqueues into the same per-person queue as
an auto play (ticket 10), never starts a turn directly.

- `runOn` is `developer` (default) or `tester`, read from the ticket's
  `Developer`/`Tester` (`internal/tickets/model.go:41-42`). "Whoever caused
  it" has no meaning for code, so it is not offered.
- `priority` is the auto play's High/Normal/Low, default Normal.
- It counts toward the per-ticket daily cap and the one-auto-run-per-ticket
  limit; automations can loop too (an automation on `play.run_finished`
  calling `runPlay`).
- At enqueue: play found, enabled, not excluded, ticket in the play's stage,
  creator allowed. These throw, so the author sees a bad call in run history.
  At the front the queue re-checks stage, play and the person, and skips
  with the muted ticket line if they no longer hold.
- The result is the entry: `queued`, `didnt_run` with a reason (no
  developer, person excluded), or `paused` (ticket at its cap). None of
  these throw: they are outcomes, shown on the ticket like an auto play's.
- The trail gets a new `Via` value, `automation`, for provenance.

### Mock context and docs

- `createMockContext` builds its own object today (`sdk/src/testing.ts:46`).
  Switch it to `buildCtx(...)` plus `calls` and `logs`, so it gets
  `runPlay` for free, recorded like any call:
  `{ method: "POST", path: "/api/plays/queue", body: { play, ticket_id, … } }`.
  Seed a default response for that key, `{ id: "mock-run", state: "queued" }`,
  so a test reading `.state` works without scripting; `responses` still
  overrides it.
- `nexul dev` then lists the call under "Would-have-called API" unchanged
  (`cli.ts:137-138`); tests compile against the same declaration.
- The guide page `website/src/content/docs/docs/guide/automations.md` gains
  a "Start a play" section: `ctx.runPlay`, `nexul types`, `runOn` and
  `priority`, the queue and cap, and what a renamed play does to deployed
  code. Its `ctx` sentence (line 46) and the testing line (line 53) name
  `runPlay`.
- Not added: an MCP tool for queuing; agents already have `play_run`.

## Open risks

- Duplicate labels. Nothing stops two plays sharing a label, so a name is
  not an identity. The first cut refuses at run time and skips in the
  declaration. Making labels unique per workspace is a schema change on
  live data (a forward migration that renames existing duplicates); the
  owner decides whether that is wanted.
- Type checks are advisory. `nexul push` does not type-check, so a typo can
  be pushed and fail at run time. Push could run `tsc --noEmit` when the
  author's project has TypeScript; that adds a step the scaffold does not
  set up today (no tsconfig, no package.json). Left out of the first cut.
- Deployed code goes stale on rename. Only run history shows it. A later
  signal could list automations whose active code mentions the old label
  on the play's settings page when it is renamed; not in the first cut.
- The augmentation is not yet compiled against the SDK's `exports` map;
  the build ticket proves it with a type test (a typo and a doc play fail).
- Fixing the run route's scope to `plays:run` changes what existing tokens
  can do: a token holding `plays:write` but not `plays:run` loses the run
  route. Check the live tokens' scopes before changing it, or grant both
  for one release.
- Ticket 10 owns the queue; if it lands its own enqueue route, `runPlay`
  reuses it with the label lookup added rather than adding a second.
