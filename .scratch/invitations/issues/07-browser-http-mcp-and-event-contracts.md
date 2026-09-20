# 07 — What are the browser, HTTP, MCP, and event contracts?

**Type:** grilling
**Status:** open
**Blocked by:** 01, 02, 03, 04

## Question

Design the public interfaces around one invitation use-case layer after the
admission, credential, access-package, and OAuth decisions are settled:

- Browser HTTP operations for preview, create, list, revoke, accept, redeem,
  account administration, and any recovery action the OAuth flow needs.
- MCP tools for listing, creating, and revoking invitations, with the raw link
  returned only by creation and never by list.
- Authorization for multi-workspace create, list, and revoke calls, including
  the behavior when the actor can manage only some grants in a bundle.
- Stable request, response, and error shapes that do not expose token hashes
  or distinguish invalid bearer credentials.
- Catalog events and outbox writes for invitation creation, redemption,
  revocation, automatic deletion, account admission, disablement, and removal.
- Whether invitation and membership changes need live WebSocket updates.
- Which operations belong to `auth`, `tenancy`, `access`, or a new domain, and
  the consumer-side interfaces that keep those domains independent.

The browser and MCP server must be peers over the same behavior, not separate
implementations with different permission rules.
