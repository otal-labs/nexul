# Architecture practices

The cross-cutting patterns every service and domain follows. Language-agnostic
where possible; the Go mechanics live in `practices/go.md` and the web client
in `practices/react-guide.md`.

## 1. Layered architecture per domain

A domain that owns entities, state transitions, and events lives in
`internal/<domain>/` with five files:

```
internal/<domain>/
  model.go          # domain types: structs, enums, value objects
  repo.go           # repository interface, implemented in internal/platform/storage/
  usecase.go        # the business rules; takes the repo and the bus as interfaces
  handler.go        # HTTP handlers, thin
  events.go         # the topics this domain publishes and the handlers for topics it consumes
```

A domain that exposes MCP tools adds `mcp.go` beside `handler.go`; both are
adapters over the same use-case (ADR 0019).

Why this shape:

- `model.go` is data plus validation and nothing else. It is the domain's
  vocabulary, so it imports nothing from the rest of the domain.
- `repo.go` declares the interface the domain needs. The implementation sits in
  `internal/platform/storage/`, so the domain never sees a concrete storage
  type and the storage engine can change under it.
- `usecase.go` is the only file that knows the rules. Handlers and MCP tools
  call it; nothing else does.
- `handler.go` and `mcp.go` parse input, call the use-case, and format output.
  A rule in a handler is a rule the other adapter does not have.
- `events.go` names every topic the domain publishes in one `Topics()`
  function. That is what makes the event catalog enumerable.

Dependency direction:

```
handler, mcp -> usecase -> repo (interface)
                        -> eventbus (interface)
                        -> model
```

Handlers never import storage. Use-cases never import handlers. Models import
nothing.

Which domains this covers, as the tree stands:

- Full five-file shape: `chat`, `codereview`, `deploy`, `dns`, `docs`,
  `memories`, `plays`, `runner`, `tickets`, `topology`, `workspace`.
- No `events.go` yet: `access`, `attachments`, `automations`, `connectors`,
  `integrations`, `pairing`, `roles`, `tenancy`. The file is added with the
  domain's first published event, never before, because an empty catalog entry
  is a contract nobody asked for.
- Thin variants: `mentions` and `repository` have no `repo.go` (they read
  through other domains or the git provider); `auth` has no `usecase.go`;
  `gitprovider` and `voice` have no `repo.go` but do publish events. Each
  follows the layers it has and grows into the full shape when it needs one.
- Protocol and infrastructure packages: `agent`, `collab`, `eventcatalog`,
  `harness`, `livekit`, `mcp`, `presence`, `t3client`, `platform`. Shaped by
  their protocol, not by this template. They still obey the dependency
  direction. `harness` declares the one `harness.Client` interface the server
  talks to every agent harness through, with one implementation per kind
  (`t3client` is the first).

## 2. The EventBus is the microservice seam

The `EventBus` interface in `internal/platform/eventbus/bus.go` is the only
way domains talk to each other asynchronously (ADR 0013). Domains never import
each other; they publish and subscribe.

- Today every domain runs in one process and the bus is the in-process
  implementation in `internal/platform/eventbus/inprocess/`. Zero network
  hops.
- When a domain moves to its own service, the bus implementation changes to a
  message broker and the domain code does not change. That day has not come,
  and no broker adapter exists in the tree; ADR 0013 records the intended one.

Read the interface and the `Event` envelope in `bus.go`, not from a copy here.
The two halves mean different things:

- `Publish` and `Subscribe` are fan-out. At-most-once is acceptable; the
  outbox (section 3) upgrades critical paths to at-least-once.
- `Enqueue` and `Consume` are competing consumers: durable, at-least-once
  worker dispatch.

Topic names are `<domain>.<entity>.<action>`, for example
`deploy.service.healthy`, `ticket.created`, `runner.build.completed`.

Each domain declares its topics in its own `events.go` `Topics()` function and
`internal/eventcatalog/catalog.go` aggregates them into the catalog of record.
The catalog is additive only: a topic, once published, is a contract
(ADR 0044). A topic missing from `Topics()` is invisible downstream.

## 3. Transactional outbox

