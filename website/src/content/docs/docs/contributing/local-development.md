---
title: Local development
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
make build          # build-cli + build-server + build-runner + build-web
make build-cli      # ./dist/nexul, the install and upgrade command
make build-server   # ./dist/nexul-server, without the web UI
make build-runner   # ./dist/nexul-runner
make build-web      # bun run --cwd web build
make build-single   # ./dist/nexul-server as released, web UI embedded via go:embed
make test           # go test ./...
make vet            # go vet ./...
make lint           # golangci-lint
make vuln           # govulncheck
make coverage       # go test -race with the 80% gate
make sqlc           # regenerate sqlcgen from internal/platform/storage/queries/*.sql
make sqlc-check     # sqlc vet + sqlc diff — fails if generated code is stale
make live-topics    # rewrite the web's copy of the topics the live socket pushes
make event-schemas  # regenerate the event contract and the SDK's event types
```

The automations host binaries come from Bun, not the Makefile:
`bun run --cwd automations build:binaries` writes
`automations/dist/nexul-automations-<os>-<arch>[.exe]` for every release
target.

## Running the server and the web dev server

Outside the debug stack, run the server directly against Go:

```sh
go run ./server/cmd
```

The server generates its auth secret and reads its config from environment
variables (see `.env.example` — nothing is required for a local run; every
variable there is an override). It listens on `:8080` unless
`NEXUL_HTTP_ADDR` says otherwise.

## Trying the installer

`nexul install` downloads every binary from a release, checked against its
`checksums.txt`. To try it against binaries you built, serve a directory laid
out like a release (`<tag>/<binary>-<os>-<arch>` plus `<tag>/checksums.txt`)
and point the installer at it with `NEXUL_RELEASE_URL`, passing the tag with
`--version`. Do this on a disposable machine or VM: the install creates
services and a system user.

The web app has its own dev server:

```sh
bun install
bun run --cwd web dev
```

## The debug compose stack

The debug stack is for developing Nexul only; releases ship no images, and a
real install runs every component as a native service.
`docker-compose.debug.yml` runs the server, web (Vite, not nginx), runner,
automations host, and an OpenObserve instance for logs together, with Delve
attached to the Go binaries (`:2345` server, `:2346` runner):

```sh
docker compose -f docker-compose.debug.yml up
```

Web (`:5173`), the server (`:8080`: the API, MCP at `/mcp`, and the runner WebSocket at `/ws/runner`), and
OpenObserve (`:5080`) are all reachable on the host.

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

## Pairing a computer locally

A computer pairs through its personal runner, so a local or end-to-end check
runs one by hand against the debug stack or a local server, reaching a
throwaway T3 Code so your own stays untouched. Keep the scratch files in a
git-excluded folder such as `.verify/`.

1. Start a throwaway T3 Code in a base directory of its own, on a port of
   its own, and give it a project:

   ```sh
   t3 serve --base-dir "$PWD/.verify/t3home" --port 47190 --host 127.0.0.1 --no-browser
   t3 project add --base-dir "$PWD/.verify/t3home" <a git folder>
   ```

   Any T3 Code build works: a release's AppImage, extracted with
   `--appimage-extract`, runs as
   `ELECTRON_RUN_AS_NODE=1 squashfs-root/t3code squashfs-root/resources/app.asar/apps/server/dist/bin.mjs`.
   Put a small `t3` script that runs that command first on `PATH`, because
   the runner mints its pairing token with the first `t3` it finds.
2. Add a computer as yourself: `POST /api/pairing/computers/enrollments`
   with `{}` returns the computer and its `token`.
3. Trade the token for the runner's credential and save it to a file:

   ```sh
   curl -s -X POST http://127.0.0.1:8080/api/runners/enroll \
     -H 'Content-Type: application/json' \
     -d '{"token":"<token>","machine":"'"$(hostname)"'","os":"linux","arch":"amd64"}'
   ```

   The answer holds `name` and `credential`; the credential is shown once.
4. Run the runner in personal mode, pointed at the throwaway's base
   directory:

   ```sh
   NEXUL_SERVER_URL=http://127.0.0.1:8080 NEXUL_CREDENTIAL_FILE=.verify/credential \
   NEXUL_RUNNER_MODE=personal NEXUL_RUNNER_NAME=<name> T3CODE_HOME="$PWD/.verify/t3home" \
   go run ./runner/cmd
   ```

   A runner with a custom `T3CODE_HOME` reads the throwaway's port from its
   runtime file and never falls back to 3773, so once the throwaway stops,
   its dials are refused instead of reaching your own T3 Code.

Within seconds the runner reports T3 Code answering and the computer pairs
on its own: `GET /api/pairing/computers` shows a `token_expires_at`, or
`pair_error` saying why it could not. `POST /api/pairing/computers/{id}/pair`
with no body pairs it again now. Stop each process by the PID you started it
with.

## Running tests

```sh
go test ./...                       # Go, all packages
bun run --cwd web test              # web and client-core, Vitest
bun run --cwd native test           # phone app, Jest
bun run --cwd desktop test          # desktop, Vitest
bun run --cwd sdk test              # sdk, bun test
bun run --cwd automations test # automations, bun test
```

## The coverage gate

`make coverage` is the enforced gate, not a suggestion:

```sh
make coverage
```

It runs `go test -race` with a coverage profile over every package but
`internal/platform/storage`, whose tests run without `-race` because the
detector makes the pure-Go SQLite engine about 25 times slower; the
`server/cmd` integration tests still drive storage under `-race`. The
Makefile filters `cmd/*`, `testutil/`, `sqlcgen/`, and `t3rpctest/` from the
profile, computes the percentage over the remaining statements, and fails
below 80%.
It writes `coverage.filtered.out` and `coverage.html`, the same artifacts CI
uploads. See
[Coding standards](/docs/contributing/coding-standards/) for what the gate
expects beyond the number.
