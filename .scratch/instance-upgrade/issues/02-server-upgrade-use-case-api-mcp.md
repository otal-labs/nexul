# 02 — Server side: upgrade record, dispatch, HTTP, MCP, boot resolution

**Status:** resolved
**Type:** task
**Blocked by:** None — the frame types already exist in `protocol.go`.

## Scope

`internal/runner` (usecase, handler, events, repo, mcp), storage
(migration + `queries/runner.sql` + sqlc regen), `server/cmd` (routes,
OpenAPI registration, boot hook), event catalog.

## Build

- Migration `0135_instance_upgrades.sql` and sqlc queries: create, get
  latest, update status/error, list unresolved. Regenerate with `make sqlc`
  and commit the generated code.
- Model `Upgrade` + statuses `pending|started|completed|failed` in the
  runner domain; repo interface + storage implementation + fake for tests.
- `Service.UpgradeStatus(ctx)` and `Service.RequestUpgrade(ctx, actor)`
  implementing the spec's reason list, using `ReleaseClient().Latest` and
  `version.Version`; the lazy 15-minute resolution lives in
  `UpgradeStatus`. `Service.ResolvePendingUpgrade(ctx)` for boot.
- Handler: send `assign_upgrade` to the connected runner with id
  `instance` and hold its job slot (add an upgrade id to `runnerConn`
  beside the deploy job; `idleConnected` treats it as busy). Handle inbound
  `upgrade_progress` (log) and `upgrade_result` (status). Disconnect while
  `pending` → failed `runner disconnected`; while `started` → no change.
- Topic `instance.upgrade_changed` in `events.go`, registered in the event
  catalog like `deploy.status_changed`; publish on every status change.
- HTTP in `server/cmd/version_http.go` (next to `/api/version`):
  `GET /api/instance/upgrade` and `POST /api/instance/upgrade`, gated by
  the instance admin fact (`instanceAdminGate`, see `wire_gates.go`),
  registered with `spec.Register` in `routes.go`. Response shapes exactly
  as in the spec.
- Boot: call `ResolvePendingUpgrade` in `server/cmd/bootstrap.go` after
  migrations.
- MCP: `instance_upgrade` (calls `RequestUpgrade` with the token's user +
  `:mcp`, ADR 0049) and `instance_upgrade_status` in `internal/runner/mcp.go`.

## Acceptance

- `go test ./...` green with race; coverage gate holds; `make sqlc-check`
  clean.
- Handler tests cover: dispatch to the instance runner, busy slot,
  disconnect in each status, boot resolution on matching and non-matching
  versions, the 15-minute failure.
