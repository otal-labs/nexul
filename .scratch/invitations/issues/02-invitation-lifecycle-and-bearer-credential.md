# 02 — What is the invitation lifecycle and bearer-credential contract?

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

Define the invitation aggregate around the settled single-use UUIDv4 bearer
credential, one-day or seven-day expiry, token hashing, and atomic consumption:

- Which active and terminal states need persistence, and when the active row is
  deleted rather than retained as revoked, expired, redeemed, or invalidated.
- What non-secret audit record remains after redemption, manual revocation,
  expiry, or automatic deletion.
- Every condition that deletes an unredeemed invitation. The creator losing
  `members:write` in any granted workspace is already one such condition;
  decide the behavior for deleted workspaces, deleted roles, changed role
  permissions, removed permission-overwrite values, and creator-account
  removal.
- How expiration is enforced and expired rows are cleaned up without a
  scheduler becoming a correctness requirement.
- The transaction boundary that guarantees one winner under concurrent
  redemption.
- What the public preview and redemption endpoints reveal for a random,
  expired, revoked, already-used, or malformed token.
- Rate limits, audit fields, structured logs, and redaction rules for the
  public credential endpoints.

The raw token may appear only in the generated link returned at creation. It
must not be recoverable from storage, logs, list responses, or MCP output
afterward.
