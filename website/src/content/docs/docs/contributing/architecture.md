---
title: Architecture
description: How Nexul is put together — one Go module, two adapters over a shared use-case layer, and everything around it.
sidebar:
  order: 1
---

Nexul is one Go module (`github.com/otal-labs/nexul`) that builds a
`server` binary and a `runner` binary. The repository also contains a React
web app, a thin Electron desktop shell, a TypeScript SDK, a Bun CLI exposed as
`nexul`, and a small automations host. Keeping these surfaces in one repository
makes the seams between them explicit.

## The domain layer

Business logic lives under `internal/<domain>/`, one directory per bounded
context — `tickets`, `deploy`, `docs`, `topology`, and so on. A domain owns
its model, its storage interface, its use-cases, and the events it produces
and consumes, and it never imports another domain directly. Most domains
follow a five-file shape: `model.go`, `repo.go`, `usecase.go`, `handler.go`,
`events.go`. Domains with no events yet, or a thinner shape, are exempt until
they need it. See [Repository Layout](/docs/contributing/repository-layout/)
for the full directory map.

## Two adapters, one use-case layer

Every capability the product offers is a use-case function. Two adapters
call it, and neither duplicates it:

- **HTTP/JSON gateway** — what the browser talks to, and what a third-party
  integration talks to with a scoped API token instead of a session.
- **MCP server** — the Model Context Protocol adapter LLM agents use to
  drive the product: search docs, create tickets, read topology, trigger
  deploys, replay a dead letter.

If a capability exists in the UI, it exists in MCP by construction, and vice
versa. Neither adapter carries business logic of its own — each parses
input, calls the use-case, and formats the output.

## SQLite is the spine

One SQLite file, one writer, WAL mode for concurrent readers, FTS5 for
full-text search over docs and tickets. Queries are written by hand in
`.sql` files under `internal/platform/storage/queries/`, and `sqlc` compiles
them into `internal/platform/storage/sqlcgen/` against the migrations
directory — no Postgres, no dual-backend abstraction.

## The EventBus is the microservice seam

Domains that need to react to each other's changes do it through the
`EventBus` interface in `internal/platform/eventbus/`, never through direct
imports. Today the bus is in-process and channel-backed — one process, zero
network hops. If a domain ever needs to become its own service, the bus
implementation swaps to NATS via Watermill and the domain code doesn't
change: it still calls `Publish`/`Subscribe` on the same interface. Critical
events go through a transactional outbox, written in the same SQLite
transaction as the domain change, so an event can never be lost or published
for a change that rolled back.

## The web app

`web/` is React 19 + Vite + Tailwind, built with shadcn/ui components. It
talks to the server over the HTTP/JSON gateway for requests and a WebSocket
for live events — never over MCP; the browser never speaks JSON-RPC.

## Desktop

`desktop/` is a thin Electron shell over the same served web app. It exists
for consistent rendering on Linux, not to host a second UI: it bootstraps a
connection token and otherwise stays out of the way.

## SDK and automations

`sdk/` is the TypeScript package automations are written against. It contains
the API client, generated event types, config schema, testing surface, and a
Bun CLI with `nexul init`, `nexul dev`, `nexul push`, and `nexul pull`.
`automations/` is the small container bundled with every instance that runs
Default automations and small Custom ones. Bigger Custom automations run
wherever their owner deploys them.
