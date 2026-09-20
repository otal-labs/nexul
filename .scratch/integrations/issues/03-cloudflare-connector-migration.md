# 03 — Cloudflare onto the connectors surface

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Cloudflare's Settings card is a disabled "Coming soon" stub and its OAuth
app config is env-var-based — the last env-var holdout, and
`connectorsDomains.md` Boundaries deliberately deferred migrating it
("risk for zero new capability"). This effort un-defers it (standing rule:
DB-encrypted config, no env vars).

Decide the migration shape:

- Cloudflare's registry entry gets a real `OAuthClient` — reuse the
  existing `internal/dns/cloudflare/oauth.go` implementation behind
  connectors' promoted interface, or re-implement thin?
- App config (client ID/secret) moves to `connector_app_config` (CN3a);
  the env-var token fallback (DN2's headless path) — keep, or does a
  DB-stored API token replace it?
- How does `internal/dns` consume connectors-held credentials without
  importing `internal/connectors` — composition-root adapter in
  `server/cmd/main.go`, mirroring the gitprovider bridge (CN7)?
- What happens to instances already connected via the old env-var path —
  any migration/backfill needed? (No production data yet, so breaking is
  allowed — confirm and keep it simple.)

Close by updating `connectorsDomains.md` (Boundaries paragraph) and
`dnsDomains.md` DN2.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review.

- **Reuse, don't re-implement:** Cloudflare's registry entry gets a live
  `OAuthClient` backed by the existing `internal/dns/cloudflare/oauth.go`
  logic, adapted to read client ID/secret from `connector_app_config`
  live (same pattern as `connectors/github`'s AppConfig reads). Wired at
  the composition root like github's.
- **Credential storage moves to connectors:** dns's own token rows retire;
  the Cloudflare token lives in `connector_credentials` under id
  `cloudflare`. `internal/dns` consumes it through a token-provider seam
  (adapter in `server/cmd/main.go` → `connectorsSvc`'s lazy-refresh
  accessor) — dns never imports connectors.
- **Env vars die** (standing rule): `NEXUL_CLOUDFLARE_*` app-config
  and token env vars are removed from config, compose files, and
  `.env.example`. The headless/manual path survives as a **manual
  credential** on the cloudflare connector (API token + account ID fields,
  ticket 10's mechanism) — DB-encrypted, runtime-configurable.
- **No backfill:** no production data yet (confirmed standing project
  fact); existing env-var setups reconnect via Settings once.
- **Frontend:** the DNS settings section's Cloudflare credential UI folds
  into the connectors card; zone/instance-record wizard parts stay in DNS
  settings.
