---
title: Coding Standards
description: A digest of the practices that govern Go, MCP, React, testing, and visual design in this repo.
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
bug: fix it or delete it, never skip or retry-mask it.

## Design language — the Mono Console

Full text: [`practices/design-language.md`](https://github.com/otal-labs/nexul/blob/master/practices/design-language.md).


Nexul is strictly monochrome and dark-first, with no accent color
anywhere in the chrome — near-black or near-white surfaces, true mirror
inversions of each other, layered by 1px hairlines rather than blur shadows.
Technical data (ids, repos, timestamps) is set in JetBrains Mono, and code
or log surfaces read like terminal windows. Color is reserved entirely for
badge and status signal, never for chrome decoration — extend the token set
in `web/src/index.css` when a design needs one, never re-theme with a new
hue.

## What enforces the rules

A rule a linter can check is checked by a linter. Go runs `golangci-lint`
(config in `.golangci.yml`) and `govulncheck` in CI and through `make lint`
and `make vuln`; the coverage gate is `make coverage`; `sqlc vet` and
`sqlc diff` keep generated code honest. Every TypeScript package runs its
`typecheck` and `test` scripts in CI, and Dependabot opens grouped weekly
update pull requests per directory.

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
- **Mobile-first.** Design and build at 320 / 375 / 414px first; verify at
  320 / 375 / 414 / 768px before calling web work done.
