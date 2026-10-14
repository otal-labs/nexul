# Invitations that explain themselves

**Status:** ready-for-agent

## Problem

The invite page is the first thing a newcomer sees. Today it shows each workspace with role chips and nothing about
what those roles mean or which projects they reach. The accept line still talks about "overrides", which no longer
exist. When an invitation has expired or been deleted, it says only "invalid or expired". When someone who already
has an account opens a link, they are let in and the link is used up, so the person it was meant for can't use it.

## Decisions

Made with the owner on 2026-10-14. The look was decided in auto design mode on the never-merged branch
`proto/invitations`; the report and references are in `~/Code/Images/invitations/`.

1. **Every role gets a one-line description.**
   - It shows on the invite page, the admin's invite form and the role editor.
   - The five template roles ship with one. Roles that already exist keep an empty description, with no backfill:
     copies are never rewritten, and a role with none shows its name alone.
2. **Before sign-in** the invite page shows:
   - the instance
   - each workspace
   - the roles given there, with their descriptions

   It shows no project names, so a forwarded link reveals nothing beyond what the roles are.
3. **After sign-in, the accept step adds the projects those roles currently reach**, with "You may be added to more
   projects later."
4. **An expired or deleted invitation** says only "This invitation has been deleted or has expired". It names no
   inviter and notifies no one.
5. **An existing account is turned away**: "You already have an account here. This invitation is for someone new."
   The link stays unused for the person it was meant for. Adding an existing person to a workspace happens on Team
   (`CONTEXT.md`, Invitation).
6. **No email.** The admin copies the link and sends it themselves. Sign-in is through whichever providers the owner
   turned on.
7. **The Invitations tab** (accepted, and expired or deleted, in the last 30 days) shipped with the welcome wizard
   (#643) and stays as it is.

## Look ("Consent", the pick; the runner-up is "Role cards")

### The invite page

- **Frame**: a centred `panel` at 72% on the still showcase field, `max-w-[28rem]`, padding 24px (32px from `sm`).
- **Header**:
  - The 40px instance logo, a muted 16px exchange arrow, then a dashed 40px circle with a person icon. After sign-in
    the circle becomes the person's gradient avatar.
  - The microheader "INVITATION", a Fraunces 1.75rem headline "Join {instance}", and the host in 12px mono muted.
- **Each workspace**:
  - A 20px `surface-2` tile with the mono initial, and the name in 13px medium.
  - Under it, one `bg-card` card with hairline dividers and one row per role, padded 14px by 12px. Each row has the
    name in 14px medium over the description in 13px muted.
- **After sign-in**:
  - Each card gains a footer on a 60% `surface-2` tint: a folder icon, then "2 projects: Website relaunch, Mobile
    app", or "No projects yet". Past 8 names it folds into "and N more".
  - Under the cards sits the line "You may be added to more projects later."
  - Then an identity row ("Lena Fischer @lena · GitHub") and two half-width buttons, Decline (outline) and Accept
    invitation (ember).
- **Before sign-in**:
  - "Sign in to accept".
  - Full-width provider buttons: the first in ember, the rest outline.
- **Footer**: "Works once · expires 21 Oct, 14:00" in 11px mono muted.
- **Verified widths**: 768, 1024 and 1440, plus 390 for clients on phones.

### Refusals

- **Expired or deleted**:
  - The same panel, centred, with the logo at 70% opacity.
  - "INVITATION", then a Fraunces 1.5rem "This invitation has been deleted or has expired", then the host.
  - No action.
- **Existing account**:
  - The logo and the person's avatar.
  - "You already have an account here" over "This invitation is for someone new."
  - Their identity row.
  - One ember button, "Go to {instance}", with the muted line "The link stays unused, so it still works for the
    person it was meant for."

### The admin's invite form

- The dialog's description reads: "For someone without an account here. Pick the workspaces and the roles the link
  gives, then copy it and send it yourself. It works once."
- Each role row puts its description under the name, in 12px muted at an 18px line height.

### The role editor

- A one-line Description field sits under Name.
- Its placeholder is "What someone with this role does, in one line".
- A mono counter (for example "70 / 140") turns amber with 20 characters left. The cap is 140.
- The help line reads: "Shown on invitations and wherever someone picks roles."

## Architecture

- **Migration**: `roles.description TEXT NOT NULL DEFAULT ''`, forward-only, with no backfill.
- **Roles carry the description**:
  - It travels on role create, update and clone, in `role_update` and `workspace_list`, and in `tenancy.TeamRole`
    and `auth.InvitationRole`.
  - `roles.TemplateBody` gains `description` (`ParseTemplate` refuses unknown fields, so it has to be added there),
    and the five shipped templates get their text.
  - It is capped at 140 characters on the server.
- **Invite preview** (`POST /api/invitations/preview`, public):
  - It adds `instance_name` and a description per role.
  - It drops the dead `every_project` and `projects` fields, keeping them in the response only if the HTTP API's
    grow-only rule requires it (check `practices/go.md`).
  - Project names are never returned here.
- **Acceptance** (`POST /api/invitations/acceptance`, after sign-in):
  - It adds a description per role and `projects[{id, name}]` per grant: the projects whose rules allow
    `projects:open` to any of the grant's roles.
  - That needs a role → projects lookup, one statement for all the grant's roles.
  - It drops the empty `allow` and `deny`.
- **Turning an existing account away**:
  - `CompleteInvitationOAuth` and `PrepareAuthenticatedAcceptance` already know the account before the invitation is
    touched. Both return a distinct error code, `existing_account`, so the page can show its screen, and leave the
    invitation unredeemed.
  - `Redeem` refuses an existing user as a backstop. The leftover sign-in handoff expires on its own.
  - Expired and deleted already share one not-found error.

## Hit every surface

- **UI**:
  - the invite page before and after sign-in
  - both refusals
  - the admin form
  - the role editor's Description field

  Verified at 768, 1024 and 1440, and the invite page at 390.
- **HTTP**: preview and acceptance, as above, with role routes carrying `description`.
- **MCP**: `role_update` takes `description`; `workspace_list` returns it; `invitation_create` is unchanged.
- **Events**: `role.updated` carries `description` beside its fields, then `make event-schemas`.
- **Live**: none new.
- **Phone**: the phone app has no invite flow, so no change.
- **Permissions**:
  - The preview stays public and shows no project names.
  - The acceptance needs the sign-in.
  - Descriptions follow `roles:write`.
- **Reverse states**: a turned-away link stays usable; Decline leaves it unused too.
- **Docs**:
  - `CONTEXT.md`'s Invitation entry already says what's decided.
  - No ADR is needed unless turning existing accounts away surprises a reader. It is recorded in `CONTEXT.md`.
- **User guide**: `people-and-access.md` (inviting, role descriptions, what the invitee sees) and `setup-wizard.md`
  if it describes inviting.

## Out of scope

- Email.
- Inviting someone who already has an account. Team adds them.
- Notifying the inviter when a link fails.

## Build order

The tickets are in `issues/`:
1. 01 comes first.
2. 02, 03 and 04 follow and can run in parallel. 03 doesn't need 01.
3. 05 comes last.
