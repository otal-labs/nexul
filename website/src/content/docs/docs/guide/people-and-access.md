---
title: People and access
description: Invite people, give them a role in each workspace, and keep a client to the one project they work on.
sidebar:
  order: 9
---

Your instance is private: nobody signs in without an invitation. Each person then has a role in each workspace they're in, and the role decides what they can do there. A person can also be held to chosen projects, so a client sees their project and nothing else.

## Invite someone

1. Open **Settings → Team** and press **Invite**.
2. Under **Link expires in**, choose **1 day** or **7 days**.
3. Under **Workspace access**, choose a workspace and a role. **Add workspace** adds another.
4. To hold them to some projects, set **Every project** to **Chosen projects** and give each project a level. See [Hold someone to chosen projects](#hold-someone-to-chosen-projects).
5. Optionally open **Optional permission overrides** to allow or deny single permissions on top of the role.
6. Press **Create link**. Team switches to its **Invitations** tab with the new link on top; press **Copy link**. It's shown once.
7. Send the link however you like. The person opens it, signs in with one of the instance's sign-in providers, reviews what they're being given, and accepts.

A link works once. The Owner role is never offered; ownership doesn't move through an invitation. The **Invitations** tab lists the links not used yet under **Active invitation links**; **Revoke invitation** kills one.

## Manage people

**Settings → Team** has two tabs. **People** lists everyone on the instance: a dot on each picture for who is online, when the others were last seen, the GitHub, Google or Discord marks of the accounts they sign in with, each beside its GitHub or Discord username, their role or how many workspaces they are in, and **Disabled** or **Removed** for an account that can't sign in. Open a person to see one tab per workspace they're in. Each tab holds their **Role**, **Edit overrides**, their projects, and **Remove from workspace**; the **+** after the last tab adds them to another workspace.

Nothing applies as you edit. A tab with changes shows a dot, **Confirm** applies them all, and **Cancel** asks before throwing them away. If one change is refused, the dialog says which and keeps the rest waiting for the next **Confirm**.

Account actions apply at once, each after its own confirmation: **Disable** stops someone signing in and keeps their access, **Reactivate** undoes it, and **Remove account** removes them. **Restore** brings a removed account back with no workspace access. Disable and Remove account also disconnect the person's computers: each one's runner uninstalls itself, and you are told only that their computers were disconnected, never which ones. Reactivating or restoring the account brings none back; the person adds each computer again from their own settings.

Seeing everyone takes `accounts:read`. Changing someone in a workspace takes `members:write` there. Without `accounts:read`, someone who manages members finds Team under **Configuration** instead, showing only the workspaces they manage.

## Roles

Every workspace has an **Owner** role. It can't be renamed, deleted, or limited: the Owner holds every permission and bypasses overrides. Every other role is yours to define in **Configuration → Roles**.

**New role** and a role's **Edit** open the role editor, a dialog with the name at the top and Save at the bottom. A new role can **Start from** a copy of another role; the copy is yours to change and the other role stays as it is. Nothing is saved until **Create role** or **Save**, and closing with changes asks before it throws them away.

A role sets one level per area: **None**, **Read**, **Write**, or **Delete**, each level including the ones before it. Some areas carry extra checkboxes under their name, such as docs' **Threads**, **Clone**, and **Lock**, which need **Read** first. The editor has three tabs, each with the number of areas it allows, a dot while it holds unsaved changes, and a top row that sets every area in it at once:

- **Instance**: areas that apply across every workspace on this install, such as runners, machines, DNS and gateways, connectors, accounts, and the audit log.
- **Workspace**: areas such as chat, channels, plays, members, and roles.
- **Every project**: a project's areas, such as tickets, docs, memories, and stacks, applied on every project.

Each role's row counts its areas per level, names the first ones it has no access to, and shows who holds it; its chevron opens every area. Its **…** menu edits, duplicates (the copy is named `Admin (copy)`), clones to another workspace, or deletes it. A role someone still holds can't be deleted: give them another role on **Team** first.

Nobody can hand out a permission they don't hold: not in a role, an override, or an invitation. Every permission is written `<area>:<action>`, such as `docs:write`, and the same names gate a person, a personal access token, an automation, and an agent.

### Instance permissions

Some permissions cover the whole instance, not one workspace: instance settings and upgrades, accounts, runners and machines, DNS, connectors, automations, integrations, templates, and creating workspaces. Holding one in any workspace you're in is enough. There's no separate administrator: whoever owns a workspace holds all of them, which is why `workspaces:create` is as strong as Owner.

## Hold someone to chosen projects

A person's **Every project** row decides which projects they see:

- **From role**: every project at their role's level, new projects included. This is the default.
- **Chosen projects**: only the projects you give them. A project with no level stays hidden, its name included, and so do projects made later. A link to one reads as not found.

Under **Chosen projects**, each project gets one level, and **Areas** opens its areas (tickets, docs, memories, and so on) to set each on its own. Switching back to **From role** keeps the levels you chose, ready for next time.

Someone on chosen projects:

- can't create a project;
- reads no public channel, only the [private channels](/docs/guide/chat-and-voice/#private-channels) they're in, their direct messages, and threads of what they can open;
- can create only private channels;
- holds no instance permission, whatever their role says.

Their tokens and agents see exactly what they see. A change reaches their open browser at once: a project taken away drops out of their sidebar.

A project's **Settings → General** lists, under **People with access**, everyone on chosen projects who can open it. Deleting a project names who loses it before you confirm.

## Permissions on a doc or a play

Right-click a doc in the Docs list or a play in **Configuration → Plays**, or open its **…** menu, and pick **Permissions…**; a doc's settings menu has it too. It shows only to someone who can change that doc's or play's rules. The list on the left holds everyone with a rule here; **Add person** lists someone else. A doc refuses someone who can't open its project and says so.

Pick a person to set their rule, one row per permission, each with three buttons:

- **Allow** (check): they have it here, whatever their role says.
- **Fallback** (slash): this rule says nothing, so their role decides. The line under the row says what they get then, and from where.
- **Deny** (cross): they don't have it here, whatever their role says.

A doc carries the docs permissions and **Manage this doc's access**; a play carries **Run**. **Remove from this doc** sets every row back to Fallback. Nothing changes until you press **Save**, and anyone else with the panel open sees the change without a refresh.

You can't allow a permission you don't hold on that doc or play, or lift a Deny on one: its button shows a lock and says why. Adding a Deny is never limited. **Check access** shows what one person can do here, each answer with the rule or role that decided it.

Changing a doc's rules takes **Manage this doc's access** on it, which its creator has; a play's takes `plays:write`.

## Why someone can or can't

Every answer has a source: the Owner role, the person's role, an override on the workspace, a doc or a play, or their access to a project. When Nexul refuses something you can see, the message names it, such as `docs:write required (Denied on doc Pricing for bob)` or `tickets:write required (Not allowed: role Client doesn't grant it)`. Something you can't see is not found and says nothing more, so it never confirms it exists.

To see every answer on one doc, play, or project at once, each with its source and the answer it would have without that doc's, play's, or project's own override:

```
GET /api/permissions/explain?resource_type=doc&resource_id=<id>&user_id=<id>
```

Leave out `user_id` for your own answers, which take only being able to open it. Someone else's take managing it: `permissions:write` on a doc, `plays:write` on a play, or `members:write` in a project's workspace. Agents ask the same with `permission_overwrite_list` and a `user_id`.

## Names and pictures

Every member of a workspace sees its people's login, display name, and picture, and nothing about their roles. Set your own name and picture in **Settings → Profile**; a change reaches everyone's open screens without a refresh.

## Sign-in accounts and GitHub

**Settings → Profile** lists the accounts you sign in with under **Sign-in accounts**. **Link** adds another provider to the same profile, and **Unlink** removes one, never your last.

Under **GitHub**, the same page shows whether your own GitHub account lists your repositories: **Connected** with your GitHub login, **Reconnect needed** after GitHub refused to refresh it, or **Not connected**. Signing in with GitHub connects it. If you sign in with Google or Discord, **Connect GitHub** runs GitHub's sign-in and links it to this profile; a GitHub account already linked to someone else is refused. **Disconnect** forgets the token, and unlinking your GitHub sign-in does too. Nexul lists repositories only with your own GitHub, never with someone else's or the instance's; see [GitHub App](/docs/guide/github-app/).

## Over MCP

Agents manage people with `account_list` (each person's `roles` per workspace), `account_update`, `invitation_create`, `invitation_list`, `invitation_delete`, `role_update`, `role_delete`, and the `permission_overwrite_*` tools; `permission_overwrite_update` takes a `state` of `allow`, `fallback`, or `deny`. `account_update` and `invitation_create` take `every_project` (`role` or `none`) and `project_access` per workspace. See [MCP server](/docs/guide/mcp-server/).
