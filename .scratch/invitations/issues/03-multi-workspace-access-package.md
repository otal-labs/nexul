# 03 — How does a multi-workspace access package behave?

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

One invitation may carry several workspace grants. Each grant chooses a
non-Owner role and optional workspace-wide permission overwrites. Settle the
validation and redemption semantics:

- How the creator proves `members:write` in every workspace at creation, and
  whether every invitation manager who can list or revoke the bundle must hold
  it in all included workspaces.
- Whether a role and its permissions are snapshotted at creation or resolved
  from the current role at redemption.
- How optional allow and deny overwrites are represented and checked against
  the permission catalog.
- What happens when a selected role, workspace, or permission disappears or
  changes before redemption.
- How existing memberships are detected and preserved while missing
  memberships are added in the same all-or-nothing transaction.
- Whether invitation redemption creates any membership when every selected
  workspace already contains the redeemer.
- What the redeemer can see about each grant before authentication and after
  authentication.

The answer must preserve the protected Owner role, Owner bypass, and the rule
that the invitation cannot grant `can_create_workspace` or resource-specific
access.
