# Private invitation links

**Status:** ready-for-agent

## Objective

Replace identifier-based admission and workspace invitations with a
single-use bearer link. An inviter selects one or more workspaces, one
non-Owner role per workspace, optional workspace-wide permission overwrites,
and a one-day or seven-day lifetime. The recipient opens the link, signs in
through an OAuth provider configured by the instance owner, reviews the
grants, and joins every workspace atomically.

The inviter never needs the recipient's GitHub username or verified email.
Unknown OAuth identities cannot register without a valid invitation. Existing
active users continue signing in normally and may redeem an invitation while
already authenticated.

## Product behavior

### Creating and managing invitations

- `members:write` is required in every selected workspace to create, inspect,
  or revoke an invitation bundle. Instance administration is not a bypass.
- A bundle contains one grant per workspace. Each grant has one non-Owner
  role and optional workspace-wide allow and deny overwrites from the shared
  permission catalog.
- Owner roles, `can_create_workspace`, resource-specific access, duplicate
  workspaces, unknown actions, empty bundles, and conflicting allow/deny
  values are rejected.
- The inviter chooses one day or seven days. Seven days is the default. There
  is no custom or permanent duration.
- Creation returns the complete link once. Lists expose the invitation id,
  creator, creation and expiry times, and grant summaries, but never the raw
  token or its hash.
- Revocation deletes the active invitation. Expired invitations and
  invitations whose creator no longer has `members:write` in every grant are
  deleted when encountered by list, preview, or redemption.
- The Members page keeps its existing roster and role-management behavior.
  Its identifier-based invitation queue is replaced by an invitation action,
  an active-invitations feed, and one-time link reveal.

### Invitation link and preview

- The raw credential is a cryptographically random UUIDv4. Storage keeps only
  its SHA-256 hash. The invitation has a separate UUIDv7 id for references and
  events.
- The generated URL is `<instance-url>/invite#<raw-token>`. The fragment keeps
  the credential out of HTTP request paths, reverse-proxy logs, referrers, and
  OAuth provider URLs.
- The public `/invite` page posts the fragment token once to the public preview
  endpoint. The server validates it, stores it in a short-lived HttpOnly,
  Secure when HTTPS, SameSite=Lax cookie, and returns non-secret preview data.
  The browser immediately removes the fragment from the address bar.
- Public preview shows the configured instance host, workspace names, role
  names, expiry, and enabled OAuth providers. It does not expose permission
  overwrites.
- Malformed, expired, revoked, already-used, incomplete, and race-lost
  invitations all produce the same invalid-or-expired result.

### OAuth and redemption

- Authentication remains OAuth-only through the owner-configured GitHub,
  Google, and Discord providers. This feature adds no password, magic-link,
  or outbound-email system.
- OAuth state and the invitation cookie are independent. The raw invitation
  token never enters OAuth `state`, provider URLs, callback URLs, session
  claims, logs, local storage, or query strings.
- A new provider identity is inserted only while a valid invitation is being
  redeemed, in the same SQLite transaction as instance admission, all missing
  memberships, new workspace overwrites, invitation consumption, and outbox
  events. A failed redemption leaves no user or partial membership behind.
- An existing active user can redeem after OAuth or from an authenticated
  invite page. Existing memberships, roles, and overwrites are preserved;
  only missing memberships receive the invitation's role and overwrites. The
  link is consumed even if every membership already existed.
- All grants validate before any write. One missing workspace, deleted role,
  Owner role, invalid permission, or unauthorized creator deletes the active
  invitation and applies nothing.
- The shared SQLite write serializer and one conditional consume enforce one
  successful redeemer. Concurrent losers receive the generic invalid result.
- Failed OAuth, explicit decline, and closing the page do not consume the
  invitation. Successful redemption clears the invitation cookie, refreshes
  the workspace list, selects the first granted workspace, and lands in the
  normal application.

### Instance admission and account administration

- `users` is the durable instance-admission record. Accounts have `active`,
  `disabled`, or `removed` status.
- Existing user rows migrate as active. Existing allowlist rows and old
  login-keyed pending workspace invites do not become bearer invitations and
  are removed by the migration.
- A known active provider identity signs in without an invitation. Disabled
  and removed identities cannot sign in or use an existing session or PAT.
  Auth middleware already reloads the user row on every request, so status
  changes invalidate stateless sessions immediately without a session table.
- The first identity on an empty instance remains the bootstrap exception. It
  is created atomically only when no user row exists, then completes the
  existing owner wizard. The check is based on user count, not the later
  `can_create_workspace` grant.
