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
| Frontend (React) | `practices/react-guide.md`, the F1 to F7 commandments are enforced | `practices/design-language.md`, the shared core and the web app section |
| SDK, automations host, desktop | `practices/typescript.md` | `practices/react-guide.md` for the desktop launcher |
| Phone app (React Native) | `practices/native.md` | `practices/react-guide.md`, the rules it inherits; `practices/design-language.md`, the shared core and the phone app section |
| Code both the web and phone apps run (`client-core/`) | `practices/typescript.md`, section 11 | ADR 0139 |
| Design or visual work in `web/`, `website/` or `native/` | `practices/design-language.md`, one language for all three: the shared core, then your surface's section | `practices/react-guide.md`, or `practices/native.md` on the phone |
| Testing, or making something faster | `practices/testing.md` | The language file above |
| Any code | `practices/architecture.md`, Principles | `practices/README.md` for the index |
| Install, CI, release | [CI and releases](https://nexul.io/docs/contributing/ci-and-releases/) | `internal/install/`, `.goreleaser.yaml`, `docker-compose.debug.yml` for development |
| MCP server or a domain's `mcp.go` | `practices/mcp.md` | `practices/architecture.md`, section 8 |
| A list call: an MCP list tool or a paged endpoint | `practices/go.md`, section 17 | `practices/mcp.md`, section 7; ADR 0140 |
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
   `cmd/*`, generated code, wire types, `testutil/`, `t3rpctest/`, `components/ui/`.
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
11. One design language across `web/`, `website/` and `native/`; extend it,
    never re-theme. `practices/design-language.md` is the spec for all three,
    and a screen that disagrees with it is the defect. Tokens only, no
    one-off colours: the one accent is `brand`, held to the roles the spec
    names, and status keeps its own hues. Fraunces is the one serif, for
    display headlines only; no script fonts. The three token files
    (`web/src/index.css`, `website/src/styles/tokens.css`,
    `native/src/global.css`) carry the same names and values; a design that
    needs a missing token adds it to each surface that uses it and to the
    token table in `practices/design-language.md`, never as a one-off class.
12. Tablet and desktop. Build `web/` at 768px first and verify at 768, 1024,
    and 1440px before calling web work done; phones are served by the
    Android app in `native/` (ADR 0080). `website/` stays mobile first: build
    at 320, 375, and 414px and verify at 320, 375, 414, and 768px.

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
| Native lint, types, tests | `bun run lint`, `typecheck`, `test` in `native/`, in CI |
| SDK and automations host types and tests | `bun run typecheck` and `bun run test` in each package, in CI |
| Desktop types, tests, build | the desktop CI job |
| Live topics the server pushes match the ones the web followers follow | `make live-topics`, `TestLiveTopicsFile_MatchesTheRules` and `web/src/hooks/liveTopics.test.tsx` |
| Event schemas match the payload types and the published contract only grows | `make event-schemas`, `TestSchemas_MatchThePublishedContract` in `internal/eventcatalog/contract_test.go` |
| HTTP bodies through `jsonx` match the encoder it replaced | `server/cmd/json_parity_test.go` |
| An event payload's empty list is `[]`, never `null` | the contract test above, and `TestInsertOutboxRow_WritesAnEmptyListAsAnArray` |
| A memoised access check answers what an unmemoised one does, and a commit clears it | `TestIntegration_PermissionSuitesAnswerTheSameWithAMemo` and `TestIntegration_LiveAudienceMemo_FollowsEveryChange` in `server/cmd/access_memo_test.go` |
| A list path's access reads stay the same at any length | the `TestStatements_*` guards in `server/cmd/access_memo_test.go` |
| A paged list tool shows each viewer what the row-by-row check showed, and reads about one page of rows | `TestListTools_ShowWhatTheRowByRowCheckShowed` and the `TestRows_*` guards in `server/cmd/list_paging_test.go` |
| Every non-GET route writes an audit row unless it is listed as read-only | `TestAudited_EveryRouteOfTheRouter_IsClassifiedByMethod` in `server/cmd/audit_integration_test.go` |
| No goroutine outlives a package's tests | `goleak.VerifyTestMain` in each goroutine-owning package's `main_test.go` |
| The web shell loads no page, editor, canvas, voice or shader | `web/src/pageChunks.test.tsx` |
| Requests one live frame or one phone session sends | `web/src/hooks/liveRequests.test.tsx`, `native/src/hooks/requestCounts.test.tsx` |
| `client-core/` lint, tests and coverage | the web job: `bun run lint` covers the folder, and vitest runs its tests and counts its code toward the 80% gate; each app's typecheck compiles what it imports |
| A phone query cached until pushed names a topic, and its topics and payload fields exist in the catalog | `bun run typecheck` in `native/`, through the `@ts-expect-error` cases in `native/src/lib/liveQuery.test.ts` |
| A catalog topic reaches exactly the phone queries that declare it | the per-topic table test in `native/src/hooks/useLiveEvents.test.tsx` |
| Phone icons import by path; the SDK's event types import as types only | `no-restricted-imports` in `native/eslint.config.js` |
| Dependency freshness | Dependabot, weekly, grouped per directory |

A rule that could be a lint rule and is not yet is a candidate for one; add
it to the tracker rather than repeating it in review.

## Hit every surface

The most common defect in this repo is a change that works on the path you
tested and is missing everywhere else. Before calling a feature done, walk
this list and say which entries applied:

- UI page or component. The browser flow works at 768, 1024, and 1440px.
- HTTP gateway route. The browser and integrations reach it (ADR 0019).
- MCP tool. Agents are peers of the browser; a capability without a tool is
  half shipped. A list tool pages in SQL through its use-case, filters by
  access in the query, and joins the parity and row guards in
  `server/cmd/list_paging_test.go` (`practices/go.md`, section 17).
- Events. A `Topics()` entry naming the payload type, `make event-schemas`,
  and an outbox write, designed for publication (ADR 0044, ADR 0137). A new
  field goes beside the existing ones, never in place of one, and is
  followed by `make event-schemas`. A client that refreshes too much because
  a frame lacks an id is fixed by adding that id this way.
- Live WebSocket push, if the UI should update without a refresh. A new
  topic gets an audience rule, `make live-topics`, a follower in the web
  domain's hooks file (`practices/react-guide.md`, The live topic contract),
  and a line in the `refreshes` of each phone query that shows the entity
  (`practices/native.md`, section 4).
- Both apps. Code the web and the phone app both need lives once in
  `client-core/`, with its tests beside it (`practices/typescript.md`,
  section 11); each app keeps only its adapter.
- Search, if the entity is indexed.
- Permissions. Enforced through the permission table, not assumed.
- Reverse states. If you added a way in, add the way out and the way to see
  it. Archive needs restore, close needs reopen. A one-way door is a bug.
- Docs. Check whether the change makes an ADR, `CONTEXT.md`, or a practices
  file inaccurate, and fix it in the same change.
- User guide. A feature added or changed updates its page under
  `website/src/content/docs/docs/guide/` in the same change, or adds one.

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

## Branches and pull requests

Parallel work happens in worktrees (`git worktree add ../nexul-<slug> -b
<slug>`). Things the commands do not tell you:

- Remove the worktree when the branch is done; each one is about 65MB.
- Parallel runs share this host. Give each its own dev-server port and say so
  up front, or they collide on the default port and the screenshots come back
  belonging to somebody else's container.
- Kill only a PID you captured at spawn, never by pattern (`pkill -f`,
  `pgrep | kill`): your own session matches the pattern.

A pull request:

- Carries one topic. A description that says "also" is two pull requests.
- Is rebased onto the latest `master` before it opens, so the diff is only
  the change.
- Is squash-merged. Its title is one sentence describing the change from the
  code's point of view, because that sentence becomes the commit on `master`
  and the line in the release notes.
- States the problem in a sentence or two, then how the change fixes it. A UI
  change carries before and after screenshots at 768, 1024, and 1440px (320,
  375, 414, and 768px for `website/`); a change that depends on motion
  carries a short video.
- Never commits a secret. A dev-only shared value is tracked as an open item
  in `.scratch/pre-release/` until it is rotated.

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
3. Register the domain's topics with their payload types in its `Topics()`
   function and run `make event-schemas`; the catalog aggregates them
   (`practices/architecture.md`, section 2).
4. Expose the use-cases to agents per `practices/mcp.md`: extend an existing
   tool first, add one only inside the tool budget, named `<object>_<verb>`.
   A list use-case takes its filters and a `paging.Window` (ADR 0140).
5. Add the HTTP routes in `server/cmd/` or a router package.
6. Add the frontend: model, hooks with the domain's live follower,
   components, page, route; on the phone, its `defineQuery` declarations.
7. Add tests: unit (table-driven) and integration (real SQLite).
8. Add the domain's terms to `CONTEXT.md`, and record any decision that was
   hard to reverse, surprising, or a real trade-off as an ADR in `docs/adr/`.

## Quick reference: adding a new service binary

Releases ship native binaries only, never images (ADR 0073).

1. Create `<service>/cmd/main.go` (composition root).
2. Ship it as `nexul-<service>-<os>-<arch>[.exe]` for every release target: a
   Go binary is a build in `.goreleaser.yaml`; anything else is built in
   `.github/workflows/release.yml` and attached through the GoReleaser
   `extra_files` and checksum globs.
3. Teach `internal/install/` to run it: a unit kind, its environment, and
   what `nexul install`, `upgrade`, `status` and `uninstall` do with it on
   systemd, launchd and the Windows service host.
4. For development, add a `<service>/Dockerfile.debug` and a service in
   `docker-compose.debug.yml`.
5. Add a CI job in `.github/workflows/ci.yml` with a `dorny/paths-filter`
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