Critical events (deploy status changes, ticket state transitions) go through
the outbox so a crash between commit and publish cannot lose them:

1. The use-case writes the domain change and an outbox row in the same SQLite
   transaction. Domains hand their repo `eventbus.OutboxEvent` values and
   import `internal/platform/eventbus`, never the `outbox` adapter package,
   because `outbox` imports `storage` and a domain importing it closes a cycle.
2. A background relay polls the outbox and publishes each row under the row's
   own ID as the bus event ID (ADR 0018). That is what makes a redelivered row
   dedupeable.
3. After a crash between commit and publish, the relay re-emits on restart.
4. Consumers are idempotent: they check the processed-events store before
   acting.

The tables, as created in the first migration:

```sql
CREATE TABLE outbox (
    id          TEXT PRIMARY KEY,
    topic       TEXT NOT NULL,
    payload     BLOB NOT NULL,
    created_at  INTEGER NOT NULL,
    published   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_outbox_unpublished ON outbox(published, created_at)
    WHERE published = 0;

CREATE TABLE processed_events (
    event_id    TEXT PRIMARY KEY,
    processed_at INTEGER NOT NULL
);
```

The bus never stores a bare `event_id`. Each subscriber wraps the store in a
scope that namespaces the key as `<consumer>:<event_id>`, so two subscribers
of one topic never see each other's records while a redelivery to the same
subscriber still dedupes. The consumer label is set explicitly in the
composition root and is stable across restarts; that is what lets outbox
redelivery dedupe survive a crash (ADR 0018).

## 4. Dead-letter queue

A message that exhausts its retries, or whose handler returns `ErrFatal`, is
written to `dead_letters` and acknowledged:

```sql
CREATE TABLE dead_letters (
    id          TEXT PRIMARY KEY,
    topic       TEXT NOT NULL,
    payload     BLOB NOT NULL,
    error       TEXT NOT NULL,
    attempts    INTEGER NOT NULL,
    created_at  INTEGER NOT NULL
);
```

The MCP server exposes `list_dead_letters` and `replay_dead_letter`, so an
agent can inspect and replay failures without a database shell.

## 5. Resilience middleware

Every subscription runs its handler inside the same chain, built in
`inprocess.Bus.addSubscriber`, innermost first:

1. Recover: a panic becomes an error instead of taking the process down.
2. Dedupe: the processed-events check under the subscriber's own scope. A
   seen event is acknowledged and skipped.
3. Retry: bounded exponential backoff with jitter (three attempts by default,
   100ms to 30s). `ErrFatal` skips the retries.
4. Dead-letter: on exhaustion or a fatal error, persist and acknowledge.

`Close` drains in-flight handlers under `DrainTimeout` (30s by default).

The chain is built per subscriber, not per bus, and that is deliberate
(ADR 0018): a chain shared across subscribers made the first subscriber to run
record the event ID and every other subscriber of the topic skip it silently.
`SubscribeWithConsumer` supplies the stable consumer identity the dedupe scope
is keyed on.

There is no throttle and no per-message timeout middleware. Handlers bound
their own work with the context deadline they derive (see `practices/go.md`,
concurrency), and back-pressure comes from the single SQLite writer.

## 6. Idempotency

Two layers, and both are required:

1. The dedupe middleware skips an event ID the subscriber has already
   processed. The key is `<consumer>:<event_id>`, so a fanned-out event is
   processed once per subscriber and a redelivery to the same subscriber
   (outbox re-emit, dead-letter replay) is skipped.
2. The handler's own mutation is idempotent: an upsert, or "set state to X",
   never "increment". With SQLite's single writer the dedupe check and the
   mutation share one transaction, so they cannot race.

The first layer is an optimisation; the second is what makes at-least-once
delivery safe when the first misses.

## 7. Graceful shutdown

Every service:

1. Catches `SIGTERM` and `SIGINT`.
2. Cancels the root context.
3. Stops accepting new work: HTTP `Shutdown`, the WebSocket server stops
   accepting, the bus stops pulling.
