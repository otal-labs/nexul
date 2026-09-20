# Practices

The coding standard for this repository, one file per language or surface.
`AGENTS.md` routes every task to the file it needs; a pull request is
reviewed against these files, not against whatever nearby code happens to
do. The coding standards page on the docs site is a digest of them.

| File | Read it when | What it holds |
|---|---|---|
| `architecture.md` | Touching any Go domain, the event bus, the MCP server, or a service boundary | The per-domain layer shape, the EventBus seam, outbox and idempotency, shutdown, the adapters |
| `go.md` | Writing Go | Layout, errors, logging, config, injection, concurrency, naming, tests, sqlc, dependency hygiene |
| `react-guide.md` | Writing anything in `web/` | The Frontend Commandments, structure, hooks, state, data fetching, forms, the API and WebSocket layers, the canvas, styling, testing |
| `typescript.md` | Writing anything in `sdk/`, `automations/`, or `desktop/` | Runtimes, strictness, the SDK's public surface, errors, testing, the Electron security rules |
| `design-language.md` | Changing how anything in `web/` looks | The Mono Console token spec, type, shape, motion, and the list, filter, detail, empty, and stepper patterns |
| `testing.md` | Writing or reviewing tests in any language | The coverage floor, the pyramid, what to test first, mocking, flaky tests, CI enforcement |
| `borrowed-practices.md` | Any code | Cross-cutting rules: one-way doors, SQLite discipline, retries, async tests, the git workflow |

A rule states what to do and why. When a rule and the code disagree, the
rule wins and the code is the defect; when a rule and this repository's
`AGENTS.md` hard rules disagree, `AGENTS.md` wins. A rule that turns out to
be wrong is changed in a pull request with the reason, never worked around.
