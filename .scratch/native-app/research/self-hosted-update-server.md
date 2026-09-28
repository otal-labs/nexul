# Self-hosted update server: what running Xprem takes

Researched 2026-09-28 against the Xprem repository at commit `f9bd112`
(2026-09-27), release v3.2.5 (2026-09-26), its documentation for v3.2.0+
(https://mercure-technologies.gitbook.io/xprem, index at `/llms.txt`), the npm
registry, and a local boot of the v3.2.5 image. The "OTA" section of
`react-native-best-practices.md` covers the protocol, store policy, pricing and
the per-instance question; none of that is repeated here.

## Summary

Xprem runs as one Go container plus Postgres. It can keep updates on a local
volume, needs no Expo account and no EAS CLI to publish, and signs every
manifest with a key it generates itself. Rollouts, rollbacks, channels and the
dashboard are all in the MIT core. The one serious risk is the bus factor: one
maintainer wrote nearly all of it. The open protocol limits that risk, because
another server can take over behind the same URL. Recommended.

## Identity and location

- The repository was renamed. It is now `mercuretechnologies/xprem`; the old
  `expo-open-ota` URL answers with a 301 redirect there (checked 2026-09-28). The description reads "Formerly
  expo-open-ota". Source: https://github.com/mercuretechnologies/xprem (repo
  metadata, 2026-09-28)
- Website https://xprem.dev. Docs https://mercure-technologies.gitbook.io/xprem
  (v3.2.0 is the latest docs version). CLI on npm: `eoas`, latest `3.2.5`
  (2026-09-26). Source: https://registry.npmjs.org/eoas

## Two operating modes (this decides the Expo-account question)

| | Stateless (`DB_URL` unset) | Control plane (`DB_URL` set) |
|---|---|---|
| Expo account | Required: `EXPO_APP_ID` + `EXPO_ACCESS_TOKEN`; the server reads channels and branches from the Expo API and checks every publish against Expo | Not needed. Apps, branches and channels live in Postgres; the CLI authenticates with `EOO_TOKEN`, a dashboard-issued API key |
| Apps per server | One | Many |
| Rollouts, branch surfing, bundle diffing, API tokens | No | Yes |

Sources: https://mercure-technologies.gitbook.io/xprem/stateless-mode/overview.md,
https://mercure-technologies.gitbook.io/xprem/eoas/overview.md, and the code
paths `config/apps.go` (stateless reads `EXPO_APP_ID`/`EXPO_ACCESS_TOKEN`) and
`apps/eoas/src/lib/auth.ts` (the CLI uses `EOO_TOKEN` when set and otherwise
falls back to Expo credentials) at `f9bd112`.

**Use control-plane mode.** It is the only mode that works without an Expo
account. Postgres is the price.

## Install

- Image: `ghcr.io/mercuretechnologies/xprem:v3.2.5`, multi-arch (linux/amd64
  and linux/arm64, checked against the registry manifest), about 87 MB. It runs
  as uid 100 / gid 101 and listens on port 3000. `latest` is a moving tag; pin
  the version. Sources: release notes
  https://github.com/mercuretechnologies/xprem/releases/tag/v3.2.5,
  `Dockerfile` and `.github/workflows/release.yml` at `f9bd112`
- Images are cosign-signed from v3.2.3 on (keyless, GitHub Actions identity).
  Source: https://mercure-technologies.gitbook.io/xprem/security/signed-releases.md
- Also available: a Helm chart (`oci://ghcr.io/mercuretechnologies/charts/xprem`),
  a single static Go binary, and a Railway template. `npx eoas server:init`
  writes a `.env.xprem` holding only the variables your choices need. Sources:
  README at `f9bd112`,
  https://mercure-technologies.gitbook.io/xprem/installation-guide/overview.md
- The repository's `docker-compose.yml` is for development (it builds from
  source with hot reload, Prometheus and ClickHouse), not for deployment. The
  documented production path is `docker run --env-file`. The Quickstart pairs
  that with `postgres:16`. Sources: `docker-compose.yml` at `f9bd112`,
  https://mercure-technologies.gitbook.io/xprem/quickstart.md,
  https://mercure-technologies.gitbook.io/xprem/deployment/docker-image.md
- Migrations run on boot. The server does not create the database itself: the
  database named in `DB_URL` must already exist, which `POSTGRES_DB` takes care
  of. Source:
  https://mercure-technologies.gitbook.io/xprem/installation-guide/database-configuration.md

## Environment variables and secrets (control plane)

