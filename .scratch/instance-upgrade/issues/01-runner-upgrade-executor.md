# 01 — Runner side: `Executor.Upgrade` and the client dispatch

**Status:** resolved
**Type:** task
**Blocked by:** None — the frame types already exist in `protocol.go`.

## Scope

Runner-side only (`internal/runner/executor.go`, `client.go`, and their
tests). The frame constants `FrameAssignUpgrade`, `FrameUpgradeProgress`,
`FrameUpgradeResult`, their `Validate()` cases, and `RequestUpgrade` are
already committed on the feature branch; do not redefine them.

## Build

- Add `Upgrade(ctx, req Frame, send func(Frame) error) error` to the
  `Executor` interface and `ShellExecutor`, following the spec's "Runner
  behaviour" list exactly (inspect own container via `os.Hostname()`, read
  labels `com.docker.compose.project`, `com.docker.compose.project.working_dir`,
  `com.docker.compose.project.config_files` (comma-separated) and
  `.Config.Image`, `docker rm -f nexul-upgrade`, `docker run -d --name
  nexul-upgrade …` with the compose pull + up script, progress frames per
  step, `upgrade_result started` with the container id). Missing labels →
  `upgrade_result failed` with `instance runner is not managed by docker compose`.
- Allow an env override `NEXUL_RUNNER_CONTAINER` for the container id
  (tests and non-default hostnames).
- `client.go`: dispatch `FrameAssignUpgrade` through the same single-job
  path as `assign_deploy` (`startJob`), so a deploy cannot run at the same
  time; mark `jobFinished` after `upgrade_result` is sent.
- Tests with the existing fake `CommandRunner`: label parsing, the exact
  docker commands (assert argument lists, not substrings), the failure path,
  and the client dispatch. No real docker.

## Acceptance

- `go test ./internal/runner/...` green with race; coverage does not drop.
- The `docker run` invocation matches the spec argument for argument.
