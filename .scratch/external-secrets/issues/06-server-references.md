# 06 — Connector credentials and the bootstrap secret by reference on the server

**Status:** resolved
**Type:** grilling
**Blocked by:** 01, 02

## Question

Where does the server's own 1Password token live, and how do connector fields (GitHub, Cloudflare,
LiveKit, the GitHub App private key) and the bootstrap auth secret become references? Decide: when the
server resolves (startup, each use, cached in memory), what happens when 1Password is unreachable at
startup when the bootstrap secret, and with it the at-rest key, depends on it, and how `nexul install`
and `upgrade` set this up.

## Answer

Decided 2026-11-12.

- **Bootstrap secret: out of scope.** `NEXUL_AUTH_SECRET` signs sessions, derives the at-rest key and
  keys the GitHub webhook and enrollment tokens. Unset, it is generated into `auth-secret` beside the
  database. Backups (`storage.Backup`) are database-only, so they already lack it, and the server's
  own 1Password token would sit on the same machine, so a reference would add almost no protection.
  It would also stop the server starting while 1Password is down. It stays a file or env value.
- **Server token:** `nexul install --1password-token-file F` or `nexul server 1password` (stdin)
  writes it into the server's 0600 `.env` and restarts the server; `upgrade` keeps it. It is scoped to
  a vault holding only Nexul's own credentials, and checked at start (`ready` or `token rejected`,
  shown in Settings).
- **What can be a reference:** the secret fields of connectors (`CredentialField.Secret`: the
  Cloudflare API token, the LiveKit API secret, the GitHub App private key and OAuth secret) and the
  client secrets of sign-in providers. Non-secret fields such as ids stay literals.
- **When:** resolved at server start and on save (save already checks a connector live), then kept in
  memory. If 1Password is down at start, the server still starts. The affected connector or sign-in
  button shows "1Password unreachable" and retries with backoff, capped at about 100 reads a day.
  Passkeys and existing sessions keep working.
- **Rotation:** when a provider answers 401/403, the server re-resolves that credential's references
  once, at most every 5 minutes, and retries the call. A restart re-resolves everything.
