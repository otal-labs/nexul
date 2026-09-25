# The MCP server speaks the 2026-07-28 and 2025-03-26 protocol revisions side by side

Superseded by ADR 0067: the official Go SDK now serves both eras, and the
hand-rolled protocol code below is gone.

Which revision a request gets is decided per request, from
`_meta["io.modelcontextprotocol/protocolVersion"]`; no `_meta` means the
legacy 2025-03-26 behaviour, and an unknown version is rejected with -32022.
Adopting the newer revision cost nothing structurally — the server was
stateless before the stateless revision existed, with no session id, no
session store, and no held-open streams — so nothing had to be removed, only
added.

The legacy revision stays until its deprecation window closes because the
clients people actually connect still speak it, and dropping it would make the
adapter correct and unusable at the same time.

Note for anyone editing protocol-level code: the 2026-07-28 specification
postdates common model training data, so read
https://modelcontextprotocol.io/specification/2026-07-28 rather than trusting
recalled field names.
