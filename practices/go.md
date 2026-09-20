# Go practices (Nexul)

The binding Go standard for this repository. `AGENTS.md` routes every Go
task here; read it before the first edit. Each rule carries the failure it
prevents, so it can be extended to cases this file does not list.

---

## 1. Project layout

```
server/
  cmd/main.go            # server binary entry point
  Dockerfile             # multi-stage (release + debug targets)
runner/
  cmd/main.go            # runner binary entry point
  Dockerfile
internal/
  access/                # permission overwrite table: allow/deny per resource and user
  agent/                 # agent turn pipeline, T3 backend adapter
  attachments/           # attachment domain, bytes stored in SQLite
  auth/                  # sign-in, provider OAuth clients, sessions
  automations/           # automation dial-in and defaults
  chat/                  # chat domain: conversations, messages, unread state
  codereview/            # code review linking domain
  collab/                # collaborative editing session hub
  connectors/            # per-connector OAuth app config and credentials
  deploy/                # deploy engine domain
  dns/                   # DNS adoption and exposure domain
  docs/                  # docs domain (use-cases, handlers)
  eventcatalog/          # aggregates every domain's published event topics
  gitprovider/           # git provider interface + GitHub impl
  harness/               # harness.Client interface, one implementation per agent harness kind
  integrations/          # external service integrations, scoped tokens
  livekit/               # minimal stdlib LiveKit client
  mcp/                   # MCP server adapter (tools/resources/prompts)
  memories/              # per-project agent memory domain, own entity from docs
  mentions/              # @-mention and chip resolution
  pairing/               # device pairing exchange
  platform/              # cross-cutting infrastructure (shared by all services)
    eventbus/            # EventBus interface, in-process bus, outbox relay, dead letters
    storage/             # SQLite engine, migrations, repo interfaces
    logging/             # slog setup, ctx-logger, trace_id propagation
    config/              # env-based config loading
  plays/                 # plays domain: user-fired Agent turn definitions and their runs
  presence/              # keeps a harness WebSocket open per paired computer
  repository/            # repo scanning for the project wizard
  roles/                 # role entity, permission catalog, protected Owner role
  runner/                # runner protocol domain (WS server side)
  t3client/              # T3 Effect RPC client over one WebSocket
  tenancy/               # workspace entity and membership
  tickets/               # tickets domain
  topology/              # topology model and canvas JSON schema
  voice/                 # voice channel domain
  workspace/             # projects, repositories, ticket moves, notifications inbox
web/                     # React frontend (see practices/react-guide.md)
sdk/                     # automation SDK (see practices/typescript.md)
automations/        # automation runtime host (see practices/typescript.md)
desktop/                 # Electron desktop shell (see practices/typescript.md)
```

### Seam rules (ADR 0017)

- `internal/platform/*` is imported by any domain or service.
- `internal/<domain>/*` is imported only by the service(s) that own it and by
  the MCP adapter. Domains do not import each other directly. They
  communicate via the `EventBus` interface.
- `server/cmd` and `runner/cmd` wire dependencies (composition root). They
  import domains and platform; domains never import cmd.
- `internal/<domain>/repo.go` declares the interface only; its implementation
  lives in `internal/platform/storage/`. Domains never import a concrete
  storage type. They depend on the interface (dependency inversion; the
  adapter stays in the outer layer).
- No circular imports. A circular import means the code needs a new platform
  package or an interface inversion.

---

## 2. Error handling

### Sentinel errors + wrapping

- The cross-cutting sentinels live in `internal/platform/errors`, imported as
  `apperrs` so callers can still import stdlib `errors`. Add one there, never
  a per-domain copy.
- Return sentinel errors from repos and use-cases; wrap with context at the
  boundary: `fmt.Errorf("get ticket %s: %w", id, err)`.
- Adapters are the only layer that translates them. `internal/platform/httpx`
  owns the status codes and the JSON error envelope every HTTP gateway
  returns; `internal/mcp/errors.go` owns the JSON-RPC mapping. A handler that
  writes its own status for a sentinel has put the mapping in two places.
- **Never** call `log.Fatal` or `os.Exit` outside `cmd/main.go`.
- **Never** swallow an error with `_ = someFunc()`. If ignoring an error is
  intentional, add a comment explaining why.

### Retryable vs Fatal (event handlers)

```go
var (
    ErrRetryable = errors.New("retryable")
    ErrFatal     = errors.New("fatal")
)

func Retryable(err error) error { return fmt.Errorf("%w: %w", ErrRetryable, err) }
func Fatal(err error) error     { return fmt.Errorf("%w: %w", ErrFatal, err) }
```

Handlers return `ErrRetryable` for transient failures (network blip, lock
contention) and `ErrFatal` for permanent ones (bad payload, schema mismatch).
The bus middleware checks `errors.Is` and routes accordingly.