Source for all rows:
https://mercure-technologies.gitbook.io/xprem/references/environment-variables.md,
cross-checked with `config/config.go` at `f9bd112`.

| Variable | Needed | Notes |
|---|---|---|
| `BASE_URL` | Yes in practice | Public HTTPS URL; manifest and asset URLs are built from it. Defaults to `http://localhost:3000` |
| `JWT_SECRET` | Yes (boot fails without it) | `openssl rand -base64 32`. Signs dashboard sessions and upload tokens |
| `DB_URL` | Yes | Postgres DSN; its presence selects control-plane mode |
| `DB_KEYS_MASTER_KEY_B64` | Yes (exactly one master-key source) | `openssl rand -base64 32`. Seals the per-app signing keys in Postgres. **Not recoverable**: losing it means new keys, a new build, and every user reinstalling |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | First boot only | Seed the first admin. The password needs 8+ characters with upper, lower, digit and symbol, or the boot fails. Never read again afterwards |
| `USE_DASHBOARD` | Optional | `true` serves the dashboard at `/dashboard` |
| `STORAGE_MODE` | Optional | `local` (default), `s3`, `gcs`, `azure` |
| `LOCAL_BUCKET_BASE_PATH` | With `local` | Defaults to `./updates` (`/app/updates` in the image) |
| `CACHE_MODE` | Optional | In-memory unless `redis`/`redis-sentinel`; in-memory is fine for one replica |
| `DISABLE_TELEMETRY` | Optional | `true` turns off the hourly usage ping (instance id, base URL, version, config shape). It is on by default |
| `DISABLE_DEVICE_TELEMETRY` | Optional | `true` stops recording device check-ins in Postgres |
| `BUNDLE_DIFFING` | Optional | `true` computes bsdiff patches (control plane only) |
| `PROMETHEUS_ENABLED` | Optional | Exposes `/metrics` |

## Storage

- Backends: local filesystem, S3 and S3-compatible (R2, MinIO, DigitalOcean
  Spaces, Supabase; set `AWS_BASE_ENDPOINT`, `AWS_REGION=auto`, and
  `AWS_S3_FORCE_PATH_STYLE=true` where the provider needs it), GCS, and Azure.
  Sources: https://mercure-technologies.gitbook.io/xprem/storage/s3-storage.md,
  README at `f9bd112`
- Local storage is "not recommended for production", but the only reason given
  is that replicas cannot share files. A single container on one host is the
  case it does support. The mount must be writable by uid 100:101. A named
  Docker volume mounted on `/app/updates` works, because the image pre-creates
  that directory with the right owner and Docker copies ownership into an empty
  named volume on first mount. Sources:
  https://mercure-technologies.gitbook.io/xprem/storage/local-storage.md,
  `Dockerfile` at `f9bd112`,
  https://docs.docker.com/engine/storage/volumes/#populate-a-volume-using-a-container
- With local storage and no CDN, the server streams assets itself with
  `Cache-Control: public, max-age=31536000`. Manifests are sent `private,
  no-store`, so a caching proxy such as Cloudflare can cache assets but never
  manifests. Sources: `internal/assets/assets.go:108`,
  `internal/services/expo_protocol_service.go:515` at `f9bd112`
