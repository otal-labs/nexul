---
title: Local Development
description: Prerequisites, make targets, the debug stack, and running tests.
sidebar:
  order: 3
---

## Prerequisites

- **Go** — `go.mod` pins the Go language version and exact toolchain. Use
  those values rather than a separately maintained version list.
- **Bun** — CI pins the version in `.github/workflows/ci.yml`. Use that same
  version for `web/`, `desktop/`, `sdk/`, and `automations/`.

## Make targets

The root `Makefile` covers the Go binaries and the web build:

```sh
make build         # build-server + build-runner + build-web
make build-server   # ./dist/nexul-server
make build-runner   # ./dist/nexul-runner
make build-web      # bun run --cwd web build
make build-single    # single-binary server, web assets embedded via go:embed
make test           # go test ./...
make vet            # go vet ./...
make coverage       # go test -race with the 80% gate
make sqlc           # regenerate sqlcgen from internal/platform/storage/queries/*.sql
make sqlc-check     # sqlc vet + sqlc diff — fails if generated code is stale
```

## Running the server and the web dev server

Outside the debug stack, run the server directly against Go:

```sh
go run ./server/cmd
```

The server generates its auth secret and reads its config from environment
variables (see `.env.example` — nothing is required for a local run; every
variable there is an override).

The web app has its own dev server:

```sh
bun install
bun run --cwd web dev
```

## The debug compose stack

`docker-compose.debug.yml` runs the server, web (Vite, not nginx), runner,
automations host, and an OpenObserve instance for logs together, with Delve
attached to the Go binaries (`:2345` server, `:2346` runner):

```sh
docker compose -f docker-compose.debug.yml up
```

Web (`:5173`), server HTTP/WS/MCP (`:8080`/`:8081`/`:8082`), and OpenObserve
(`:5080`) are all reachable on the host.

**Named `node_modules` volumes.** `web-node-modules` and
`automations-node-modules` are named Docker volumes, not bind mounts —
they exist so the container's installed dependencies don't get shadowed by
whatever (or nothing) is in your host checkout's `node_modules`. That means
a host-side `bun install` doesn't reach the container: after changing
`package.json` in `web/` or `automations/`, install inside the running
container instead:

```sh
docker compose -f docker-compose.debug.yml exec web bun install
docker compose -f docker-compose.debug.yml exec automations bun install
```

Never run `docker compose -f docker-compose.debug.yml down -v` to pick up a
dependency change — that also wipes the debug database and OpenObserve
volumes. `exec ... bun install` is the fix; `down -v` is not.

## Running tests

```sh
go test ./...                       # Go, all packages
bun run --cwd web test              # web, Vitest
bun run --cwd desktop test          # desktop, Vitest
bun run --cwd sdk test              # sdk, Vitest
bun run --cwd automations test # automations, bun test
```

## The coverage gate

`make coverage` is the enforced gate, not a suggestion:

```sh
make coverage
```

It runs `go test -race -coverprofile=coverage.out -covermode=atomic ./...`.
The Makefile filters `cmd/*`, `testutil/`, and `sqlcgen/` from the profile,
computes the percentage over the remaining statements, and fails below 80%.
It writes `coverage.filtered.out` and `coverage.html`, the same artifacts CI
uploads. See
[Coding Standards](/docs/contributing/coding-standards/) for what the gate
expects beyond the number.