---

## 3. Structured logging (slog)

- Use `log/slog` (the standard library, Go 1.21+). slog is the only logging
  API application code calls.
- Every request or message handler creates a child logger with `trace_id` and
  domain-specific fields via `slog.With(...)`.
- Pass the logger through `context.Context` using a typed key:

```go
type ctxKey struct{}

func FromCtx(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}

func CtxWithLogger(ctx context.Context, l *slog.Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, l)
}
```

- `trace_id` is a UUIDv7 minted at the entry point (HTTP middleware, WS frame
  handler, MCP request handler) and carried in the context from there. It is
  time-sortable, so a log store can order records without a clock.
- Log at the right level: `Debug` for dev tracing, `Info` for lifecycle
  events, `Warn` for recoverable issues, `Error` for things that need
  attention.
- Never log secrets, tokens, or PII. Use redaction middleware if needed.
- The OpenTelemetry slog bridge is a sink beneath slog, never a second
  logger. `logging.NewWithOTLP` fans records out to the log store when one is
  configured and never blocks or fails the caller on it. ADR 0008 is why the
  bridge exists: the log store receives records over OTLP, so a handler
  underneath slog is how they reach it without giving application code a
  second logging API to call.
- stderr is always the primary sink. The browser posts its console errors to
  `POST /api/logs` and the server relays them under `source=web`. Nothing
  outside the server talks to the log store.

---

## 4. Configuration

- All config comes from environment variables, loaded once at startup in
  `cmd/`.
- Use a typed struct with `env` tags, or manual `os.Getenv` calls plus
  validation.
- No global mutable config. Pass config structs into constructors.
- Secrets (DB path, tokens) are env vars or mounted files. Never hardcode
  them and never commit them to source.
- Nothing is required at install time (ADR 0020). Every field either has a
  working default, is generated and persisted on first start, or is
  collected by the setup wizard into the database. Adding a required env
  var is a regression.
- Config structs implement `Validate() error`: required fields are checked
  for empty values, addresses parse as `host:port`, and invalid enum-like
  values (for example `LOG_LEVEL`) fail with a clear message instead of
  silently defaulting. Call `Validate()` in `cmd/` at startup and fail fast,
  because a bad config that starts anyway surfaces as a confusing runtime
  failure far from its cause instead of a clear one at boot.

---

## 5. Dependency injection

- Constructor injection. Every domain type has a `New(...)` that takes its
  dependencies as interface parameters.
- No global singletons. No `init()` side effects (except registering blank
  imports like `_ "modernc.org/sqlite"`).
- The composition root (`cmd/main.go`) wires everything. If wiring gets long,
  extract a `wire.go` or `bootstrap.go` in the same package. Do not
  introduce a DI framework.

---

## 6. Interfaces

- Define interfaces at the consumer side, not the provider side. The
  `EventBus` interface lives in `internal/platform/eventbus/` because that
  is where the consumers (domains) import it.
- Keep behavior interfaces small, three methods or fewer. Split an interface
  once it grows past that.
- Storage repos are exempt from the size rule. One `Repo` per domain
  aggregate is the pattern (`internal/tickets/repo.go`,
  `internal/docs/repo.go`). It carries every query its use-case needs. Past
  five methods, compose the repo by embedding small per-aggregate interfaces
  instead of one flat method list. Use-cases that need only a slice take the
  narrow interface.
- Accept interfaces, return structs.

---

## 7. Concurrency

- Channels transfer ownership. Mutexes protect state.
- Always pass `context.Context` as the first parameter to any function that
  does I/O, sleeps, or blocks.
- Every inbound handler (HTTP, WS, MCP) derives a `context.WithTimeout` (or
  `WithDeadline`) before doing work, and every outbound call (HTTP client,
  DB, bus publish) honors that deadline. No unbounded waits. A hung handler
  is a bug.
- Use `errgroup` for parallel work that needs cancellation; plain
  `sync.WaitGroup` covers the simple cases. If you introduce `errgroup`, add
  `golang.org/x/sync` to `go.mod`; it is not a direct dependency today.
- `sync.WaitGroup.Go` (Go 1.25) replaces the hand-rolled
  `wg.Add(1); go func() { defer wg.Done(); ... }()` pair for new code. One
  call starts the goroutine and registers it with the group, so the two
  steps can no longer drift out of sync.
- Never launch a goroutine without a clear shutdown path. Every `go func()`
  must be joinable, via `errgroup.Wait()`, a `done` channel, or context
  cancellation.
- Every test that needs a context calls `t.Context()` instead of building
  its own `context.Background()`. It cancels automatically once the test
  completes, before cleanup functions run, so a test can never outlive the
  cancellation it depends on.