- Since v3.2.0, assets are content-addressed (`appId/cas/{sha256}`) and shared
  across updates, so republishing unchanged files costs no storage. Nothing
  ever purges old updates: retention policies are an open feature request
  (issue #263, 2026-09-19). Storage only grows, and cleanup is manual. Sources:
  https://mercure-technologies.gitbook.io/xprem/stateless-mode/overview.md,
  https://github.com/mercuretechnologies/xprem/issues/263
- To move off local disk later, R2 plus `CDN_BASE_URL` is the documented path
  (Cloudflare in front of a public R2 bucket). Source:
  https://mercure-technologies.gitbook.io/xprem/cdn/overview.md

## Publishing

- `npx eoas@3.2.5 publish --branch <branch> --nonInteractive` with `EOO_TOKEN`
  (and `RELEASE_CHANNEL`) in the environment. The CLI resolves the runtime
  version, runs `expo export` from the project's own `expo` package, hashes
  every file, uploads only files the server does not already hold, and moves
  the branch to the new update. Identical code is not republished. Source:
  https://mercure-technologies.gitbook.io/xprem/eoas/publish-an-update.md
- EAS CLI is not involved. v3.2.2 removed the last EAS build dependencies
  (PR #242), and the CLI source makes no Expo API calls when `EOO_TOKEN` is set.
  Sources: https://github.com/mercuretechnologies/xprem/releases/tag/v3.2.2,
  `apps/eoas/src` at `f9bd112`
- Traps for CI:
  - Pin `eoas` to the exact server version. A server on 3.2.0+ rejects older
    CLIs.
  - If `EOO_TOKEN` is missing, the CLI silently falls back to Expo auth, and
    the failure shows up as a rejected publish.
  - The CLI refuses to run on a dirty git tree. Commit first, or pass
    `--disableRepositoryCheck`.
  - The CLI does not load `.env` files.
  - Source: https://mercure-technologies.gitbook.io/xprem/eoas/publish-an-update.md
- Runs in GitHub Actions on Linux; the docs show exactly that (`EOO_TOKEN` as
  a CI secret, `--nonInteractive`). Each asset is uploaded as its own request.
  When the server sits behind a proxied Cloudflare record, each upload must stay
  under Cloudflare's 100 MB body limit on Free/Pro plans. JS bundles are far
  below that. Source:
  https://developers.cloudflare.com/support/troubleshooting/http-status-codes/4xx-client-error/error-413/
- **Conflicts with the stack recommendation.** Xprem's docs warn against
  `runtimeVersion: { policy: "fingerprint" }` because "CI and local
  environments can produce different fingerprints for one commit. The only
  symptom is an update that never arrives." They recommend a fixed
  `runtimeVersion` that you bump on every native change. With fingerprint kept,
  the APK build and `eoas publish` must run in the same CI job image; with a
  fixed version, the risk becomes forgetting to bump it. This needs a decision.
  Source: https://mercure-technologies.gitbook.io/xprem/eoas/publish-an-update.md

## Code signing

- In control-plane mode, the server generates an RSA-2048 pair when the app is
  created and seals it in Postgres with the master key. The self-signed
  certificate is valid for 10 years. An admin downloads it from App Info
  (`app-<id>-certificate.txt`), saves it as `certs/certificate.pem`, and
  commits it to the app repo. The private key never leaves the server. Sources:
  https://mercure-technologies.gitbook.io/xprem/key-store/keys.md,
  `internal/crypto/crypto.go:106,218` at `f9bd112`
- `npx eoas init` writes the app config. The `updates` block ends up as:

  ```ts
  updates: {
    url: "https://xprem.nexul.io/manifest",
    enabled: true,
    codeSigningCertificate: "./certs/certificate.pem",
    codeSigningMetadata: { keyid: "main", alg: "rsa-v1_5-sha256" },
    requestHeaders: {
      "expo-channel-name": "production",   // keep a literal, not process.env
      "expo-app-id": "<dashboard app UUID>",
      "xprem-branch": "",                   // enables branch surfing on this build
    },
  },
  ```

  `init` actually writes the two signing fields behind a
  `process.env.DISABLE_CODE_SIGNING` guard so a dev client can run without the
  key. Sources: `apps/eoas/src/commands/init.ts:107-131` at `f9bd112`,
  https://mercure-technologies.gitbook.io/xprem/installation-guide/configure-your-application.md.
  Expo defines `codeSigningCertificate` as the "Local path of a PEM-formatted
  X.509 certificate … When provided, all updates downloaded by expo-updates
  must be signed", and `rsa-v1_5-sha256` is the only allowed `alg`. Source:
  https://docs.expo.dev/versions/latest/config/app/ (2026-09-28)
- Signing keys can only live in the database or in AWS Secrets Manager.
  Nothing documents exporting a key from the database. The sealing code is MIT,
  so decrypting it with the master key is possible but untested. This matters
  only when leaving Xprem.

## Channels, branches, rollouts, rollback

Source for all: https://mercure-technologies.gitbook.io/xprem/eoas/progressive-rollouts.md,
https://mercure-technologies.gitbook.io/xprem/eoas/rollback.md,
https://mercure-technologies.gitbook.io/xprem/eoas/republish.md,
https://mercure-technologies.gitbook.io/xprem/concepts/branches-and-release-channels.md.

- These work the same way as in EAS. A build is bound to a channel (the
  `expo-channel-name` header), updates are published to a branch, and the
  dashboard maps each channel to a branch. Switching channels at runtime with
  `setUpdateRequestHeadersOverride` works, because the header is declared at
  build time.
- Per-update rollout: `eoas publish --rollout-percentage 10`. Devices are
  bucketed deterministically on the `EAS-Client-ID` header. The dashboard can
  raise the percentage, finish the rollout, or revert it. Reverting republishes
  the control update. While a rollout is active, the branch is locked for that
  runtime version.
- Channel rollout: the dashboard splits a channel between two branches, then
  promotes or reverts.
- `eoas rollback --branch <b> --nonInteractive` serves `rollBackToEmbedded`.
  It does not work with `disableAntiBrickingMeasures`. `eoas republish`
  re-points a branch at an earlier update.
- Automatic rollback is a roadmap item for the commercial edition. The
  client-side crash fallback in `expo-updates` still applies.
- Branch surfing lets testers switch a build to another branch through an
  in-app picker (control plane only). Source:
  https://mercure-technologies.gitbook.io/xprem/features/branch-surfing.md

## Dashboard and auth

- Dashboard users sign in with email and password (bcrypt hashes in Postgres).
  Roles are admin, or member (read-only). Only admins can create API keys or
  download the certificate. A demotion takes effect on the next request.
  Source: https://mercure-technologies.gitbook.io/xprem/dashboard/users.md
- Publish tokens are per-app `EOO_TOKEN` keys, stored hashed and shown once.
  Scoping a token to specific branches or an IP allowlist ("Token access") is
  commercial. Sources:
  https://mercure-technologies.gitbook.io/xprem/eoas/overview.md,
  https://mercure-technologies.gitbook.io/xprem/references/open-core-and-licensing.md
- SSO, per-app RBAC, audit log, Observe telemetry and the device registry
  enrichment are all commercial. None of them is needed for OTA. Same source.

## Resource footprint

- Measured locally on 2026-09-28 (v3.2.5 plus `postgres:16-alpine`, local
  storage, dashboard on, no apps, idle): 16.7 MiB for the server and 46 MiB for
  Postgres, both near 0% CPU. `/hc` returned 200 about 15 s after start.
- The project's own load test (2026-08-01, 1 vCPU / 2 GiB Graviton, telemetry
  on) simulates a 1M-MAU fleet: 230 req/s at 1.46 ms mean, and 133 MB process
  memory at peak. Source:
  https://mercure-technologies.gitbook.io/xprem/references/benchmark.md
- Disk grows with every published update (see storage above).

## Maintenance and license

- Releases: v3.2.0 (2026-09-07), v3.2.1 (09-09), v3.2.2 (09-12), v3.2.3
  (09-16), v3.2.4 (09-23), v3.2.5 (09-26). Last push 2026-09-27. Source:
  https://github.com/mercuretechnologies/xprem/releases
- Popularity: 640 stars, 93 forks. Open items: 3 issues (retention policies,
  symbolication, managed builds) and 2 PRs. The first commit is from
  2025-01-29. The README says it "has served OTA updates in production since
  early 2025, to apps totaling more than a million monthly active users" (the
  project's own claim). Sources: GitHub repo metadata and issues list,
  2026-09-28
- Bus factor is 1. Axel Marciano has 507 of the roughly 550 commits overall and
  112 of about 130 in the last six months; no one else has more than 6. Source:
  https://api.github.com/repos/mercuretechnologies/xprem/contributors, git log
  at `f9bd112`
- License: MIT for everything outside `ee/` directories. `ee/` is under a
  commercial license: you can read it and use it for development, but
  production use needs a key. The EE code ships in the same binary and stays
  dormant without a key. The maintainer states that "a feature released under
  the MIT license will never move to the commercial edition". Sources:
  `LICENSE.md`, `ee/LICENSE`, `ee/README.md`, `CONTRIBUTING.md` at `f9bd112`,
  https://mercure-technologies.gitbook.io/xprem/references/open-core-and-licensing.md
- Exit risk is limited. The server speaks the open expo-updates protocol, so a
  replacement can take over behind the same `updates.url`. It must serve the
  same signing key or ship a new build.

## Alternatives, where Xprem falls short

Xprem covers everything the ticket asks for, so the alternatives only matter
for its two gaps: the bus factor and open-core creep.

- **Expo's reference `custom-expo-updates-server`** (MIT, last commit
  2025-09-17). It is a demo: it serves updates from local files, and has no
  channels, auth, dashboard or publish CLI. The README says it is "not
  guaranteed to be complete, stable, or performant enough to use as a
  full-fledged backend", and feature PRs "will likely be closed". It is only
  useful as a reference for writing our own server. Source:
  https://github.com/expo/custom-expo-updates-server
