# Testing practices (Nexul)

This document defines the testing philosophy for Nexul. It applies
to all languages and stacks. Language-specific mechanics live in
`practices/go.md` for Go and in the Testing (frontend) section of
`practices/react-guide.md` for TypeScript and React.

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

- Go coverage drops paths containing `/cmd/`, `/testutil/`, or `/sqlcgen/`.
- Pure wire types (structs with no methods, no validation) never appear in
  the coverage profile at all, since `go test` only emits statements for
  executable code.
- Frontend: `components/ui/` (shadcn primitives, tested by their own
  suite).

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

- E2E: manual, plus a few critical-path automated tests.
- Integration: a real SQLite database (in-memory or temp file), real
  WebSocket, real HTTP.
- Unit tests: fast, isolated, table-driven.

- Unit tests (70% of effort): test one function or method in isolation.
  Mock the repo interface, assert the use-case logic. Fast, under 1ms each.
- Integration tests (25% of effort): test a domain end-to-end with a real
  SQLite database, in-memory or temp file. No mocks for the database.
  Verify that the outbox relay publishes, that the dedupe middleware skips
  duplicates, and that the dead-letter table gets written.
- E2E tests (5% of effort): a few critical paths (create a ticket, deploy
  it, verify the topology update) run against a real server and runner in
  Docker. These are slow and brittle; keep them few and high-value.

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

## 7. Frontend testing

### Unit tests (Vitest and React Testing Library)

- Test hooks in isolation (render with a wrapper, assert return values).
- Test components by behavior, not implementation:
  - Query by role, label, or text; never by class name or test ID unless
    there is no accessible alternative.
  - Use `userEvent` over `fireEvent`; it simulates real browser behavior.
  - Use `screen` for queries.
  - Use `findBy*` for async assertions.
- Test Zustand stores by calling actions and asserting state; no rendering
  needed.

### Integration tests

Mock the API module with `vi.mock`, not the network. Every hook test under
`web/src/hooks/*.test.tsx` and page test under `web/src/pages/*.test.tsx`
calls `vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn(),
... } }))` and drives the mocked methods with
`vi.mocked(api.get).mockResolvedValue(...)`. Render a page or hook against
that mock and assert the full flow: load, display, interact, mutate, toast.

### Coverage

- Same 80% gate, measured by Vitest's v8 provider.
- Exemptions: `components/ui/` (shadcn), `lib/utils.ts` (trivial), test
  files, config files.

---

## 8. CI enforcement

### Go

The `coverage` target in the `Makefile` is the source of record for the exact filtering and
threshold logic. It runs `go test` with `-race` and a coverage profile over
`./...`, excludes paths containing `/cmd/`, `/testutil/`, or `/sqlcgen/` from
the profile, then fails the build below 80%. `golangci-lint` (config in
`.golangci.yml`) and `govulncheck` run as their own steps in the same Go CI
job, alongside `go vet` and `go build`. `coverage.filtered.out` and
`coverage.html` upload as the `go-coverage` CI artifact.

### Frontend

`web/vitest.config.ts` is the source of record for the coverage thresholds
and the exempt-path list (`components/ui/`, `lib/utils.ts`, the test setup
files, config files, `main.tsx`). The web job runs typecheck, lint, and
test with coverage, then build. The sdk and automations jobs each run
their own typecheck step, then `bun test`. `web/coverage/lcov.info` uploads
as the `web-coverage` CI artifact.

---

## 9. Test data

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
- Use `t.Cleanup` for teardown, not `defer` (cleanup runs even after
  `t.Fatal`).

---

## 10. Flaky tests

- A flaky test is a bug. Fix it or delete it. Do not `t.Skip` it or add
  retries to mask it.
- Common causes: time-dependent logic (use `testing/synctest` for goroutine
  code, or inject a clock), port conflicts (use `:0` for a random port),
  race conditions (run with `-race`).
- CI runs with `-race` on every PR.
- A test that needs a timeout or a sleep to pass is wrong. Wait on the real
  signal instead: a channel, a receipt, or a WaitGroup.
