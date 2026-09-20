# 10 — Non-OAuth credentials on the connectors surface

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

LiveKit's credential is URL + API key + secret — no OAuth flow. The
connectors framework (CN2) is OAuth-shaped: `OAuth == nil` currently means
"coming soon," and the whole card flow assumes authorize/exchange/refresh.

Decide how a static-credential connector fits:

- Extend `Connector` with a second credential kind (a form-based
  `ManualCredential` spec: field names, which are secret) so LiveKit gets
  a real Settings card with Connect = "fill in these fields" — or is a
  bespoke settings section honest enough for one connector?
- Storage: same `connector_credentials` table (encrypted, never
  serialized) or the `connector_app_config` shape (CN3a) — LiveKit's
  credential is instance-wide owner config, which smells like app-config,
  not per-user connection. Pick and justify.
- "Connected" status for a static credential: verify with a live API call
  at save time (mirroring CN5's `/user/installations` check) so the green
  pill means something.
- Does this generalize (future non-OAuth tools) or stay minimal? Standing
  rule applies: DB-encrypted, runtime-configurable, no env vars.

Close by extending `connectorsDomains.md` (CN2/CN3) with the decided
shape.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review.

- **Generalize, minimally:** `Connector` gains
  `Manual []CredentialField{Key, Label, Secret bool}`. Non-nil Manual →
  the card's Connect renders a form instead of an OAuth redirect. A
  connector may have both (Cloudflare eventually: OAuth or manual API
  token).
- **Storage: `connector_credentials`, same table.** CN1 already makes
  everything instance-scoped singletons, so "per-connection" vs
  "app-config" collapses for manual connectors — the filled fields are
  the connection. Fields stored as one encrypted JSON blob; never
  serialized, logged, or evented; `CredentialStatus` unchanged.
- **Verify at save:** each manual connector registers a `Verifier` that
  makes one live API call before the credential is marked configured
  (LiveKit: ListRooms) — mirroring CN5's `/user/installations` check, so
  the green pill means something. Failure surfaces inline, nothing
  stored.
- **LiveKit registry entry:** id `livekit`, fields: WebSocket URL
  (`ws_url`), API key (`api_key`), API secret (`api_secret`, Secret).
  Category "communication".
- Standing rule holds: DB-encrypted, runtime-configurable, no env vars.