- Tests of time-dependent goroutine code use `testing/synctest` instead of
  sleeping and polling for a result. `synctest` gives the test a virtual
  clock, so time advances deterministically instead of racing the real
  clock. This is the Go-native form of `practices/borrowed-practices.md`'s
  rule to test async code by draining, not sleeping.
- `encoding/json/v2` stays out of this codebase while it is gated behind
  `GOEXPERIMENT=jsonv2`; its API is not final until it ships without the
  flag.
- SQLite writes are serialized through a single write channel or a mutex in
  the storage layer. Do not fire concurrent write transactions from multiple
  goroutines without the serializer.
- `storage.OpenDB` keeps a pool of 8 with `SetConnMaxLifetime(5min)` over
  WAL and `busy_timeout(5000)` pragmas. A single connection turns one stuck
  statement into an outage, and the lifetime cap recycles stale WAL
  snapshots. Writes stay serialized in the storage layer; do not raise the
  pool without a written rationale.

---

## 8. Naming

- Packages: short, lowercase, no underscores (`eventbus`, not `event_bus`).
- Exported names: `CamelCase`. Unexported: `camelCase`.
- Interfaces: name by behavior (`Reader`, `TicketStore`), not by
  implementation (`SQLiteTicketStore`).
- Receiver names: a one or two letter abbreviation of the type (`s` for
  `*Server`, `r` for `*Router`). Keep it consistent within a file.
- Test helpers: `mustXxx` for functions that call `t.Fatal` on error.

---

## 9. Early return, no else

```go
func (s *Service) GetTicket(ctx context.Context, id string) (*Ticket, error) {
    ticket, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get ticket: %w", err)
    }
    return ticket, nil
}
```

No `else` after a return. No `else if` chains; use sequential `if`
statements with early returns. This keeps the happy path unindented and the
error paths at the top.

---

## 10. Testing (Go)

See `practices/testing.md` for the philosophy. These are the Go-specific
rules.

### Table-driven tests

```go
func TestParseDeployStatus(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    DeployStatus
        wantErr bool
    }{
        {"valid healthy", "healthy", DeployStatusHealthy, false},
        {"valid running", "running", DeployStatusRunning, false},
        {"invalid", "bogus", "", true},
        {"empty", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseDeployStatus(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Testify

- Use `testify/require` for preconditions; it stops the test on failure.
- Use `testify/assert` for checks where seeing every failure in one run
  matters.
- Never call `t.Errorf` directly when `assert` or `require` gives a better
  message.

### Test file placement

- Unit tests: same package, `foo_test.go` next to `foo.go`.
- Integration tests: `integration_test.go` in the package, gated by the
  `//go:build integration` build tag or a `testing.Short()` check.
- Test helpers: a `testutil/` package, or exported helpers in `_test.go`
  files inside a `test` sub-package.

### Coverage

- CI enforces 80% line coverage as a hard gate; see `practices/testing.md`
  for the philosophy.
- Run locally: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
- `make coverage` filters the coverage profile before computing the
  percentage, dropping `cmd/`, `testutil/`, `node_modules/`, and generated
  code matched as `*.pb.go`, `*.connect.go`, `zz_generated*`, and
  `sqlcgen/`.
- Packages with no executable statements, such as wire-type-only packages,
  never appear in the coverage profile at all, so they never count against
  the denominator.
- The coverage script runs `go test` with `-race` over every package
  (`make coverage`); the separate `go build ./...` step in CI does not
  use `-race`.

### Static analysis

- `go vet ./...`, `golangci-lint` with the config in `.golangci.yml`, and
  `govulncheck` are the three hard gates. All three run in CI on the Go job
  and locally through `make lint` and `make vuln`.
- golangci-lint runs `errcheck`, `govet`, `staticcheck`, `ineffassign`, and
  `unused` as linters and `gocyclo` at complexity 15, plus `gofmt` and
  `goimports` (module path as the local prefix) as formatters.
- Fix all findings before opening a PR. No exclusions without a written
  rationale.

### Integration tests

- Use a real SQLite in-memory or temp-file database (not mocks) for repo
  tests.
- Use `t.Cleanup` to tear down.
- Use `t.Parallel()` where safe; in-memory SQLite per test is safe.

---

## 11. Imports

Group in three blocks, blank line between:

1. Standard library
2. Third-party
3. Internal (`github.com/otal-labs/nexul/internal/...`)

golangci-lint enforces this grouping through its goimports formatter,
configured with the module path as the local prefix. No dot imports. No
blank imports except for driver registration (`_ "modernc.org/sqlite"`).

---

## 12. Comments

- Exported symbols get a doc comment starting with the symbol name. One
  line.
- No inline comments that restate the code. Comments state why, not what,
  and stay to one line. A why that needs more than one line becomes an ADR
  in `docs/adr/` if it is a decision, and otherwise belongs in the feature's
  spec under `.scratch/`.
