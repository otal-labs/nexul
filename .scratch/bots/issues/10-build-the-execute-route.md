# 10 — The execute route: posting through the URL

**What to build:** `POST /api/botwebhooks/{id}/{token}` per the spec's "The
URL, the token, and limits" and "The payload" sections, mounted before
authentication like `/hooks/*` and kept out of the audit middleware. Parse
and validate Discord's execute-webhook JSON (`content` up to 2000
characters, `username`, `avatar_url`, up to ten `embeds` within Discord's
per-field and 6000-character limits, `allowed_mentions`; at least one of
`content` or `embeds`; 64 KiB body). Compare the token in constant time
against the decrypted one. An in-memory limiter: 30 posts a minute per bot,
60 rejected requests a minute per IP. Answers mirror Discord: 204, the
message with `wait=true`, 404 code 10015 for a wrong id or token alike, 429
with `retry_after` and `X-RateLimit-*`, 400 with Nexul's error body. One
transaction writes the message (author kind `bot`, the snapshot of the
post's or the bot's name and avatar, the embeds), its
`chat.message.created` outbox event, one audit row (actor type `bot`,
token stripped), and the bot's last-post time and count. `@user` mentions
are filtered by `allowed_mentions`; `@Agent` never fires.

**Blocked by:** 09

**Status:** resolved

- [x] End to end over real SQLite with the GitHub Actions, Grafana, and
      Uptime Kuma payloads from `research/01-discord-webhook-contract.md`
- [x] Rejections: the same 404 body for a wrong token and a wrong id, a
      deleted bot, and a regenerated token; 429 from both limits; 400 for
      an empty post and an oversize body
- [x] The token never reaches a log line, an audit row, or an event
- [x] A disallowed mention is stored as plain text; the agent pipeline
      skips the message
- [x] The four bot lines on `.scratch/pre-release/issues/03-security-review-integration-model.md` can be ticked
