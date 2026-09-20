# 04 — How does OAuth carry and redeem an invitation safely?

**Type:** grilling
**Status:** open
**Blocked by:** 01, 02, 03

## Question

Define the complete path for a signed-out or signed-in person who opens an
invitation, reviews it, authenticates through an owner-configured provider,
and redeems every workspace grant atomically:

- How the bearer credential survives the OAuth redirect without entering a
  provider callback URL, session token, log line, or unsafe client storage.
- Which step performs the explicit acceptance and when the invitation is
  consumed.
- How CSRF protection binds the browser that started OAuth to the callback
  without binding the invitation to a pre-known identity.
- How a newly created User, durable instance admission, workspace memberships,
  and invitation consumption commit without exposing a partially registered
  account or partially applied access package.
- How an already authenticated user redeems the same link without another
  OAuth round trip.
- What happens after provider denial, callback failure, a disabled account,
  an invitation expiring mid-flow, repeated callbacks, or two browser tabs
  racing the same credential.
- Where successful redemption sends the user when one or several workspaces
  were granted.

Unknown OAuth identities without a valid invitation remain rejected, and
failed authentication or acceptance must not consume the link.
