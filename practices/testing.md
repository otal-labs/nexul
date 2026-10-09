# Testing practices (Nexul)

The testing standard for every language in the repository. Language-specific
mechanics live beside the code they test: `practices/go.md` section 10 for Go,
the Testing (frontend) section of `practices/react-guide.md` for the web app,
and the testing sections of `practices/native.md`, `practices/typescript.md`
and `practices/mcp.md`.

---

## 1. The coverage number is a floor, not a goal

CI enforces 80% line coverage as a hard gate and treats 90% as the team
target. Coverage is a minimum hygiene check, not a quality metric. A
codebase with 95% coverage and no error-path tests is worse than one with
70% coverage and every error path exercised.

### What the gate actually enforces

- 80% line coverage across the measured scope (see exemptions below).
- Branch coverage on the resilience middleware (retry, dedupe, dead-letter
  routing). These are the paths that break in production.
- Error-path tests for every use-case that returns an error. If a function
  has `if err != nil`, a test hits that branch.

### What the gate exempts

- Go coverage drops paths containing `/cmd/`, `/testutil/`, `/sqlcgen/`, or
  `/t3rpctest/` (the fake T3 server the T3 clients' tests share).
- Pure wire types (structs with no methods, no validation) never appear in
  the coverage profile at all, since `go test` only emits statements for
  executable code.
- Frontend: `components/ui/` (shadcn primitives, tested by their own
  suite), `lib/utils.ts`, test setup and config files, and `main.tsx`, as
  listed in `web/vitest.config.ts`.

### Why not 100%

Chasing 100% teaches the wrong habit. Engineers end up testing getters, DTO
serializers, and `switch` default branches that can never fire. That time
is better spent on property-based tests, integration tests, or reviewing
the error paths that actually matter.

---

## 2. The test pyramid (our shape)

```
        /  E2E  \
       / Integration \
      /  Unit tests  \
     /________________\
```

- Unit tests (70% of effort): one function or method in isolation, fast
  (under 1ms each) and table-driven. Mock the repo interface, assert the
  use-case logic.
- Integration tests (25% of effort): a domain end to end with a real SQLite
  database (in-memory or temp file), real WebSocket, real HTTP; no mocks for
  the database. Verify that the outbox relay publishes, that the dedupe
  middleware skips duplicates, and that the dead-letter table gets written.
- E2E tests (5% of effort): manual, plus a few automated critical paths
  (create a ticket, deploy it, verify the topology update) against a real
  server and runner in Docker. These are slow and brittle; keep them few and
  high-value.

---

## 3. What to test (priority order)

1. Error paths. Every `if err != nil` branch. Every `ErrFatal` and
   `ErrRetryable` path. Every validation failure.
2. State transitions. Ticket open, in-progress, done. Deploy pending,
   running, healthy or failed. Test the valid transitions and the invalid
   ones.
3. Idempotency. Process the same event twice; assert no duplicate side
   effects.
4. Boundary conditions. Empty strings, zero values, max-length inputs,
   concurrent access (for the write serializer).
5. Happy paths. Test them too, but they are the last priority because they
   are the paths least likely to break silently.

---

## 4. What not to test

- Framework behavior. Do not test that React renders a `div`, or that
  `http.ServeMux` routes correctly.
- Third-party library internals. Do not test that `slog` formats JSON
  correctly.
- Trivial getters and setters with no logic.
- UI snapshot tests for layout. They break on every CSS change and teach
  nothing.

---

## 5. Test naming

```
Test<Function>_<Scenario>_<Expected>
```

Examples:
- `TestCreateTicket_WithEmptyTitle_ReturnsValidationError`
- `TestProcessDeployEvent_DuplicateEvent_IsSkipped`
- `TestRetryMiddleware_ExhaustedRetries_SendsToDeadLetter`

For table-driven tests, the `name` field in the test case is the scenario.

---

## 6. Mocking

- Mock at the interface boundary. The use-case takes a `TicketStore`
  interface; the test provides a fake. Never mock a concrete type.
- Every test double in this repo is a hand-written fake: a `type fakeXxx
  struct` that implements the real interface, backed by a map or a slice
  behind a mutex. A fake exercises the same contract the real SQLite
  implementation does, so it catches interface drift the moment a method
  signature changes; a mock library adds a second DSL a reader has to learn
  on top of the interface itself.
- When a test needs a call-count expectation (called exactly twice with
  these arguments), add a counter field to the hand-written fake and assert
  on it. That stays inside the same fake instead of pulling in a mocking
  library for one assertion shape.

---

## 7. CI enforcement

### Go

The `coverage` target in the `Makefile` is the source of record for the exact filtering and
threshold logic. It runs `go test` with `-race` and a coverage profile over
`./...` except `internal/platform/storage`, whose tests run without `-race`:
there the detector instruments the pure-Go SQLite engine and turns 24 seconds
into ten minutes, while the `server/cmd` integration tests still drive the
storage code concurrently under `-race`. It drops the exempt paths
(section 1) from the profile, then fails the build below 80%.
`golangci-lint` (config in `.golangci.yml`) and `govulncheck` run as their
own steps in the same Go CI job, alongside `go vet` and a `go build` without
`-race`. `coverage.filtered.out` and
`coverage.html` upload as the `go-coverage` CI artifact.

### Frontend

`web/vitest.config.ts` is the source of record for the coverage thresholds
(the same 80% gate, measured by Vitest's v8 provider) and the exempt-path
list. The web job runs typecheck, lint, and
test with coverage, then build; `client-core/` is linted, tested and counted
toward the gate there. The native job runs typecheck, lint and Jest. The sdk
and automations jobs each run their own typecheck step, then their test
script (Vitest in `sdk`, `bun test` in `automations`).
`web/coverage/lcov.info` uploads as the `web-coverage` CI artifact.

---

## 8. Test data

- Use builder functions, not raw struct literals:

```go
func newTestTicket(opts ...func(*Ticket)) *Ticket {
    t := &Ticket{ID: "test-1", Title: "Test", Status: TicketOpen}
    for _, opt := range opts {
        opt(t)
    }
    return t
}

func withTitle(title string) func(*Ticket) {
    return func(t *Ticket) { t.Title = title }
}
```

- Never share mutable state between tests. Each test creates its own data.
- Never write to a live or shared database from a test. Tests use a
  temporary SQLite file; data is copied in, never symlinked, and never
  written back out.
- Use `t.Cleanup` for teardown, not `defer` (cleanup runs even after
  `t.Fatal`).

---

## 9. Flaky tests

- A flaky test is a bug. Fix it or delete it. Do not `t.Skip` it or add
  retries to mask it.
- Common causes: time-dependent logic (use `testing/synctest` for goroutine
  code, or inject a clock), port conflicts (use `:0` for a random port),
  race conditions (run with `-race`).
- CI runs with `-race` on every PR, over every package but storage
  (section 7).
- Async code is tested by draining, never by sleeping. Wait on the real
  signal: a channel, a receipt, a WaitGroup, or `testing/synctest`'s virtual
  clock in Go. A test that needs a timeout or a sleep to pass is wrong,
  because the timeout is the bug wearing a disguise.

---

## 10. Performance changes

- A performance fix ships with a guard test that fails when the fix is
  reverted: revert it locally, watch the test go red, restore it. Without the
  guard the next refactor undoes the win and nobody notices. The guard
  counts something deterministic (components rendered, requests sent,
  statements run, allocations, modules in an import graph, goroutines left
  running), never wall-clock time, which flakes on a shared CI runner.
  `pageChunks.test.tsx`, the request counts in `liveRequests.test.tsx` and
  the phone's `requestCounts.test.tsx`, the statement guards in
  `server/cmd/access_memo_test.go` and goleak are guards of this kind.
- Numbers come from a production or release build on realistic data, never
  from a dev build: the web app from `vite build` under CPU throttling, the
  phone from a release build, the server on a copy of the database scaled to
  a heavy user. Dev builds add double renders, dev-only checks and
  unminified bundles that hide a cost or invent one. Alternate base and head
  runs and report medians, each timing beside the deterministic count that
  explains it.