- Instance administrators continue to be users with
  `can_create_workspace`. They can list accounts, disable or reactivate an
  account, remove an account, and restore a removed account.
- Disabling preserves memberships and credentials but blocks their use.
  Reactivating restores use.
- Removing is a reversible tombstone. It removes workspace memberships and
  permission overwrites, revokes PATs and pairings, clears
  `can_create_workspace`, and deletes invitations created by the account.
  Authored content and the provider identity remain attached to the tombstone.
  Restoring permits sign-in again but does not restore removed memberships or
  credentials.
- The last active instance administrator cannot be disabled or removed.
- An active account with no workspace membership may sign in and sees an empty
  workspace state.

## Domain and storage design

Invitation ownership remains in `internal/tenancy`, which already owns
workspace membership and the current pending-invite behavior. Auth continues
to own provider identity, sessions, and account status. The composition root
adapts narrow consumer-side interfaces between them; neither domain imports
the other.

The migration adds account status and replaces `allowlist` and
`workspace_invites` with:

```sql
CREATE TABLE invitations (
    id          TEXT PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    invited_by  TEXT NOT NULL REFERENCES users(id),
    created_at  INTEGER NOT NULL,
    expires_at  INTEGER NOT NULL
);

CREATE TABLE invitation_grants (
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    workspace_id  TEXT NOT NULL REFERENCES workspaces(id),
    role_id       TEXT NOT NULL,
    allow_json    TEXT NOT NULL DEFAULT '[]',
    deny_json     TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (invitation_id, workspace_id)
);
```

`role_id` deliberately has no foreign key. Deleting a role must invalidate
and delete the complete invitation rather than block role deletion or silently
drop one grant. Redemption resolves the current role, so role permission
changes before redemption take effect normally.

A specialized storage repository owns the aggregate transaction. It uses
`sqlcgen.New(tx)` inside the shared serializer to validate, register a new
identity when needed, add missing memberships and overwrites, consume the
invitation, and enqueue events. It does not nest calls to other repository
methods that start their own transactions.

No scheduler is required for correctness. Reads and mutations prune expired
or invalid invitations before returning.

## HTTP contracts

Public routes:

- `POST /api/invitations/preview` with `{ "token": "..." }` validates the
  token, sets the HttpOnly invitation cookie, and returns public preview data.
- `GET /api/invitations/preview` reads the invitation cookie after an OAuth
  redirect and returns the same preview.

Authenticated routes:

- `POST /api/invitations` creates a bundle and returns metadata plus `url`.
- `GET /api/invitations` lists bundles the actor may manage without a raw
  token or hash.
- `DELETE /api/invitations/{id}` revokes a bundle.
- `POST /api/invitations/redeem` redeems the cookie-held credential for the
  authenticated account.
- `GET /api/auth/accounts` lists registered accounts for an instance admin.
- `PATCH /api/auth/accounts/{id}` changes status between active and disabled
  or restores removed to active.
- `DELETE /api/auth/accounts/{id}` tombstones the account.

The OAuth start and callback handlers read the invitation cookie when present.
The callback admits or redeems before minting a session and redirects back to
`/invite` so the page can show success. All token failures use one stable
machine code and generic message.

## MCP and events

Tenancy exposes `create_invitation`, `list_invitations`, and
`revoke_invitation`. They call the same service methods as HTTP. Creation is
the only MCP result containing the complete link.

The catalog gains additive topics:

- `invitation.created`
- `invitation.revoked`
- `invitation.redeemed`
- `invitation.deleted`
- `account.admitted`
- `account.disabled`
- `account.reactivated`
- `account.removed`
- `account.restored`
- `workspace.member.added`

Mutation repositories write their events to the outbox in the same
transaction. Event payloads may contain invitation ids, actor ids, account
ids, workspace ids, role ids, and a deletion reason. They never contain a raw
token, token hash, OAuth credential, or session credential.

The web client invalidates its query caches after its own mutations. No new
live WebSocket behavior is required for this slice.

## Frontend structure

- `/members` remains the management page. The selected workspace is the first
  grant by default, and the inviter may add other workspaces.
- The create dialog uses React Hook Form and Zod, one named grant-row component
  per workspace, existing `Select`, `RadioGroup`, `Collapsible`, dialog, and
  permission-grid patterns, and no new dependency.
- Each grant fetches roles for its selected workspace and excludes Owner.
  Detailed permission overwrites stay collapsed by default.
- The success view shows a copyable link and the warning that it cannot be
  shown again. Active rows show metadata and revoke through the shared
  confirmation dialog.
