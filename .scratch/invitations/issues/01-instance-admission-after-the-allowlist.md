# 01 — What replaces the allowlist for instance admission?

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

Invitation redemption becomes the normal way an unknown OAuth identity gains
entry to a private instance, while registered users keep signing in without a
new invitation. Define the durable admission model that replaces the current
login/email allowlist:

- Which account states exist after registration, including active, disabled,
  and removed, and who may move a user between them.
- Whether disabling access invalidates existing stateless sessions immediately
  or only blocks the next sign-in.
- What removing an account does to its workspace memberships, authored
  content, personal access tokens, paired harnesses, and outstanding
  invitations.
- How the first-user bootstrap remains possible without opening public
  registration.
- What an admitted user with no workspace memberships sees after sign-in.
- Which instance-level permission governs account administration. The current
  model uses `can_create_workspace` for allowlist management; decide whether
  that remains the right boundary.
- Whether an administrator may admit someone without a workspace invitation,
  or whether every new non-bootstrap account must arrive through a link.

Keep instance admission, workspace membership, and workspace roles distinct
in the domain model even when redemption performs them in one flow.
