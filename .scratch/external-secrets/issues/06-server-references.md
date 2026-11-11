# 06 — Connector credentials and the bootstrap secret by reference on the server

**Status:** open
**Type:** grilling
**Blocked by:** 01, 02

## Question

Where does the server's own 1Password token live, and how do connector fields (GitHub, Cloudflare,
LiveKit, the GitHub App private key) and the bootstrap auth secret become references? Decide: when the
server resolves (startup, each use, cached in memory), what happens when 1Password is unreachable at
startup when the bootstrap secret, and with it the at-rest key, depends on it, and how `nexul install`
and `upgrade` set this up.
