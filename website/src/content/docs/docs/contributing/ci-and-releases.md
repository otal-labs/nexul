---
title: CI and releases
description: What runs on every push, how releases are cut and versioned, and the merge policy.
sidebar:
  order: 6
---

## Continuous integration — `.github/workflows/ci.yml`

CI runs on pushes and pull requests against `master`, and it only runs the
jobs a change actually touches. A `dorny/paths-filter` step tags the diff
against seven filters, and each job is gated on its own tag:

| Filter | Paths | Job |
|---|---|---|
| `go` | `server/**`, `runner/**`, `internal/**`, root `*.go`, `go.mod`, `go.sum`, `sqlc.yaml`, `.goreleaser.yaml` | `go-test` |
| `web` | `web/**`, `client-core/**` | `web-test` |
| `desktop` | `desktop/**` | `desktop-test` |
| `website` | `website/**` | `website-build` |
| `sdk` | `sdk/**`, `internal/eventcatalog/schemas.json` | `sdk-test` |
| `automations` | `automations/**` | `automations-test` |
| `native` | `native/**`, `client-core/**`, `sdk/src/events.generated.ts` | `native-test` |

- **`go-test`** — checks the committed `sqlcgen` output is current
  (`sqlc vet` + `sqlc diff`), builds, vets, lints, checks dependencies for
  known vulnerabilities (`govulncheck`), validates
  `.goreleaser.yaml` with `goreleaser check`, then runs the coverage gate
  (`make coverage`). Uploads `coverage.filtered.out` and
  `coverage.html` as the `go-coverage` artifact.
- **`web-test`** — installs with a frozen lockfile, type-checks, lints,
  runs the test suite with coverage, then builds. `client-core/` is linted,
  tested and counted toward the coverage gate here. Uploads
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
- **`native-test`** — installs with a frozen lockfile, type-checks, lints,
  then runs the phone app's Jest suite. No APK is built in CI.

A change that touches only `web/` never spins up a Go job, and vice versa.

## Releases — `.github/workflows/release.yml`

Nexul has one version for the whole product, and git tags are that version
(ADR 0070). There is no version file in the repository.

- **Beta** runs once a day at 00:07 UTC (ADR 0071). When `master` has moved
  since its last release, it tags the head `v<line>.<n>-beta` (ADR 0084):
  `<line>` is `0.3` until a stable release starts a newer one, and `<n>` is
  one past the highest patch already tagged on that line, beta or stable
  (`v0.3.0-beta`, `v0.3.1-beta`, ...). When `master` hasn't moved, the run
  stops before building anything. A beta's release notes list
  every pull request merged since the previous beta. To cut one now, run the
  workflow by hand with `channel: beta`.
- **Stable** is a manual run (`channel: stable`) with a `bump` input.
  `patch` releases the newest beta under its own number (`v0.3.12-beta`
  becomes `v0.3.12`); `minor` or `major` start a new line (`v0.4.0`), and
  betas continue at `v0.4.1-beta`. It builds the commit of the newest beta, so stable only ever ships a commit a
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

## The phone app — `native-release.yml` and `native-update.yml`

The phone app has its own version, `version` in `native/app.config.ts`,
and its own tags, `phone-v<version>-beta`, next to the server's `v…` tags.
Both workflows release only by hand (`workflow_dispatch`); nothing about the
app ships on push or on a schedule.

- **`native-release.yml`** builds a signed Android APK (`expo prebuild`,
  then `gradlew assembleRelease` with the keystore from the repository
  secrets) and an unsigned iOS IPA on a macOS runner (`expo prebuild`,
  `pod install`, then `xcodebuild archive` with signing off), and creates one
  pre-release, `phone-v<version>-beta`, with `nexul-android-<version>.apk`
  and `nexul-ios-<version>.ipa` attached. The run asks for the version and
  refuses when it differs from `app.config.ts` or when the tag already
  exists. There is no Apple Developer account: SideStore signs the IPA on the
  iPhone with a free Apple ID at install time. The release is never marked
  latest, and the server's release client, `install.sh` and this site's
  changelog ignore every tag that does not start with `v` (the older
  `android-v…` releases too), so a phone release cannot become the server's
  latest. A pull request that touches `native/` builds the IPA as an
  artifact and releases nothing.
- **`native-update.yml`** publishes the JavaScript and assets of the
  current commit, for Android and iOS, to the self-hosted update server's `production` branch as
  an over-the-air update. Phones running the same app version pick it up on
  their next launch. The run stops with an error before installing anything
  when the update server's token or variables are missing.
- **Which one to run:** a change to a native dependency or to
  `app.config.ts` bumps `version` and needs a new phone release, because the runtime
  version follows the app version and an update never crosses versions. A
  JavaScript-only change ships as an update within the current version.
- **Secrets:** `ANDROID_KEYSTORE_BASE64` (the PKCS12 release keystore,
  base64), `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`,
  `ANDROID_KEY_PASSWORD`, and `EOO_TOKEN` (a publish token from the update
  server's dashboard). The keystore never changes: an APK signed with another
  key cannot install over the old one.
- **Variables:** `NEXUL_UPDATES_URL`, the update server's manifest URL
  (`https://<update server>/manifest`), and `NEXUL_UPDATES_APP_ID`, the app's
  id in the update server's dashboard. Both are baked into every build and
  read by the publish step, so a build without them never receives updates.
- **Code signing of updates:** once the update server's certificate is
  committed as `native/certs/certificate.pem`, the app config adds it and
  every update must be signed by the server. Builds work without it; the
  first APK built with it is a new version.

## Merge policy

Merges to `master` are squash-only — a pull request lands as one commit,
whatever its branch history looked like.
