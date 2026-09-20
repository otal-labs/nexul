# 08 — How do existing allowlists and pending invites migrate?

**Type:** grilling
**Status:** open
**Blocked by:** 01, 02, 03

## Question

Replace the current instance allowlist and login-keyed workspace invite flow
without unexpectedly locking out an existing installation:

- How current registered users become durably admitted users under the new
  account model.
- What happens to allowlist entries whose identities have never signed in.
- What happens to pending workspace invites keyed by GitHub username or
  verified email, including ones spanning several workspaces.
- Whether the old allowlist and invite APIs receive a compatibility period or
  disappear in one release.
- How the separate Allowlist settings and workspace invite UI are removed or
  redirected to invitation-link management.
- Which migrations are reversible, what rollback would mean after a link has
  been redeemed, and which beta installations need preservation.
- Which parts of ADR 0024 and ADR 0040 the new design supersedes, and which
  distinctions remain valid.

The result must preserve the private-instance boundary throughout upgrade and
must not make an old identifier-based invite redeemable by the wrong OAuth
identity.
