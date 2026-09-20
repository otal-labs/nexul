# 06 — What should invitation acceptance and failure feel like?

**Type:** prototype
**Status:** open
**Blocked by:** 01, 02, 03, 04

## Question

Build a throwaway prototype of the recipient-facing journey and use it to
choose the final interaction model:

- The public preview showing the instance, workspaces, roles, and expiry while
  withholding detailed permission overwrites until authentication.
- Provider selection when the owner enabled GitHub, Google, Discord, or any
  subset of them.
- The explicit acceptance action for signed-out and already signed-in users.
- The authenticated review of detailed grants before final redemption.
- Generic invalid-or-expired treatment for malformed, revoked, expired, used,
  or concurrently consumed links without leaking which case occurred.
- Recovery after OAuth denial, callback failure, an invitation expiring during
  authentication, or a disabled account.
- The confirmation and landing choice after one or several workspace
  memberships are added, including the case where all memberships already
  existed.

Use the same link and data shape chosen by the lifecycle, access-package, and
OAuth tickets. Resolve the product behavior, not production components.
