---
title: CI and Releases
description: What runs on every push, how the runner release workflow fires, and the merge policy.
sidebar:
  order: 6
---

## Continuous integration — `.github/workflows/ci.yml`

CI runs on pushes and pull requests against `master`, and it only runs the
jobs a change actually touches. A `dorny/paths-filter` step tags the diff
against six filters, and each job is gated on its own tag:

| Filter | Paths | Job |
|---|---|---|
| `go` | `server/**`, `runner/**`, `internal/**`, `go.mod`, `go.sum`, `sqlc.yaml` | `go-test` |
| `web` | `web/**` | `web-test` |
| `desktop` | `desktop/**` | `desktop-test` |
| `website` | `website/**` | `website-build` |
| `sdk` | `sdk/**`, `internal/integrations/catalog.go` | `sdk-test` |
| `automations` | `automations/**` | `automations-test` |

- **`go-test`** — checks the committed `sqlcgen` output is current
  (`sqlc vet` + `sqlc diff`), builds, vets, then runs the coverage gate
  (`make coverage`). Uploads `coverage.filtered.out` and
  `coverage.html` as the `go-coverage` artifact.
- **`web-test`** — installs with a frozen lockfile, type-checks, lints,
  runs the test suite with coverage, then builds. Uploads
  `web/coverage/lcov.info` as `web-coverage`.
- **`desktop-test`** — same shape as `web-test`, with
  `ELECTRON_SKIP_BINARY_DOWNLOAD=1` so CI never downloads the Electron
  binary — the desktop unit tests never launch it.
- **`website-build`** — installs with a frozen lockfile, runs `bun run test`,
  then runs `bun run build`, so a broken page or frontmatter fails the check.
- **`sdk-test`** — installs with a frozen lockfile, runs the SDK typecheck,
  then runs its Vitest suite.
- **`automations-test`** — installs with a frozen lockfile, runs the
  automations host typecheck, then runs its Bun test suite.

A change that touches only `web/` never spins up a Go job, and vice versa.

## Releases — `.github/workflows/release.yml`

Nexul has one version for the whole product, not one per component.
The `VERSION` file at the repo root holds the version line currently in
beta, with its suffix: `0.2.0-beta`.

- **Beta** runs on every push to `master`, so every squash-merge is a
  release. It tags a prerelease `v<VERSION>-<NNN>` (`v0.2.0-beta-001`,
  `v0.2.0-beta-002`, ...) from the pushed commit. The number is the newest
  published beta on that `VERSION` line plus one, read from the releases
  list rather than from a file, so queued runs never collide and nothing
  commits back to `master`. Beta never touches `VERSION`.
- **Stable** is a manual `workflow_dispatch` (`channel: stable`). It does not
  build master directly: it resolves the commit of the newest published
  beta and builds *that* commit, so stable only ever ships a commit a
  beta has already carried. It tags the release `v<VERSION>` with the
  `-beta` suffix dropped (a `version` input can override this), then bumps
  `VERSION`'s patch component, keeping the suffix (`0.2.0-beta` becomes
  `0.2.1-beta`), and pushes that to master as `chore(release): prepare next
  version`. Pushing a bare `v*.*.*` tag by hand runs the same stable build
  against that exact tag, without the `VERSION` bump — a hand-pushed tag can
  target an old commit, so it must never move `VERSION` forward.
- **What a release carries:** four GHCR images (`nexul-server`,
  `-web`, `-runner`, `-automations`, tagged with the release version
  plus the moving `beta` or `latest` tag; until the first stable release
  exists, betas carry `latest` too, so a default install pulls the newest
  beta), one runner binary per
  supported OS/arch (`nexul-runner-<goos>-<goarch>[.exe]`:
  linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), the
  same five targets as single-binary server tarballs
  (`nexul-server-<goos>-<goarch>.tar.gz`, embedding the web frontend
  via `go:embed`), and a `checksums.txt` covering every binary asset. The
  runner binary naming is a contract: the server's download proxy serves
  its own version's assets under these exact names.
- **Beta image cleanup:** a `prune` job runs after beta image pushes
  and deletes old beta-tagged GHCR image versions, keeping the ten
  newest per image and never touching a `latest` or stable-semver tag.
- Every Go build in this workflow is stamped with the release version via
  `-ldflags -X .../internal/platform/version.Version=...`, and the runner's
  download proxy (`GET /api/runners/download/{target}`) now serves the
  server's own version rather than a separately-versioned runner release.

## Merge policy

Merges to `master` are squash-only — a pull request lands as one commit,
whatever its branch history looked like.
