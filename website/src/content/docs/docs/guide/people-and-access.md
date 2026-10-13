---
title: People and access
description: Invite people, give them roles in each workspace, and choose who's in each project.
sidebar:
  order: 9
---

Your instance is private: nobody signs in without an invitation. Each person then holds roles in each workspace they're in, several or none, and together their roles decide what they can do there. Projects are opt-in: a person sees a project only once it lists one of their roles, or them by name, so a client sees their project and nothing else.

## Invite someone

1. Open **Settings → Team** and press **Invite**.
2. Under **Link expires in**, choose **1 day** or **7 days**.
3. Under **Workspace access**, choose a workspace, then tick the roles the link gives there under **Roles in** that workspace. Nothing is ticked at first, and each workspace needs at least one role: **Create link** stays off until it has one. **Add workspace** adds another.
4. Press **Create link**. Team switches to its **Invitations** tab with the new link on top; press **Copy link**. It's shown once.
5. Send the link however you like. The person opens it, signs in with one of the instance's sign-in providers, reviews each workspace and the roles they're being given, and accepts.

A link works once and gives nothing but its roles. The Owner role is never offered, and a role holding more than you hold sits under **Can't give** with what it holds that you don't; ownership doesn't move through an invitation. The **Invitations** tab lists the links not used yet under **Active invitation links**; **Revoke invitation** kills one.

## Manage people

**Settings → Team** has two tabs. **People** lists everyone on the instance: a dot on each picture for who is online, when the others were last seen, the GitHub, Google or Discord marks of the accounts they sign in with, each beside its GitHub or Discord username, their roles as chips (three, then a count whose tooltip names the rest, or **No roles**) or how many workspaces they are in, and **Disabled** or **Removed** for an account that can't sign in. Open a person to see one tab per workspace they're in. Each tab holds their **Roles**, one checkbox per role with how much it allows beside it, and **Remove from workspace**; the **+** after the last tab adds them to another workspace, where you tick at least one role before **Confirm**.

The roles you may give come first. Under **Can't give** sit the Owner role and every role holding something you don't, each with the reason; their boxes keep the person's tick and can't be changed, since you can't take away a role you couldn't give. Unticking the last role warns that the person will see no projects or plays there, but you can still confirm it.

Nothing applies as you edit. A tab with changes shows a dot, **Confirm** applies them all, and **Cancel** asks before throwing them away. If one change is refused, the dialog says which and keeps the rest waiting for the next **Confirm**.

Account actions apply at once, each after its own confirmation: **Disable** stops someone signing in and keeps their access, **Reactivate** undoes it, and **Remove account** removes them. **Restore** brings a removed account back with no workspace access. Disable and Remove account also disconnect the person's computers: each one's runner uninstalls itself, and you are told only that their computers were disconnected, never which ones. Reactivating or restoring the account brings none back; the person adds each computer again from their own settings.

A person with no role in a workspace stays a member. Opening it, they see **No roles in** that workspace, a line saying what that means, and its Owners, each a button that opens a direct message with them, which takes no role. Their sidebar shows no project; channels and direct messages show once they have some, and the public channels and their direct messages work as usual.

Everyone sees their own roles in **Settings → Profile → Your roles**, one line per workspace, and the account row at the bottom of the sidebar lists them for the workspace in view.

Seeing everyone takes `accounts:read`. Changing someone in a workspace takes `members:write` there. Without `accounts:read`, someone who manages members finds Team under **Configuration** instead, showing only the workspaces they manage.

## Roles

Every workspace has an **Owner** role. It can't be renamed, deleted, or limited: the Owner holds every permission and bypasses overrides. Every other role is yours to define in **Configuration → Roles**.

**New role** and a role's **Edit** open the role editor, a dialog with the name at the top and Save at the bottom. A new role can **Start from** a copy of another role; the copy is yours to change and the other role stays as it is. Nothing is saved until **Create role** or **Save**, and closing with changes asks before it throws them away.

