# 06 — Register the Cloudflare OAuth app

**Status:** wontfix for now

**Blocked by:** None — needed before the DNS domain's credential flow can work.

**What to build:** The DNS domain's credential flow uses Cloudflare OAuth, which
needs a registered Cloudflare OAuth application (client ID + secret) — the same
pattern as the GitHub OAuth App.

Owner chose OAuth over pasted API tokens during the DNS domain discussion
(2026-08-11), then settled on the pasted API token with its ticker dialog
(2026-09-27): first run sets up the domain before anything else, and an OAuth
callback cannot exist before the domain does. OAuth is an optional later
addition, never a first-run dependency.

- [ ] Confirm which OAuth scopes Cloudflare offers for zone DNS editing —
      do this **before** implementation, since it may constrain the design
- [ ] Cloudflare OAuth application registered
- [ ] Client ID + secret supplied via env / secrets manager, never committed
- [ ] Callback URL registered and matching config

## Surface when

- Someone asks to connect Cloudflare by signing in instead of pasting a token.
