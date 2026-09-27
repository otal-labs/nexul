---
title: CI and Releases
description: What runs on every push, how releases are cut and versioned, and the merge policy.
sidebar:
  order: 6
---

## Continuous integration — `.github/workflows/ci.yml`

CI runs on pushes and pull requests against `master`, and it only runs the
jobs a change actually touches. A `dorny/paths-filter` step tags the diff
against six filters, and each job is gated on its own tag:

| Filter | Paths | Job |
|---|---|---|
| `go` | `server/**`, `runner/**`, `internal/**`, root `*.go`, `go.mod`, `go.sum`, `sqlc.yaml`, `.goreleaser.yaml` | `go-test` |
| `web` | `web/**` | `web-test` |
| `desktop` | `desktop/**` | `desktop-test` |
| `website` | `website/**` | `website-build` |
| `sdk` | `sdk/**`, `internal/integrations/catalog.go` | `sdk-test` |
| `automations` | `automations/**` | `automations-test` |

- **`go-test`** — checks the committed `sqlcgen` output is current
  (`sqlc vet` + `sqlc diff`), builds, vets, lints, validates
  `.goreleaser.yaml` with `goreleaser check`, then runs the coverage gate
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
  The build reads the GitHub releases for `/changelog/`, so it needs network
  access to the GitHub API.
- **`sdk-test`** — installs with a frozen lockfile, runs the SDK typecheck,
  then runs its Vitest suite.
- **`automations-test`** — installs with a frozen lockfile, runs the
  automations host typecheck, then runs its Bun test suite.

A change that touches only `web/` never spins up a Go job, and vice versa.

## Releases — `.github/workflows/release.yml`

Nexul has one version for the whole product, and git tags are that version
(ADR 0070). There is no version file in the repository.

- **Beta** runs once a day at 03:17 UTC (ADR 0071). When `master` has moved
  since its last release, it tags the head `v<next>-beta.<n>`: `<next>` is
  the newest stable tag with its patch bumped (`0.2.0` while no stable
  release exists) and `<n>` is one more than the highest beta already tagged
  on that line (`v0.2.1-beta.1`, `v0.2.1-beta.2`, ...). When `master` hasn't
  moved, the run stops before building anything. A beta's release notes list
  every pull request merged since the previous beta. To cut one now, run the
  workflow by hand with `channel: beta`.
- **Stable** is a manual run (`channel: stable`) with a `bump` input:
  `patch`, `minor` or `major`, counted from the last stable release. It
  builds the commit of the newest beta, so stable only ever ships a commit a
  beta has already carried, and it refuses when that commit is already
  released. Pushing a bare `vX.Y.Z` tag by hand builds that tag as stable.
- **How a release is built:** the workflow builds the web UI into
  `server/webui/dist`, compiles the automations host for every target with
  `bun build --compile` (`bun run --cwd automations build:binaries`), tags
  the commit, and runs GoReleaser (`.goreleaser.yaml`), which builds the Go
  binaries and publishes everything to the GitHub release.
- **What a release carries:** four binaries per platform (linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), named
  `<binary>-<os>-<arch>[.exe]`: `nexul`, the install and management command;
  `nexul-server`, the server with the web UI embedded; `nexul-runner`; and
  `nexul-automations`, the automations host. A `checksums.txt` covers every
  binary. No container images are released. The binary names are a
  contract: `install.sh`, `nexul install`, `nexul upgrade`, the runner
  download proxy and runner self-update fetch these exact names.
- **Changelog rebuild:** a `website` job runs after each release and calls
  the Cloudflare Pages deploy hook in the `CLOUDFLARE_PAGES_DEPLOY_HOOK`
  secret, so nexul.io/changelog lists the new release. Without the secret the
  job skips, and the changelog catches up on the next site build.
- The Go binaries are stamped with the release tag via
  `-ldflags -X .../internal/platform/version.Version=...`; `nexul version`
  prints it.

## Merge policy

Merges to `master` are squash-only — a pull request lands as one commit,
whatever its branch history looked like.