A role sets one level per area: **None**, **Read**, **Write**, or **Delete**, each level including the ones before it. Plays' first level is **See**. Some areas carry extra checkboxes under their name, such as docs' **Threads**, **Clone**, and **Lock**, which need **Read** first (plays' **Run** needs **See**). A few permissions are a plain yes or no, set with a switch at the top of their tab. The editor has three tabs, each with the number of areas and switches it allows, a dot while it holds unsaved changes, and a top row that sets every area in it at once, leaving the switches as they are:

- **Instance**: areas that apply across every workspace on this install, such as runners, machines, DNS and gateways, connectors, accounts, and the audit log.
- **Workspace**: across the workspace, whatever project someone is in. It leads with **Create projects** (`projects:create`) and **Edit the workspace Interview template** (`memories:template`), then areas such as chat, channels, plays, members, and roles.
- **In projects**: a project's areas, such as tickets, docs, memories, and stacks, which a holder has only inside the projects they're in. It leads with **Manage this project's access** (`projects:access`): choosing who's in a project and what they can do there.

Each role's row counts its areas per level, names the first ones it has no access to, and shows who holds it; its chevron opens every area. Its **…** menu edits, duplicates (the copy is named `Admin (copy)`), clones to another workspace, or deletes it. Deleting a role someone holds asks first, saying how many people lose it, then takes it from them, along with the rules that name it and the pending invites that give it; someone left with no role stays a member and sees no project.

Nobody can hand out a permission they don't hold: not in a role, an override, or an invitation. Every permission is written `<area>:<action>`, such as `docs:write`, and the same names gate a person, a personal access token, an automation, and an agent.

### Instance permissions

Some permissions cover the whole instance, not one workspace: instance settings and upgrades, accounts, runners and machines, DNS, connectors, automations, integrations, templates, and creating workspaces. Holding one in any workspace you're in is enough. There's no separate administrator: whoever owns a workspace holds all of them, which is why `workspaces:create` is as strong as Owner.

## Who's in each project

**Access** in the sidebar's Workspace section, between Automations and Configuration, is where you choose who's in each project. It shows to anyone who can manage a project's access (`projects:access`); the Owner sees every project, anyone else only the projects they're in and manage.

The **Projects** tab is a grid: one row per project, one column per role, with how many people hold each role under its name. Tick a role on a project and everyone holding it sees the project and everything in it. The **Every project** row ticks a role on every project at once, or clears it from all of them when every project already has it; past ten projects a filter narrows the rows, and the top row then acts on the rows shown. **People**, the last column, names someone on a project whatever roles they hold: the add button opens a list of the workspace's people. A project that lists nobody says **Owners only**, since only Owners see it.

The grid is the whole workspace at a glance. For one project, its settings' **People in this project** lists everyone in it, through which role or by name, and anyone kept out of it by name.

The **Public channels** tab is the Owner's. Every member sees a public channel, so every box starts ticked; untick a role to hide that channel from everyone holding it. Private channels keep their own members.

Each tab saves from its own footer, and the change reaches everyone at once: a project someone was just listed in shows up in their sidebar without a reload. Leaving the page with unsaved ticks asks first.

### After upgrading

The release that made projects opt-in wiped every per-person permission, so right after upgrading no project lists anyone and only Owners see projects. An Owner's first visit to the workspace then opens **Who's in each project**: the Projects grid and the Public channels grid on one page, and one **Finish setup** that saves both. Until an Owner finishes, the sidebar offers Owners **Set up access** to get back to it, and members see **Projects being set up**; any page that would show a project tells them their projects are on their way, and chat works as usual. **Finish setup** works with projects left empty: they stay Owners only until you list someone. Afterwards the page is the workspace's Access page, which shows who finished the setup and when.

## Permissions on a doc or a play

Right-click a doc in the Docs list or a play in **Configuration → Plays** and pick **Permissions…**; a doc's settings menu has it too. It shows only to someone who can change that doc's or play's rules. The list on the left holds everyone with a rule here; **Add person** lists someone else. A doc refuses someone who can't open its project and says so.

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

Leave out `user_id` for your own answers, which take only being able to open it. Someone else's take managing it: `permissions:write` on a doc, `plays:write` on a play, or `projects:access` on a project. Agents ask the same with `permission_overwrite_list` and a `user_id`.

## Names and pictures

Every member of a workspace sees its people's login, display name, and picture, and of their roles only who is an Owner. Set your own name and picture in **Settings → Profile**; a change reaches everyone's open screens without a refresh.

## Sign-in accounts and GitHub

**Settings → Profile** lists the accounts you sign in with under **Sign-in accounts**. **Link** adds another provider to the same profile, and **Unlink** removes one, never your last.

Under **GitHub**, the same page shows whether your own GitHub account lists your repositories: **Connected** with your GitHub login, **Reconnect needed** after GitHub refused to refresh it, or **Not connected**. Signing in with GitHub connects it. If you sign in with Google or Discord, **Connect GitHub** runs GitHub's sign-in and links it to this profile; a GitHub account already linked to someone else is refused. **Disconnect** forgets the token, and unlinking your GitHub sign-in does too. Nexul lists repositories only with your own GitHub, never with someone else's or the instance's; see [GitHub App](/docs/guide/github-app/).

## Over MCP

Agents manage people with `account_list` (each person's `roles` per workspace), `account_update`, `invitation_create`, `invitation_list`, `invitation_delete`, `role_update`, `role_delete`, and the `permission_overwrite_*` tools. `account_update` gives and takes away roles per workspace with `add_roles` and `remove_roles`, and `invitation_create` takes `roles` per workspace, at least one; neither sets anything per person. `permission_overwrite_update` sets a `state` of `allow`, `fallback`, or `deny` for people (`user_ids`) or roles (`role_ids`) on a doc, play, project, or channel: allowing `projects:open` on a project lists a role or a person in it, denying it keeps a person out, and denying `channels:read` on a public channel hides it, which only an Owner may do. `workspace_update` with `finish_access_setup` ends the one-time setup after an upgrade, and `workspace_list` says whether it's done. See [MCP server](/docs/guide/mcp-server/).
