# Instance administration is permission bits held in any workspace

ADR 0024 made instance-level capability a flag on the user, `can_create_workspace`, outside the permission table,
because the table was scoped to a workspace and creating one happens before the creator belongs anywhere. Since
ADR 0087 the table answers instance-level checks too: runners, the topology, DNS, and connectors are checked against
every workspace the caller belongs to. The flag was left as a second, all-or-nothing way to hold power, and it could
not be split: whoever could rotate a connector's app could also remove accounts and upgrade the instance, and an
owner who wanted a colleague to do one of those had to hand over all of them.

Decision: the flag is gone, and every capability it opened is a permission bit checked in any workspace the caller
belongs to, the same way as ADR 0087's instance-level areas:

| Capability | Permission |
|---|---|
| Instance URL, sign-in providers, upgrading, failed events | `instance:write` |
| Version and upgrade facts, connection token, public address, failed events | `instance:read` |
| Reading accounts, the directory, the whole Team | `accounts:read` |
| Disabling, reactivating, restoring an account | `accounts:write` |
| Removing an account | `accounts:delete` |
| Creating a workspace | `workspaces:create` (a verb, ADR 0057) |
| Enrolling and removing a runner | `runners:write`, `runners:delete` |
| Enrolling and removing an automations host | `automations:write`, `automations:delete` |
| Registering a connector's app | `connectors:write` |
| Integration installs, subscriptions, deliveries | `integrations:read`, `integrations:write`, `integrations:delete` |
| The audit log | `audit:read` |

Renaming a workspace takes `workspaces:write` in that workspace, where the entity lives. The Owner of any workspace
holds every bit through ADR 0042's bypass, so the first user, whom the owner wizard makes the default workspace's
Owner, can do everything with no flag. `/api/auth/me` reports the bits a user holds anywhere as
`instance_permissions`, so a client shows the instance-level areas the server lets through, not the ones one
selected workspace allows.

Two rules come with it, because a bit in a role now carries instance power:

- **Nobody grants a bit they don't hold.** Creating, editing, or cloning a role, assigning a role, setting an allow
  override, and an invitation package may only carry what the giver holds in the workspace the grant lands in. An
  edit is judged by what it adds, so a permission a role or override already carried may stay. Invitations are
  checked again when redeemed, so one whose creator has since lost a bit no longer admits anyone. Adding a deny is
  never limited, but lifting one hands back what the role gives, so it counts as a grant. The Owner holds
  everything, so the Owner is never refused.
- **The last active Owner can't be disabled or removed**, which replaces ADR 0061's last active administrator.

The consequences worth knowing: `workspaces:create` is as strong as Owner, since a workspace's creator becomes its
Owner and an Owner anywhere holds every instance-level bit, so it belongs only in a role meant for that. An upgraded
instance keeps its Owner's power; a former administrator who is not an Owner keeps only what their roles grant, and
an instance whose administrators held no active Owner role has its earliest active administrator made the default
workspace's Owner by the migration, so an upgrade never locks everyone out. Scoped tokens stay held to their
creator's grant (ADR 0087), and a token still can never start an upgrade or enroll a runner, whatever its scopes.

Rejected: keeping the flag and adding bits beside it, which left two answers to "may this person do X"; and checking
instance bits only in the default workspace, which made that one workspace special and broke for an owner who works
from another.

Supersedes the instance-administration layer of ADR 0024 (sign-in and per-membership roles stand) and the last
active administrator rule of ADR 0061. Amends ADR 0085, whose Team now answers to `accounts:read`, and ADR 0087.
Decided 2026-09-29. Amended by ADR 0097: a restricted membership holds no instance bit, whatever its role carries.
Amended by ADR 0103: `templates:write` edits the instance templates.
