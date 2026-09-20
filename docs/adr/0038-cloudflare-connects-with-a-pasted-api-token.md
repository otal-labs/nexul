# Cloudflare connects with a pasted API token, which is why connectors have a manual credential kind

Every other connector registers an OAuth app whose client ID and secret the
owner sets once, but Cloudflare only issues OAuth clients to partners — a
self-hoster can never fill that form, so the OAuth-only connector model would
have left the product's own DNS and tunnel features unreachable. A connector
therefore carries an optional `Manual` list of credential fields alongside its
OAuth client, stored as one encrypted blob in the same credentials row, and
Cloudflare declares an `api_token` field. Cloudflare's OAuth client stays
wired for an instance that does hold a partner app; the card just prefers the
manual form.

Because a pasted token has no consent screen to fail on, it is verified
against the live API before it is stored — one request per permission the
product actually uses (token validity, zone read, DNS edit, tunnel edit), each
surfaced as its own ticked or crossed row, so a 403 names the missing
permission instead of failing later inside a deploy.

Decided: 2026-09-03
