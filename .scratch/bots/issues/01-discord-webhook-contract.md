# 01 — What exactly is Discord's execute-webhook contract?

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

The URL must accept what Discord's webhook URL accepts, so that any sender
already pointed at a Discord webhook works against Nexul by swapping the
URL. Establish from Discord's own developer documentation, not a write-up
of it:

- The URL shape (`/api/webhooks/{id}/{token}`), the token's length and
  alphabet, and whether the id is meaningful to senders.
- Every request field of Execute Webhook: `content`, `username`,
  `avatar_url`, `tts`, `embeds`, `allowed_mentions`, `components`, `files`,
  `payload_json`, `attachments`, `flags`, `thread_name`, and any others,
  with which are required and their limits (content length, embed count,
  per-embed field limits, the total embed character cap).
- The full embed object: title, description, url, timestamp, color, footer,
  image, thumbnail, author, fields, and the limits on each.
- The query parameters `wait` and `thread_id`, and what the response is
  with and without `wait` (status code, body).
- Error responses and codes a sender may branch on, and the rate limit that
  applies per webhook.
- The multipart form shape for file uploads.
- The message endpoints on the same URL: get, edit (PATCH), and delete a
  webhook message.
- The compatibility suffixes Discord accepts on a webhook URL (`/github`,
  `/slack`) and what each translates.
- Three real payloads senders emit today, quoted verbatim where the docs
  show them: GitHub's Discord notification through the `/github` suffix,
  Grafana's Discord contact point, and Uptime Kuma's Discord notification.

Findings go to `.scratch/bots/research/01-discord-webhook-contract.md`,
each claim with its source URL.

## Answer

Contract pinned from Discord's own docs. The URL is
`POST /webhooks/{id}/{token}`: the id identifies the webhook, the token is an
opaque secret with no documented length or alphabet. A post needs at least
one of `content`, `embeds`, `components`, `file`, or `poll`; `content` caps
at 2000 characters, ten embeds, 25 fields per embed, 6000 characters across
all embed text. `wait=false` (the default) answers `204`, `wait=true` returns
the message; the `/slack` and `/github` routes default `wait` to true. Files
go as `multipart/form-data` with the JSON in `payload_json` and `files[n]`,
referenced from embeds as `attachment://name`. Rate limits are per webhook
with the standard `X-RateLimit-*` headers and a 429 body, but no fixed quota
is published. Edit and delete of a webhook's own messages live on the same
URL under `/messages/{id}`. GitHub, Grafana, and Uptime Kuma payloads are
quoted verbatim.

Unverified from a primary source: the token's exact format, a numeric
per-webhook quota, the status code on `wait=true`, and the transformation
rules behind `/github` and `/slack`.

Research: `.scratch/bots/research/01-discord-webhook-contract.md`
