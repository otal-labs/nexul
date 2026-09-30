# 08 — Runner log frames, `stacks:logs`, and the logs routes

**Type:** task
**Status:** done
**Blocked by:** None — can start immediately

## What to build

The backend of container logs, per decisions 01 to 04:

- **Runner frames and executor** (`internal/runner`):
  - `logs_request`, `logs_chunk`, `logs_end` and `logs_cancel`, with
    validators in `frameValidators`.
  - The executor runs `docker logs --timestamps --tail <n> [--follow]
    <container>` with separate stdout and stderr pipes. Each line becomes
    `{ts, stream, line}`.
  - Chunks batch every 250ms or at 32KB.
  - A stream sends at most 64KB/s, with a "N lines skipped" line in place of
    what it drops. A runner holds at most 16 streams and refuses the next
    with a readable error.
  - `logs_cancel`, or a disconnect, kills the process.
- **Server side** (`internal/runner/handler.go`):
  - A request/stream registry keyed by id, like `discoverWaiters` but
    multi-frame.
  - A use-case in `internal/deploy`, `ServiceLogs(ctx, stackID, service,
    tail, follow)`, that resolves the container name and machine and returns
    a line channel plus a cancel function.
- **Permission:** a new verb `stacks:logs` (ADR 0057 pattern).
  - A forward-only migration grants it to every role that holds
    `stacks:write`, as a backfill.
  - It gets an `Access.tsx` entry and a label in the permission catalog.
- **Masking:** before a line leaves the server, every value of the stack's
  env that is 6 or more characters long is replaced with `••••`.
- **HTTP snapshot:** `GET /api/stacks/{id}/services/{name}/logs?tail=N`
  (default 200, max 1000) returns `{lines: [{ts, stream, line}]}`.
- **Live:** `GET /ws/stacks/{id}/services/{name}/logs?tail=N`, one socket per
  viewer.
  - Auth is the session cookie, or `?token=` as `/ws/events` does it.
  - Open starts a following stream and close cancels it.
  - A slow socket gets a "skipped" line instead of a growing buffer.
- **Guards:** both routes check `stacks:logs` against the stack's project
  (ADR 0087) and are not on the setup-pass allowlist.
- **ADR:** one ADR, "Container logs are read live from Docker, not stored",
  with the deferred-history trigger. Add the terms to `CONTEXT.md` if new.

## Acceptance criteria

- [ ] Frame round trip over a real `ShellExecutor` with `docker logs` faked
      at the command boundary: tail, follow, cancel kills the process, and
      the stderr lines are tagged.
- [ ] The rate cap and the stream cap produce their readable lines and
      errors.
- [ ] The snapshot route returns masked lines. Without `stacks:logs` it
      returns 403, and with only a setup pass it returns 401.
- [ ] Closing the WebSocket cancels the runner stream (integration test with
      a fake runner connection).
- [ ] The migration grants `stacks:logs` to existing roles with
      `stacks:write`, and an upgrade from the previous schema is tested.
- [ ] `go test ./...`, `make lint`, `make coverage` and `make sqlc-check`
      are green.