- **hot-updater, self-hosted.** MIT, 1,743 stars, active (pushed 2026-09-28),
  `0.36.15` stable with `1.0.0-rc.16`. Self-hosting means writing a small
  server with `@hot-updater/server` (Hono plus Drizzle/Postgres plus S3/R2
  storage); there is no ready-made image. It replaces `expo-updates` in the
  app. It is the right choice only if per-instance update URLs ever become a
  requirement, which the OTA section argues against. Sources:
  https://github.com/gronxb/hot-updater (`docs/content/docs/custom/hosting/docker.mdx`,
  `docs/content/docs/custom/overview.mdx`), https://registry.npmjs.org/hot-updater

## Recommendation

Run Xprem v3.2.5 in control-plane mode as a Nexul compose stack at
`xprem.nexul.io`: one Xprem container, one Postgres, and a named volume for
local storage. Pin `eoas@3.2.5` in CI. That avoids the Expo account, EAS, and
any paid plan, and bundle signing comes free (EAS limits signing to its
Production plan). Move storage to R2 behind `CDN_BASE_URL` only if download
volume ever matters. Before building the app, settle the fingerprint versus
fixed runtime version question.

## Install checklist

1. **Secrets** (store in Nexul's secret pool, and back up the master key
   offline):
   - `POSTGRES_PASSWORD`
   - `JWT_SECRET` = `openssl rand -base64 32`
   - `DB_KEYS_MASTER_KEY_B64` = `openssl rand -base64 32` (irrecoverable)
   - `ADMIN_EMAIL`, `ADMIN_PASSWORD` (first boot only)
   - Later: `EOO_TOKEN` as a GitHub Actions secret in the app repo.
