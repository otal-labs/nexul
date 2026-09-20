# 03 — The URL, the secret, and abuse

**Type:** grilling
**Status:** resolved
**Blocked by:** 01

## Question

The URL is the credential and posting needs no other permission. That makes
it the one unauthenticated write path on the instance, so its lifecycle and
limits are the design. With Discord's contract from ticket 01 in hand:

1. Path shape: mirror `/api/webhooks/{id}/{token}` exactly, so senders that
   validate the URL's shape, or that append `/github`, keep working.
2. Token generation and storage: length, alphabet, and whether it is stored
   hashed (shown once at creation, like scoped tokens) or in the clear (always
   visible to anyone with `botwebhook:read`, like Discord). Discord's choice is
   the convenient one; the scoped-token choice is the repo's precedent.
3. Regenerate: a new token on the same bot, old one dead immediately, the
   way in and the way out.
4. Limits: per-bot rate limit, body size cap, embed caps enforced with
   Discord's numbers or ours, and what a rejected post returns.
5. Audit: whether posts through a URL leave an audit entry, or only creation,
   regeneration, and deletion do.
6. Pre-release item 03 (security review of the integration model): raise it
   with the owner if the answer to any of the above widens the surface past
   "one URL, one conversation, rate limited".

## Answer

Grilled 2026-09-17, three rounds.

- **Path** `/api/botwebhooks/{id}/{token}`, Discord's shape with
  `botwebhooks` in place of `webhooks` so it never reads as the outbound
  signed webhooks integrations use. A sender therefore swaps host and one
  path segment, not the host alone. Mounted before authentication like the
  existing `/hooks/*` routes, and never through the audit middleware, which
  records the raw path and would log the token.
- **Token**: the scoped-token generator, 32 random bytes as 43 URL-safe
  characters. Stored readable, encrypted at rest with the existing crypto
  helper, and the full URL is shown to anyone with `botwebhook:write` at any
  time. Copy-again beats show-once for a credential whose blast radius is
  one conversation.
- **Regenerate** issues a new token on the same bot; the old one dies
  immediately.
- **Limits**: a hand-rolled in-memory bucket, no new dependency. 30 posts
  per minute per bot, 60 rejected requests per minute per IP, 64 KiB JSON
  body cap. Multipart stays in the fog and takes the attachments cap when it
  arrives.
- **Rejections** mirror Discord: 404 with code 10015 "Unknown webhook" for a
  wrong id or token (no oracle between the two), 429 with `retry_after` and
  the `X-RateLimit-*` headers, and 400 with Nexul's normal error body for a
  bad payload.
- **Posting is synchronous**: message row, outbox event, and audit row in
  one transaction; the sender gets 204, or the message with `wait=true`, or
  the error directly. The event bus fans out afterwards.
- **Audit**: management actions are recorded by the existing `/api`
  middleware for free. Each accepted post writes one audit row inline, actor
  type `bot`, actor id the bot, action with the token stripped. Rejects are
  never audited, only logged and counted by the limiter; the sender already
  gets the reason on the API, and a retry loop or token guessing would
  otherwise fill the table. The bot row keeps last-post time and a post
  count for the Bots tab.
- **The bot dies with its conversation** through an on-delete cascade on
  its conversation foreign key, the same way a thread's messages already go
  when its ticket is deleted. Channel deletion itself does not exist today
  and belongs to chat, not this effort.
- **Security review**: four lines added to pre-release item 03's checklist
  (token entropy, rate limit, no oracle on a wrong token, revoke and
  regenerate cut access immediately).