- `/invite` is public and sits outside the authenticated onboarding gate. It
  reads the URL fragment once, clears it, and renders preview, provider,
  authenticated-detail, success, or generic-invalid states in one thin page
  with named components.
- Instance settings replaces the Allowlist section with registered-account
  management. OAuth-provider copy no longer tells administrators to allowlist
  usernames or email addresses.
- All new UI is mobile-first and verified at 320, 375, 414, and 768 pixels.
  It extends the existing Mono Console with hairline rows, current dialogs,
  semantic tokens, and no new visual theme.

## Commands

- Generate SQL: `make sqlc`
- Check generated SQL: `make sqlc-check`
- Go focused tests: `go test ./internal/auth ./internal/tenancy ./internal/platform/storage ./internal/mcp ./internal/eventcatalog ./server/cmd`
- Go build: `go build ./...`
- Go vet: `go vet ./...`
- Go lint: `make lint`
- Go vulnerabilities: `make vuln`
- Go coverage: `make coverage`
- Web install per worktree: `bun install --cwd web --frozen-lockfile`
- Web types: `bun run --cwd web typecheck`
- Web lint: `bun run --cwd web lint`
- Web tests: `bun run --cwd web test -- --coverage`
- Web build: `bun run --cwd web build`

## Code style

- Go follows domain model to repository interface to use-case to HTTP/MCP
  adapters. It uses early returns, consumer-side interfaces, parameterized
  sqlc queries, and one-line why-only comments.
- React follows Page to Feed to Section to Row, explicit negative-first query
  states, `&&` component rendering, TanStack Query for server state, React
  Hook Form for form state, named components, and no prop drilling.
- The implementation reuses the standard library, `google/uuid`, existing
  permission sets, existing dialogs, and existing UI primitives. It adds no
  dependency.

## Testing strategy

Tests are written before behavior. Error and abuse paths come first.

- Unit tests cover validation, account-state transitions, all-workspace
  permission gates, protected roles, expiry, generic token failures, and UI
  form behavior.
- Real-SQLite integration tests cover atomic new-user admission, existing-user
  redemption, preserved memberships and overwrites, all-or-nothing rollback,
  concurrent one-winner consumption, tombstoning, migration, and outbox rows.
- HTTP and MCP tests prove adapter parity and that list/preview results never
  expose token material.
- Frontend tests cover multi-grant creation, duplicate prevention, expiry,
  one-time reveal, revoke, preview, OAuth choices, detailed acceptance,
  decline, generic invalid state, and post-redemption navigation.
- A browser walkthrough proves both a new-user OAuth registration and an
  already-authenticated redemption at the four required widths.

## Boundaries

Always:

- Validate every grant and the creator's current authority on the server.
- Hash the bearer token, redact it everywhere, and use a generic public error.
- Commit user admission, memberships, overwrites, consumption, and events in
  one SQLite transaction.
- Preserve existing memberships and overwrites.
- Keep browser and MCP behavior behind one use-case layer.

Ask first:

- Adding a dependency, changing the OAuth provider set, changing the
  permission vocabulary, or introducing a background scheduler.

Never:

- Store or log the raw invitation token.
- Put it in OAuth state, provider URLs, callback URLs, session claims, query
  strings, or persistent browser storage.
- Permit public registration, reusable links, Owner grants, partial
  redemption, or client-side-only authorization.
- Retain the old allowlist as a second admission path.

## Success criteria

- A manager can create a one-day or seven-day link spanning several
  workspaces, copy it once, list its metadata, and revoke it through both HTTP
  and MCP.
- A new person can open the link, choose an enabled OAuth provider, register,
  and receive every grant atomically without the inviter knowing their
  identity.
- An existing active user can redeem the same kind of link without changing
  any membership they already hold.
- The link cannot be reused, raced into two successes, recovered from storage,
  or distinguished from another invalid token through public errors.
- Unknown OAuth identities without a valid link are rejected without creating
  a user row. Disabled and removed users are rejected immediately on every
  authenticated surface.
- The manual allowlist and login-bound pending invitation UI, API, storage,
  and copy are gone.
- Account disable, remove, reactivate, and restore preserve the private
  instance boundary and cannot remove the last active instance admin.
- Events, OpenAPI registration, `CONTEXT.md`, affected ADRs, README access
  summary, and tests agree with the shipped behavior.
- Local Go and web CI-equivalent checks pass, and the browser flow is verified
  at 320, 375, 414, and 768 pixels.

## Open questions

None. The owner approved the product behavior during the Wayfinder grilling
rounds and authorized implementation.