4. Drains in-flight work under a deadline (30s by default).
5. Nacks unfinished messages so they redeliver on the next boot. Safe because
   handlers are idempotent (section 6).
6. Closes the database, flushes logs, exits 0.

## 8. The MCP server is one adapter of two

The MCP server in `internal/mcp/` and the HTTP gateway are two adapters over
the same use-case layer (ADR 0019). Every tool maps to a use-case function;
every resource maps to a read query. So:

- A capability in the UI exists in MCP by construction, and the reverse.
- Operational internals (dead letters, run logs, topology mutation) are
  exposed to agents because they are just use-cases.

Tool names are `<verb>_<object>`: `search_docs`, `replay_dead_letter`,
`create_ticket_from_doc`, `deploy_stack`. Verb first matches how agents read a
tool list and how the wider MCP ecosystem names tools, so an agent's prior
carries over. Composite tools that cross domains live in
`internal/mcp/registry.go`; domain-owned tools live in the domain's `mcp.go`.

Domain errors map to JSON-RPC codes in `internal/mcp/errors.go`:

- `ErrNotFound` to `-32002`
- `ErrUnauthorized` to `-32001`
- internal errors to `-32603`

## 9. The canvas JSON is the stored topology

The React Flow `nodes` and `edges` the browser serialises is the topology
the backend stores as a JSON blob and the MCP server mutates through tools.
There is no second hand-written topology type (ADR 0033). The stored JSON holds
nodes, edges, and positions only; the network boxes, gateway rows, and
hostname pills the canvas draws are derived at render time and never
persisted. The schema carries `schema_version` so it can migrate.

## 10. Git provider abstraction

`internal/gitprovider/provider.go` declares the interface; GitHub is the
implementation in `internal/gitprovider/github/`:

```go
type GitProvider interface {
    GetRepo(ctx context.Context, owner, name string) (*Repo, error)
    ListPRs(ctx context.Context, owner, name string, opts PROpts) ([]*PR, error)
    GetPR(ctx context.Context, owner, name string, number int) (*PR, error)
    CreateWebhook(ctx context.Context, owner, name string, cfg WebhookConfig) (string, error)
    DeleteWebhook(ctx context.Context, owner, name, hookID string) error
    ListInstallationRepos(ctx context.Context) ([]*Repo, error)
    GetTree(ctx context.Context, owner, name, ref string) ([]TreeEntry, error)
    GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error)
}
```

Adding another host is a new package behind the same interface. Nothing
outside `gitprovider` imports the GitHub package.

## 11. Runner WebSocket protocol

The runner connects out to the server over one WebSocket (ADR 0031); the
server never dials the runner, so a runner behind NAT works. Frames are JSON
with a `type` discriminator, declared in `internal/runner/protocol.go`:

- Server to runner: `assign_build`, `assign_deploy`, `cancel`.
- Runner to server: `heartbeat`, `build_progress`, `build_result`,
  `deploy_log`, `deploy_progress`, `deploy_result`.

The server-side handler translates runner frames into bus events. It is a
producer on the bus, not the bus.

## 12. Single-tenant per install

One installation holds one owner's data (see `CONTEXT.md`, Instance). No
tenant IDs in queries,
no row-level security, no cross-tenant leakage to reason about. Anyone who
needs isolation runs a second instance.

## 13. Integrations are external services

Third-party integrations extend the product without running inside it
(ADR 0043):

- An integration is its own service. The server never loads third-party code,
  so an integration only knows what its scoped token allows.
- Outbound webhooks are one more subscriber: the catalog fans out to registered
  URLs as HMAC-signed POSTs through the outbox, retry, and dead-letter path.
  No new machinery.
- Integrations call the same HTTP gateway the browser uses, authenticating
  with a scoped, revocable API token instead of a session (ADR 0019). Not a
  third adapter.
- The contracts are the product: the versioned event schemas (ADR 0044) and
  the generated OpenAPI document (ADR 0045). The SDK derives from them, which
  is what lets an integration be written in any language.
- Trust is tiered. Registry entries are `verified` or `community`. Least
  privilege scopes, signed webhooks, revocable tokens, and an audit log are
  the baseline.
