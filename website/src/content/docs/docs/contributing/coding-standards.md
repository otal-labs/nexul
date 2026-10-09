---
title: Coding standards
description: A digest of the practices that govern Go, MCP, React, the phone app, testing, and visual design in this repo.
sidebar:
  order: 4
---

This page is a digest. The full standard is the set of files in
[`practices/`](https://github.com/otal-labs/nexul/tree/master/practices) in the
repository, one file per language or surface, and every pull request is
reviewed against those files. Read the file for the language you are touching
before writing code; this page tells you what to expect from it.

## Go

Full text: [`practices/go.md`](https://github.com/otal-labs/nexul/blob/master/practices/go.md)
and [`practices/architecture.md`](https://github.com/otal-labs/nexul/blob/master/practices/architecture.md).


- Interfaces are defined at the consumer side (`EventBus` lives where the
  domains that call it live), kept small (1–3 methods), and returned as
  structs, accepted as interfaces.
- Sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrUnauthorized`), wrapped
  with context at the boundary; event handlers distinguish `ErrRetryable`
  from `ErrFatal`.
- `log/slog` only — no third-party logger, no `fmt.Println`. Every request
  or message handler carries a `trace_id`-scoped child logger through
  `context.Context`.
- Constructor injection, no global singletons, no DI framework.
- Table-driven tests with `testify` (`require` for preconditions, `assert`
  for checks).
- Queries are hand-written `.sql` files; `sqlc` generates the typed Go. See
  ADR [0009](https://github.com/otal-labs/nexul/blob/master/docs/adr/0009-sqlc-generates-the-storage-queries.md).
- SQLite has one writer: every write goes through the storage serializer,
  transactions begin `IMMEDIATE`, pooled connections stay warm, and lists
  paginate in SQL, by keyset or behind the MCP offset contract, with indexes
  that carry their tiebreaker.
- Loops that deliver committed rows wake on the serializer's commit
  broadcast instead of polling, and a slow client gets its own bounded send
  queue so it never stalls the others. Packages that start goroutines guard
  them with `goleak`.
- Every adapter (HTTP, MCP, live frames) encodes through one JSON encoder,
  `jsonx`, so an empty list is always `[]`; long lists encode through plain
  wire structs, and a list filters by permission in its SQL, so a page reads
  one page of rows and its total counts exactly what the pages hold.
- The architecture guide opens with three principles: a one-way door is a
  bug, build the smallest model that makes behavior unsurprising, and keep
  complexity in the adapters.

## MCP

Full text: [`practices/mcp.md`](https://github.com/otal-labs/nexul/blob/master/practices/mcp.md).

The MCP server runs on the official Go SDK behind one stateless endpoint,
`POST /mcp`, and domains declare tools through a typed contract instead of
importing the SDK. Tools are shaped per task and kept under 100: one-field
setters fold into patch-style updates, list variants into filtered lists.
Names are `<object>_<verb>` in the glossary's words, every parameter is
described, every tool carries read-only and destructive hints, lists are
paginated, and a tool's failure comes back as an `isError` result that says
how to recover.

## React (the Frontend Commandments)

Full text: [`practices/react-guide.md`](https://github.com/otal-labs/nexul/blob/master/practices/react-guide.md).


F1–F7 are hard rules for anything in `web/` — a PR that breaks one is not
done, regardless of what nearby code does:

- **F1 — Page → Feed → Section → Card.** Every screen decomposes into named,
  single-responsibility components; no inline `.map()` rendering a
  `<section>` or large JSX block.
- **F2 — `&&`, negative-first.** Loading → error → empty → data, each its
  own `&&` block. No `if (x) return <Component/>` for rendering.
- **F3 — Shared display components.** `LoadingDisplay`, `ErrorDisplay`,
  `NoDataDisplay`, `Container` — never hand-rolled equivalents.
- **F4 — `&&` for components, ternaries only for values.** Never a ternary
  choosing between two components.
- **F5 — `useState`/`useEffect` are a last resort.** Server state lives in
  TanStack Query, cross-surface state in Zustand, form state in React Hook
  Form; effects are for real side effects only.
- **F6 — No prop drilling.** Past ~2 levels, the child fetches its own data
  or reads a store instead.
- **F7 — Files own one concern.** Pages stay thin; sub-components live in
  `components/<domain>/`.

Live data follows a few rules of its own. The server's audience rules and
the browser's topic tables are pinned together: `make live-topics`
regenerates the topic list and tests on both sides fail when they drift, so
a new live topic ships with its rule, the regenerated list and a handler. A
frame that carries the entity patches the cache from its payload instead of
refetching lists, every ticket view goes through one cache module, and a
page's batch reads are keyed by project rather than by every id. Every page
is its own chunk that the shell never imports, and rows of a live list are
memoised components with stable props.

## The phone app

Full text: [`practices/native.md`](https://github.com/otal-labs/nexul/blob/master/practices/native.md).

The Expo app inherits the web guide, F1–F7 included, with a Screen in place
of a Page. Reference data is cached forever and refreshed by a pushed topic,
and a test fails for any such query without one; chat frames patch the
cached thread; record queries are matched by id or key through one helper;
the session token is read once at launch; and icons import by path, which a
lint rule enforces.

## TypeScript outside the browser

Full text: [`practices/typescript.md`](https://github.com/otal-labs/nexul/blob/master/practices/typescript.md).

The SDK and the automations host run on Bun and test with `bun test`; the
desktop shell is Electron with the same strictness flags as the web app and
the Electron security checklist (context isolation on, node integration off,
a preload bridge with an explicit allow-list) stated as absolutes.

## Testing

Full text: [`practices/testing.md`](https://github.com/otal-labs/nexul/blob/master/practices/testing.md).


Coverage (80% gate, 90% target) is a floor, not a goal — error paths,
state transitions, and idempotency come before happy paths. Mock at the
interface boundary; prefer fakes over mocks for repos. A flaky test is a
bug: fix it or delete it, never skip or retry-mask it, and async code is
tested by draining, never by sleeping. A performance fix ships with a guard
test that fails when the fix is reverted, and its numbers come from a
production or release build on realistic data.

## Design language — frosted panels and one ember accent

Full text: [`practices/design-language.md`](https://github.com/otal-labs/nexul/blob/master/practices/design-language.md).

One design language covers the web app, the public site and the phone app:
the file opens with a shared core and has a section per surface. In the web
app every page's content floats in a raised, frosted panel over a near-black
canvas that carries a soft light field; light mode is its own soft grey
canvas with white panels. One ember accent, the `brand` token, marks the
primary action, the active nav item, selection, your own chat messages,
checked controls and progress, and nothing else; focus is an ink outline, and
status keeps its own hues as a dot or icon beside plain text. Page titles are
set in Fraunces, technical data (ids, repos, timestamps) in JetBrains Mono,
everything else in Inter. The public site uses the same tokens and roles:
`website/src/styles/tokens.css` maps the app's values onto the docs theme, and
reading pages keep their text on the canvas. The phone app carries the same
tokens in `native/src/global.css`, on solid surfaces with no blur. When a
design needs a new token, add it to every surface's token file that uses it
(`web/src/index.css`, `website/src/styles/tokens.css`, `native/src/global.css`);
never a one-off colour.

## What enforces the rules

A rule a linter can check is checked by a linter. Go runs `golangci-lint`
(config in `.golangci.yml`) and `govulncheck` in CI and through `make lint`
and `make vuln`; the coverage gate is `make coverage`; `sqlc vet` and
`sqlc diff` keep generated code honest. Every TypeScript package runs its
`typecheck` and `test` scripts in CI, and Dependabot opens grouped weekly
update pull requests per directory. Tests guard the rest: the live topic
contract, the JSON encoder's parity with the old output, goroutine leaks,
the web shell's import graph, and the phone's forever-cached queries; a lint
rule keeps phone icons imported by path.

## Hard rules from AGENTS.md

These apply everywhere, regardless of surface:

- **Early return, no `else`.** In Go and TypeScript alike — handle the
  exceptional case and return, leave the happy path unindented.
- **No barrels.** No `index.ts` re-export files. Import by full path.
- **Structured logging.** `slog` in Go, console with a propagated
  `trace_id` in TypeScript. No `fmt.Println`, no `console.log` in
  production code.
- **Comments are sparse.** Default is no comment. One exists only to say a
  *why*, a non-local warning, or a pointer — never to restate the code,
  never a change-history note, one line, and no commented-out code.
- **Tablet and desktop.** Build the web app at 768px first and verify at
  768 / 1024 / 1440px before calling web work done; phones are served by the
  Android app. The website stays mobile-first at 320 / 375 / 414px.