- No change-history comments ("previously did X", "switched from Y") and no
  commented-out code; git history holds both.
- No TODO comment without a tracking issue reference: `// TODO(#42): ...`

---

## 13. go.mod hygiene

- Minimal direct dependencies. Every `require` must earn its place.
- Before adding a dependency, check whether the standard library already
  covers it: `slices` and `maps` shipped in Go 1.21, the `net/http`
  method-and-wildcard router shipped in Go 1.22, and range-over-func
  iterators (`iter.Seq`) shipped in Go 1.23. The `go` directive in `go.mod`
  is the floor, and the `toolchain` line pins the exact release every build
  uses.
- Run `go mod tidy` before committing.
- Pin to semantic versions; avoid `+incompatible` unless unavoidable.
- Tools pinned for the repo, such as `sqlc` and `govulncheck`, are declared
  as `tool` directives in `go.mod` and run through `go tool <name>`, rather
  than `go run pkg@version` or a Makefile variable. A tool directive is
  checksummed in `go.sum` like any dependency, so the dependency bot tracks
  and bumps it the same way it tracks a library.

---

## 14. SQL queries (sqlc)

Repos in `internal/platform/storage/` do not hand-write column lists,
placeholder rows, and argument lists that have to be kept in sync by eye.
Queries live in `internal/platform/storage/queries/<table>.sql`; `sqlc`
generates `internal/platform/storage/sqlcgen/` (committed, coverage-exempt)
from them against the migrations in `internal/platform/storage/migrations/`.
`deploys_repo.go` is the reference shape.

- Schema source is the migrations directory as-is. Adding a column is a new
  migration plus one edit in the `.sql` query, then `make sqlc`. CI runs
  `sqlc vet` and `sqlc diff` and fails when the generated code is stale.
- The domain repo interfaces (`internal/<domain>/repo.go`) are the contract.
  The `*Repo` structs stay as thin wrappers over `sqlcgen.Queries`: they map
  domain types to `Params` structs, convert rows back (`toDeploy`), wrap
  errors with context, and keep the `WithTx` and outbox pattern. Writes go
  through `r.q.WithTx(tx)` inside `r.w.WithTx`.
- Use `:one` for single-row reads, `:many` for lists, `:exec` for writes,
  and `:execrows` when the caller needs the affected-row count (a missing
  row returns `ErrNotFound`). `SELECT *` is fine; sqlc expands it at
  generation time.
- Nullable columns come out as `sql.NullString` and similar types. Convert
  at the wrapper; `nullString` wraps a required lookup key for a nullable
  column. Map `sql.ErrNoRows` with `notFoundIfNoRows`; constraint violations
  go through `classifyWriteErr`.
- Runtime `IN (...)` lists are `sqlc.slice('ids')`; keep the caller's
  empty-input early return. A parameter used more than once in a query is
  `sqlc.arg(name)` in every position it appears, because SQLite binds
  positional `?` placeholders in the order they appear, and a repeated bare
  `?` would double-count the value instead of reusing it. A parameter used
  once may stay a bare `?`; single-use `?` placeholders coexist with named
  `sqlc.arg` elsewhere in the same query, exactly as
  `internal/platform/storage/queries/access.sql`'s `CountOverwriteAllowAny`
  does, and it passes `sqlc vet`. Query names share one namespace across
  `queries/`, so prefix each with its entity.
- A bare aggregate (`SELECT MAX(x)`) is typed `any` because an empty set
  yields NULL; read it with `optionalInt`.
- Queries sqlc cannot express (table or column names chosen at runtime,
  optional `WHERE` clauses assembled in Go) stay hand-written next to the
  wrapper with a one-line `// hand-written: sqlc cannot express ...`
  comment. Do not contort the SQL to fit.
- Generated types use primitives (`string`, `int64`). Do not add sqlc type
  overrides pointing at domain packages; the wrapper is where the
  conversion belongs.

---

## 15. Library choices

These choices are settled. Reopen one only with a written reason.

- `modernc.org/sqlite` is the SQLite driver because it is pure Go, which the
  `CGO_ENABLED=0` static build requires.
- `coder/websocket` is the WebSocket library because it is the maintained
  successor of `nhooyr.io/websocket`, the same project under new
  maintainers rather than a fork.
- `google/uuid` generates every ID, including the UUIDv7 `trace_id` from
  section 3, because it already covers UUIDv7 generation correctly.
- `testify` is the assertion library across the test suite.
- `goldmark` renders markdown; it is the CommonMark-compliant standard
  choice for Go.
- `go.yaml.in/yaml/v3` is the YAML library, the maintained successor path
  now that the original `go-yaml` repository stopped taking changes.
- OpenTelemetry appears only as the log exporter sitting beneath slog
  (section 3, ADR 0008), never as a second logging API.

No version numbers here; `go.mod` is the manifest of record.
