# Private instances admit users through single-use invitation links

This supersedes the allowlist decision in ADR 0040 and the first, two-step
admission rule in ADR 0024. Authentication remains provider OAuth, instance
administration remains separate from Workspace Roles, and a Role still lives
per membership.

An unknown OAuth identity may register only through a valid Invitation. The
Invitation is an unbound, single-use bearer link containing an access package
for one or more Workspaces: one non-Owner Role and optional workspace-wide
Permission overwrites per Workspace. The inviter needs `members:write` in
every included Workspace. Existing active Users continue to sign in without
another Invitation.

The raw UUIDv4 credential appears only in the generated URL and is stored as a
SHA-256 hash. It expires after one or seven days. OAuth authenticates the
recipient but does not admit them. The recipient reviews the complete package
and explicitly accepts it before one SQLite transaction creates the User when
needed, adds missing memberships, applies overwrites, consumes the Invitation,
and writes its outbox events. Existing memberships and overwrites are never
changed by redemption.

Registered Users have an active, disabled, or removed Account status. Auth
middleware reloads the User on each session or personal-token request, so
disabling or removing an Account takes effect without a session store.
Removal keeps authored content attributed while removing access. The final
active instance administrator cannot be disabled or removed.

This replaces the need to discover a recipient's GitHub username or verified
email before inviting them. The cost is that an Invitation URL is itself a
short-lived credential: whoever receives it can claim the specified access,
so it is shown once, revocable, and never sent by Nexul.
