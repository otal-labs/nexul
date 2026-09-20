# Wayfinder map: private invitation links

Charted 2026-09-20 after three grilling rounds with the owner. An
**invitation** is a single-use bearer link that admits one person to a private
Nexul instance and grants a preselected access package across one or more
workspaces. The inviter does not need to know the person's OAuth identity.

## Destination

A decision-complete spec at `.scratch/invitations/spec.md`, ready to slice
into implementation tickets: private instance admission without a manual
allowlist, multi-workspace grants, the bearer-link lifecycle, OAuth
registration and redemption, account removal, the browser and MCP flows, and
the HTTP, event, and permission contracts. Planning only; implementation
starts after the spec is approved.

## Notes

- Grilling tickets invoke `/grilling` and `/domain-modeling`. Prototype
  tickets invoke `/prototype` and `design-mode`; they use real Nexul data and
  verify the chosen direction at 320, 375, 414, and 768px before resolution.
- Security is part of every ticket. The invitation token is a credential, and
  tickets touching it invoke `security-and-hardening` before deciding.
- Grounding: `CONTEXT.md` (Workspace, Instance, Permission, Permission
  overwrite, Allowlist, Auth, Access), ADR 0024 (sign-in, instance admin, and
  workspace membership are separate), ADR 0040 (provider OAuth only), ADR
  0041 (stateless sessions), and ADR 0042 (permission overwrite precedence and
  Owner bypass). Code: `internal/auth/`, `internal/tenancy/`,
  `internal/access/`, `internal/roles/`, and their adapters under
  `internal/platform/storage/`, `server/cmd/`, and `web/src/`.
- The resulting spec must say which existing ADRs it supersedes. `CONTEXT.md`
  stays truthful to the shipped product until implementation changes it.
- The auth pre-release item at
  `.scratch/pre-release/issues/02-update-github-app-urls.md` must be surfaced
  before testing an OAuth callback on a non-localhost domain.

### Settled at charting

- The inviter controls the invitation. The redeemer may accept or decline and
  choose one of the OAuth providers configured by the instance owner; they
  cannot alter any grant.
- Authentication stays provider OAuth only. Nexul does not add passwords,
  magic links, or an email sender.
- One invitation may carry grants for several workspaces. Every grant selects
  a non-Owner role and may add workspace-wide permission overwrites. It cannot
  grant resource-specific access or the instance-level ability to create a
  workspace.
- The invitation is unbound to an email address, provider, or username.
  Possession of the link is authority to redeem it once.
- The credential is a cryptographically random UUIDv4. Nexul stores only its
  hash, returns the complete link once at creation, and never logs or lists the
  raw token.
- The inviter chooses a one-day or seven-day lifetime; seven days is the
  default. Invitations cannot be permanent or use a custom lifetime.
- Redemption is atomic across all grants. If any grant is invalid, none are
  applied. Concurrent redemption lets one request succeed and gives every
  other request the same generic invalid-or-expired result.
- Existing workspace memberships are left unchanged. Redemption adds only
  missing memberships and still consumes the link.
- Creating and revoking an invitation requires `members:write` in every
  included workspace. If the creator later loses that permission in any of
  them, Nexul deletes the unredeemed invitation rather than leaving it to fail
  at redemption.
- The link opens an acceptance screen showing the instance, workspaces, roles,
  and expiry. Detailed permission overwrites stay private until the person has
  authenticated.
- Nexul generates a copyable link and does not deliver it. Browser, HTTP, and
  MCP invitation management share the same use-case and permission checks.
- Invitation redemption replaces the manual allowlist as the normal first
  admission path. A registered user may keep signing in without another
  invitation. An unknown OAuth identity needs a valid invitation.
- A registered user remains admitted after losing their last workspace
  membership until an instance administrator disables or removes the account.

## Decisions so far

<!-- one line per resolved ticket: gist, then the link for the detail -->

## Not yet specified

- The exact account states and session effect of disabling or removing a
  registered user. This becomes sharp after the replacement for the allowlist
  is decided.
- Which invitation actions need durable audit history after the active
  credential row is deleted, and which catalog events should represent them.
  This depends on the invitation lifecycle.
- Limits on active invitations per creator or instance, cleanup of expired
  rows, and rate limits on preview and redemption. These depend on the bearer
  credential threat model.
- The post-redemption landing page and recovery paths for interrupted OAuth,
  expired links, and already-registered users. These graduate after the OAuth
  contract and acceptance prototype settle.
- Whether membership changes need live WebSocket updates on the Members page.
  This depends on the management flow.

## Out of scope

- Passwords, email-and-password registration, magic links, and new OAuth
  providers.
- Outbound email, direct messages, or any other delivery service for the
  generated link.
- Reusable, permanent, identity-bound, or custom-duration invitations.
- Inviting somebody as a workspace Owner or granting instance-level workspace
  creation.
- Per-document, per-play, or other resource-specific grants in the invitation
  dialog.
- Public registration, join requests, workspace discovery, and open invite
  directories.
- Implementing the feature while this map is active. The destination is the
  approved specification.
