# Practices

The coding standard for this repository, one file per language or surface.
`AGENTS.md` routes every task to the file it needs; a pull request is
reviewed against these files, not against whatever nearby code happens to
do. The coding standards page on the docs site is a digest of them.

| File | Read it when | What it holds |
|---|---|---|
| `architecture.md` | Touching any Go domain, the event bus, the MCP server, or a service boundary | The principles (one-way doors, the smallest model, complexity at the adapters), the per-domain layer shape, the EventBus seam, outbox, retries and idempotency, shutdown, the adapters |
| `go.md` | Writing Go | Layout, errors, logging, config, injection, concurrency and delivery loops, tests and goleak, SQLite and sqlc, JSON on the wire, the access memo and access checks in lists |
| `mcp.md` | Touching the MCP adapter, the tool contract, or any domain's `mcp.go` | Transport and security, the tool budget, naming, declaring tools, result and error shape, instructions, resources and prompts, testing |
| `react-guide.md` | Writing anything in `web/` | The Frontend Commandments, structure, data fetching and the ticket cache, state, forms, the API layer, live followers and the live topic contract, the canvas, styling mechanics, testing, performance |
| `native.md` | Writing anything in `native/` | The phone stack, which web rules carry over and how, navigation, styling, query declarations and the socket, testing, running and releasing |
| `typescript.md` | Writing anything in `sdk/`, `automations/`, `desktop/`, or `client-core/` | Runtimes, strictness, the SDK's public surface, errors, logging, the Electron security rules, testing, packaging, the client core the web and phone apps share |
| `design-language.md` | Changing how anything in `web/`, `website/` or `native/` looks | One design language for all three: a shared core (tokens, the accent, focus, status, type, shape, motion, the decision ledger), then a section per surface with its adaptations, patterns and motion locks |
| `testing.md` | Writing or reviewing tests in any language, or making something faster | The coverage floor, the pyramid, what to test first, mocking, CI enforcement, test data, flaky tests, guard tests and measurement for performance changes |

A rule states what to do and why. When a rule and the code disagree, the
rule wins and the code is the defect; when a rule and this repository's
`AGENTS.md` hard rules disagree, `AGENTS.md` wins. If a rule fights the task
in front of you, say so and get the owner's sign-off before breaking it: a
silently broken rule is indistinguishable from one nobody knew about. A rule
that turns out to be wrong is changed in a pull request with the reason,
never worked around.
