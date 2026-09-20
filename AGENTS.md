# AGENTS.md

Read this top to bottom before writing any code.

## What is this project?

Read `README.md` (one page). It explains the product, where the idea comes
from, and the MCP-first loop. The deployment model is in the install guide on
nexul.io, and the standing principles are at the end of `ROADMAP.md`.

The product is implemented. The work now is improving it domain by domain.

## How to navigate the docs

| You are doing | Read first | Then read |
|---|---|---|
| Anything | `CONTEXT.md` (the vocabulary) | `docs/adr/` for the decisions in your area |
| Picking up work | `.scratch/`, the issue tracker | `docs/agents/issue-tracker.md` for its conventions |
| Go backend | `practices/go.md` | `practices/architecture.md` |
| Frontend (React) | `practices/react-guide.md`, the F1 to F7 commandments are enforced | `practices/design-language.md` |
| SDK, automations host, desktop | `practices/typescript.md` | `practices/react-guide.md` for the desktop launcher |
| Design or visual work | `practices/design-language.md`, the Mono Console spec | `practices/react-guide.md` |
| Testing | `practices/testing.md` | The language file above |
| Any code | `practices/borrowed-practices.md`, the cross-cutting rules | `practices/README.md` for the index |
| Docker, CI, deploy | [CI and releases](https://nexul.io/docs/contributing/ci-and-releases/) | Root `docker-compose.yml` |
| MCP server | `practices/architecture.md`, section 8 | `docs/adr/` |
| Event bus or resilience | `practices/architecture.md`, sections 2 to 6 | `docs/adr/` |
| Why is it built this way? | `docs/adr/` | The effort's spec in `.scratch/` |
| Unfamiliar with a term | `CONTEXT.md` | (the ubiquitous language) |

The practices files are the standard. The contributing pages on nexul.io are a
digest derived from them, never the other way round.

## Agent skills

This repo is configured for the mattpocock/skills engineering set.

Issues and specs live as markdown under `.scratch/<effort-slug>/`, committed
to the repo. See `docs/agents/issue-tracker.md`.

The five triage labels are `needs-triage`, `needs-info`, `ready-for-agent`,
`ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

`CONTEXT.md` is the glossary, `docs/adr/` is the decision log, and the tracker
holds the specs. See `docs/agents/domain.md`.

`.scratch/pre-release/` holds deferred concerns, each with a "Surface when"
list. If a trigger matches the current task, raise it with the owner before
implementing. Never act on one silently.

## The hard rules

These override everything else. If a practices file says something different,
these win.

1. Read the practices before writing code, every session. The navigation
   table above is a gate, not a suggestion. Before the first edit, read the
   `practices/` files matching the work. Delegated work inherits this: any
   brief that involves writing code names the files to read before editing,
   and they get read there. A summary in the brief does not count.
2. MCP-first, not MCP-only. The browser uses the HTTP/JSON gateway. The MCP
   server is for LLMs. Both are adapters over the same use-case layer, which
   is the single source of behavior.
3. SQLite is the spine. One file, one owner, WAL mode, FTS5. No Postgres. No
   dual-backend abstraction. Semantic search deferred.
4. The EventBus interface is the microservice seam. Domains communicate only
   through it. Today it is in-process channels; a broker adapter can replace
   it without touching a domain.
5. Coverage is a floor (80%), not a goal. Test error paths first. Exempt:
   `cmd/*`, generated code, wire types, `testutil/`, `components/ui/`.
6. No barrels. No `index.ts` re-exports. Import by full path.
7. Early return, no `else`, in Go and TypeScript alike.
8. Structured logging: `slog` in Go, console with a propagated `trace_id` in
   TypeScript. No `fmt.Println`, no `console.log` in production code.
9. Comments are sparse. The default is no comment; the code and its names
   carry the meaning. A comment exists only to say a why, a warning about a
   non-local consequence, or a pointer to context, and it fits on one line.
   Knowledge that needs more than one line is an ADR if it is a decision,
   otherwise it belongs in the tracker spec for the work. No comment restates
   the code. No change history ("previously", "per PR"). No commented-out
   code. No TODO without a tracking issue in `.scratch/`.
10. Web work obeys the Frontend Commandments. F1 to F7 in
    `practices/react-guide.md` are hard rules for anything in `web/`: Page,
    Feed, Section, Card; defensive ordering; shared display components; `&&`
    over ternary; no gratuitous `useState` or `useEffect`; no prop drilling;
    thin files. Implement against the rule, not against nearby code that
    predates it. Run the self-review checklist before calling web work done.
11. Extend the Mono Console, never re-theme. No new hues, no serif or script
    fonts. Color is reserved for status signal. A design that needs a missing
    token adds it to `web/src/index.css` and to the token table in
    `practices/design-language.md`, never as a one-off class.
12. Mobile first. Design and build at 320, 375, and 414px first; desktop is
    the enhancement. Verify at 320, 375, 414, and 768px before calling web
    work done.

## What enforces the rules

A rule a linter can check is checked by a linter, because prose rules get
skipped and lint errors do not.

| Rule | Gate |
|---|---|
| Go static analysis, formatting, imports, complexity | `golangci-lint` with `.golangci.yml`, in CI and `make lint` |
| Known-vulnerable Go dependencies | `govulncheck`, in CI and `make vuln` |
| Go coverage floor | `make coverage`, in CI and locally |
| Generated SQL code matches the queries | `sqlc vet` and `sqlc diff`, in CI and `make sqlc-check` |
| Web lint, types, tests, coverage | `bun run lint`, `typecheck`, `test` in `web/`, in CI |
| SDK and automations host types and tests | `bun run typecheck` and `bun run test` in each package, in CI |
| Desktop types, tests, build | the desktop CI job |
| Dependency freshness | Dependabot, weekly, grouped per directory |

A rule that could be a lint rule and is not yet is a candidate for one; add
it to the tracker rather than repeating it in review.

## Hit every surface

The most common defect in this repo is a change that works on the path you
tested and is missing everywhere else. Before calling a feature done, walk
this list and say which entries applied:

- UI page or component. The browser flow works at 320, 375, 414, and 768px.
- HTTP gateway route. The browser and integrations reach it (ADR 0019).
- MCP tool. Agents are peers of the browser; a capability without a tool is
  half shipped.
- Events. A catalog row and an outbox write, designed for publication
  (ADR 0044).
- Live WebSocket push, if the UI should update without a refresh.
- Search, if the entity is indexed.
- Permissions. Enforced through the permission table, not assumed.
- Reverse states. If you added a way in, add the way out and the way to see
  it. Archive needs restore, close needs reopen. A one-way door is a bug.
- Docs. Check whether the change makes an ADR, `CONTEXT.md`, or a practices
  file inaccurate, and fix it in the same change.

## Plans and work artifacts

- Do not commit implementation plans, research notes, audit reports, or agent
  scratch files. Temporary working material lives outside the tree or in a
  path listed in `.git/info/exclude`.
- `.scratch/` holds specs and tickets for open efforts, nothing else. When an
  effort ships, its directory is deleted.
- A merged PR is the implementation record. Close its ticket when the work
  lands; do not keep a second checklist anywhere in the repository.
- Nothing shipped names the tool that wrote it. Commit messages, PR bodies,
  comments, and docs describe the change from the code's point of view.

## Worktree workflow

Parallel work happens in worktrees (`git worktree add ../nexul-<slug> -b
<slug>`). Two things the commands do not tell you:

- Remove the worktree when the branch is done; each one is about 65MB.
- Parallel runs share this host. Give each its own dev-server port and say so
  up front, or they collide on the default port and the screenshots come back
  belonging to somebody else's container.

CI runs per service through `dorny/paths-filter`, so only the affected
service's jobs run.

## Code discovery

`rg` is the default and is never stale. Reach for the `codebase-memory-mcp`
server only for structural questions it answers better: `trace_path` for who
calls what, `search_graph` to disambiguate an overloaded name. Reindex
(`index_repository`) before tracing if files moved, not after every new file.

## Quick reference: adding a new domain

1. Create `internal/<domain>/` with `model.go`, `repo.go`, `usecase.go`,
   `handler.go`, `events.go`, and `mcp.go` if it exposes tools.
2. Add the repo implementation in `internal/platform/storage/`, with the
   queries in `internal/platform/storage/queries/<table>.sql` and `make sqlc`.
3. Register the domain's topics in its `Topics()` function; the catalog
   aggregates them (`practices/architecture.md`, section 2).
4. Add the MCP tools, one per use-case, named `<verb>_<object>`.
5. Add the HTTP routes in `server/cmd/` or a router package.
6. Add the frontend: model, hooks, components, page, route.
7. Add tests: unit (table-driven) and integration (real SQLite).
8. Add the domain's terms to `CONTEXT.md`, and record any decision that was
   hard to reverse, surprising, or a real trade-off as an ADR in `docs/adr/`.

## Quick reference: adding a new service binary

1. Create `<service>/cmd/main.go` (composition root).
2. Create `<service>/Dockerfile` (multi-stage: `release` and `debug` targets).
3. Add it to the root `docker-compose.yml` and `docker-compose.debug.yml`.
4. Add a CI job in `.github/workflows/ci.yml` with a `dorny/paths-filter`
   entry.

## When in doubt

- Read `CONTEXT.md`. It defines the project's vocabulary. Use its terms and
  respect its `_Avoid_` lists.
- Read `docs/adr/`. It has the decisions and the trade-offs behind them. If
  your work contradicts one, say so explicitly rather than quietly overriding
  it.
- Read the effort's spec in `.scratch/`. It has the requirements for the work
  in front of you.
- If something is genuinely ambiguous, raise it with the owner and record the
  answer as an ADR if it is a real decision, otherwise in the tracker spec.
- Push back if a rule seems wrong. The docs evolve, but only with a written
  rationale.