2. **DNS and proxy:** Cloudflare record `xprem.nexul.io` pointing at the
   Nexul host, with HTTPS terminated by the reverse proxy (Expo clients require
   HTTPS). The proxy routes the whole host to `xprem:3000`. Proxied (orange
   cloud) is fine: manifests are `no-store`, and assets are cacheable for a
   year. To keep the admin UI off the public internet, restrict `/dashboard*`,
   `/api/*`, `/mcp*` and `/oauth/*` at the proxy. Leave these public:
   - `/manifest`, `/assets` and `/branch_lists`, which devices call
   - `/hc`
   - `/{APP_ID}/*`, the `EOO_TOKEN`-authenticated publish routes that CI calls

   Route paths are from `internal/router/routes_client.go` and
   `routes_publish.go` at `f9bd112`.
3. **Compose sketch:**

   ```yaml
   services:
     xprem:
       image: ghcr.io/mercuretechnologies/xprem:v3.2.5
       restart: unless-stopped
       depends_on: [db]
       environment:
         BASE_URL: https://xprem.nexul.io
         DB_URL: postgres://xprem:${POSTGRES_PASSWORD}@db:5432/xprem?sslmode=disable
         DB_KEYS_MASTER_KEY_B64: ${DB_KEYS_MASTER_KEY_B64}
         JWT_SECRET: ${JWT_SECRET}
         STORAGE_MODE: local
         LOCAL_BUCKET_BASE_PATH: /app/updates
         USE_DASHBOARD: "true"
         ADMIN_EMAIL: ${ADMIN_EMAIL}
         ADMIN_PASSWORD: ${ADMIN_PASSWORD}
         DISABLE_TELEMETRY: "true"
       volumes:
         - updates:/app/updates
       expose: ["3000"]          # reverse proxy only; no host port
     db:
       image: postgres:16-alpine
       restart: unless-stopped
       environment:
         POSTGRES_USER: xprem
         POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
         POSTGRES_DB: xprem
       volumes:
         - pgdata:/var/lib/postgresql/data
   volumes:
     updates:
     pgdata:
   ```

4. **First boot:**
   - `curl https://xprem.nexul.io/hc` should return 200.
   - Sign in at `/dashboard` and create the app. Copy its App ID.
   - Download the certificate into the app repo as `certs/certificate.pem`.
   - Create the channel `production` and map it to branch `production`.
   - Create an API token and store it as the `EOO_TOKEN` Actions secret.
   - Optionally, remove `ADMIN_*` from the stack afterwards.
5. **App config:** run `npx eoas@3.2.5 init` once, answering with the App ID,
   `https://xprem.nexul.io`, and yes to the certificate question. Commit the
   `updates` block shown above, with `expo-channel-name` as a literal.
6. **Publish in CI** (GitHub Actions, Linux, after checkout and install):

   ```yaml
   - run: npx eoas@3.2.5 publish --branch production --platform android --nonInteractive
     env:
       EOO_TOKEN: ${{ secrets.EOO_TOKEN }}
       RELEASE_CHANNEL: production
   ```

7. **Backups:** dump Postgres (it holds the sealed keys and the update index)
   and snapshot the `updates` volume. Keep the master key separately; without
   it the dump cannot sign.
8. **Upgrades:** bump the image tag and the `eoas@` version together.
