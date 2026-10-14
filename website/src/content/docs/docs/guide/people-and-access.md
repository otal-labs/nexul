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

A person with no role in a workspace stays a member. Opening it, they see **No roles in** that workspace, a line saying what that means, and its Owners, each a button that opens a direct message with them, which takes no role. Their sidebar shows no project; channels and direct messages show once they have some. They can read the public channels but not post in them, and their direct messages work as usual, so they can always message an Owner; posting in a channel takes a role that allows it.

Everyone sees their own roles in **Settings → Profile → Your roles**, one line per workspace, and the account row at the bottom of the sidebar lists them for the workspace in view.

Seeing everyone takes `accounts:read`. Changing someone in a workspace takes `members:write` there. Without `accounts:read`, someone who manages members finds Team under **Configuration** instead, showing only the workspaces they manage.

## Roles

Every workspace has an **Owner** role. It can't be renamed, deleted, or limited: the Owner holds every permission, is in every project, and no rule can deny them. Every other role is yours to define in **Configuration → Roles**.

**New role** and a role's **Edit** open the role editor, a dialog with the name at the top and Save at the bottom. A new role can **Start from** a copy of another role; the copy is yours to change and the other role stays as it is. Nothing is saved until **Create role** or **Save**, and closing with changes asks before it throws them away.

A role sets one level per area: **None**, **Read**, **Write**, or **Delete**, each level including the ones before it. Plays' first level is **See**. Some areas carry extra checkboxes under their name, such as docs' **Threads**, **Clone**, and **Lock**, which need **Read** first (plays' **Run** needs **See**). A few permissions are a plain yes or no, set with a switch at the top of their tab. The editor has three tabs, each with the number of areas and switches it allows, a dot while it holds unsaved changes, and a top row that sets every area in it at once, leaving the switches as they are:

- **Instance**: areas that apply across every workspace on this install, such as runners, machines, DNS and gateways, connectors, accounts, and the audit log.
- **Workspace**: across the workspace, whatever project someone is in. It leads with **Create projects** (`projects:create`) and **Edit the workspace Interview template** (`memories:template`), then areas such as chat, channels, plays, members, and roles.
- **In projects**: a project's areas, such as tickets, docs, memories, and stacks, which a holder has only inside the projects they're in. It leads with **Manage this project's access** (`projects:access`): choosing who's in a project and what they can do there.

Each role's row counts its areas per level, names the first ones it has no access to, and shows who holds it; its chevron opens every area. Its **…** menu edits, duplicates (the copy is named `Admin (copy)`), clones to another workspace, or deletes it. Deleting a role someone holds asks first, saying how many people lose it, then takes it from them, along with the rules that name it and the pending invites that give it; someone left with no role stays a member and sees no project.

Nobody can hand out a permission they don't hold: not in a role, a rule's Allow, or an invitation, and can't lift a Deny on one. Every permission is written `<area>:<action>`, such as `docs:write`, and the same names gate a person, a personal access token, an automation, and an agent.

### Template roles

Five roles ship with Nexul as templates. Every new workspace, a fresh install's first one included, starts with a copy of each beside its Owner, with the permissions and Welcome wizard steps the template holds when the workspace is created. From then on the copy belongs to the workspace: editing or deleting a template changes only workspaces created afterwards, and a workspace that existed before gets none. Projects are opt-in, so a copied role sees no project until a project lists it.

| Template role | What it can do | Welcome steps |
|---|---|---|
| Client | Read and write tickets, docs (with their threads) and attachments; read memories | Your projects |
| Junior Developer | The same, plus read pull requests, code reviews, repositories, stacks and deploys with their logs, and see and run plays | Your computer, Your projects |
| Senior Developer | Delete in most project areas, write repositories, stacks and deploys, create projects, edit plays and auto plays, edit the Interview template, and read runners, machines, DNS and the topology | Connect GitHub, Your computer |
| DevOps | Write tickets, docs, repositories, stacks and deploys, delete stacks, and manage runners, machines, DNS and gateways, and the topology registry | Add a deploy server |
| Administrator | Every project and workspace permission plus the deploy infrastructure, but none of the instance's own administration, which stays with the Owner | Connect GitHub |

All five can chat and use channels, voice, mentions, and notifications.

**Settings → Templates** lists them on its **Roles** card, each with its levels, its Welcome steps, and whether it is the default or edited. A row's **…** menu opens the same role editor as a workspace role, with its Instance, Workspace, In projects, and Welcome tabs. The five that ship have **Reset to default** and can't be deleted. **New template role** adds one of your own, and its **Delete** removes it; workspaces that already have a copy keep theirs. Changing template roles takes `templates:write`.

### Instance permissions

Some permissions cover the whole instance, not one workspace: instance settings and upgrades, accounts, runners and machines, DNS, connectors, automations, integrations, templates, and creating workspaces. Holding one in any workspace you're in is enough. There's no separate administrator: whoever owns a workspace holds all of them, which is why `workspaces:create` is as strong as Owner.

To hand out one part of the instance without a custom role, such as letting bob manage DNS but not connectors, the instance Owner (the Owner of the default workspace) sets a rule on that **instance area** for a role or a person, the same Allow, Fallback and Deny as on a project. A role rule names a role of any workspace and reaches its holders. The areas and what each rule may set:

| Area | Key | Permissions |
|---|---|---|
| The instance itself: its URL, upgrades, failed events, public address | `instance` | `instance:read`, `instance:write`, `events:read` |
| Sign-in providers | `sign_in_providers` | `instance:read`, `instance:write` |
| Each connector | `connector:github`, `connector:cloudflare`, `connector:livekit` | `connectors:read`, `connectors:write` |
| DNS and gateways | `dns` | `dns:read`, `dns:write`, `dns:delete` |
| Instance templates | `templates` | `templates:read`, `templates:write` |
| Accounts | `accounts` | `accounts:read`, `accounts:write`, `accounts:delete` |
| Runners and machines | `runners` | `runners:read`, `runners:write`, `runners:delete`, `machines:read`, `machines:write` |
| The topology registry | `topology` | `topology:read`, `topology:write`, `topology:delete` |
| Automation hosts | `automations` | `automations:read`, `automations:write`, `automations:delete` |
| Integrations | `integrations` | `integrations:read`, `integrations:write`, `integrations:delete` |
| The audit log | `audit` | `audit:read` |

A person holds an area's permission when one of their roles in any workspace grants it, unless a rule there says otherwise: between the rules for their roles Allow wins, and a rule naming them beats those. An Owner of any workspace is never denied, so nobody can lock the instance away from its Owners. A rule on one area says nothing about another, so allowing `connectors:read` on GitHub leaves Cloudflare as their roles have it.

Only the instance Owner sees or changes these rules. In **Settings**, each instance entry ends with an **Access** card listing its areas: Instance, Team (accounts), Sign-in providers, DNS and Templates each show their own, Connectors has one row per connector on its **Access** tab, and the areas without an entry of their own (runners and machines, the topology registry, automation hosts, integrations, the audit log) sit under Instance. Each row says whose rules it carries, and opens the permissions panel on that area: the roles of every workspace, each named with its workspace ("Member in Northwind"), and every account, with **Check access** showing a person's answer for each permission and the rule or role it came from. A Deny on a read that a person or role holds today asks first, since a rule beats every role they hold and it may be their last way in.

To let sam manage DNS but not connectors: open **Settings → DNS**, then **DNS and gateways** in its Access card, add sam, set **Read** and **Create and update** to Allow, and save. Sam now sees DNS under Settings and nothing under Connectors, and **Check access** on sam reads "Allowed on DNS and gateways for sam" for both, and "Not allowed: role Member doesn't grant it" for Delete. With the API:

```
PUT /api/permissions
{"resource_type": "instance_area", "resource_ids": ["dns"], "user_ids": ["<id>"], "actions": ["dns:read", "dns:write"], "state": "allow"}
GET /api/permissions?resource_type=instance_area&resource_id=dns
```

## Who's in each project

**Access** in the sidebar's Workspace section, between Automations and Configuration, is where you choose who's in each project. It shows to anyone who can manage a project's access (`projects:access`); the Owner sees every project, anyone else only the projects they're in and manage.

The **Projects** tab is a grid: one row per project, one column per role, with how many people hold each role under its name. Tick a role on a project and everyone holding it sees the project and everything in it. The **Every project** row ticks a role on every project at once, or clears it from all of them when every project already has it; past ten projects a filter narrows the rows, and the top row then acts on the rows shown. **People**, the last column, names someone on a project whatever roles they hold: the add button opens a list of the workspace's people. A project that lists nobody says **Owners only**, since only Owners see it.

The grid is the whole workspace at a glance. For one project, its settings' [People in this project](#people-in-this-project) lists everyone in it, through which role or by name, and anyone kept out of it by name.

The **Public channels** tab is the Owner's. Every member sees a public channel, so every box starts ticked; untick a role to hide that channel from everyone holding it. Private channels keep their own members. A box here is the same rule as **See this channel** in the channel's own [permissions](/docs/guide/chat-and-voice/#who-sees-and-posts-in-a-channel), where whoever holds `channels:write` also sets who posts in it.

Each tab saves from its own footer, and the change reaches everyone at once: a project someone was just listed in shows up in their sidebar without a reload. Leaving the page with unsaved ticks asks first.

### After upgrading

The release that made projects opt-in wiped every per-person permission, so right after upgrading no project lists anyone and only Owners see projects. An Owner's first visit to the workspace then opens **Who's in each project**: the Projects grid and the Public channels grid on one page, and one **Finish setup** that saves both. Until an Owner finishes, the sidebar offers Owners **Set up access** to get back to it, and members see **Projects being set up**; any page that would show a project tells them their projects are on their way, and chat works as usual. **Finish setup** works with projects left empty: they stay Owners only until you list someone. Afterwards the page is the workspace's Access page, which shows who finished the setup and when.

## People in this project

A project they aren't in, with everything in it, is not found for someone: lists, links, search and agents alike. Inside a project they're in, their roles decide what they can do, and the project's rules can change that for a role or a person.

A project's **Settings → General** holds **People in this project**, for whoever has **Manage this project's access** there:

- **Roles** lists the roles it lets in, each with how many people hold it.
- **People** lists the people with a rule of their own here.
- **Kept out** lists the people a rule keeps out, whatever their roles.

**Add a role or person** lists one, which lets them in. Open a row to set what that role or person can do in the project. Its **×** takes it off the project, along with every rule it holds there, after asking; on someone kept out, it lets them back in. To keep someone out, open them and set **See this project** to **Deny**. A change reaches open browsers at once: a project opens in the sidebar of everyone a new role lets in, and drops out of the sidebar of someone kept out.

The project wizard's **Who's in it** step sets the same roles when you create a project. Deleting a project says how many people lose it before you confirm.

## Permissions on a project, a folder, a doc or a play

Right-click a project in the sidebar's switcher, a folder or a doc in the Docs list, or a play in **Configuration → Plays** and pick **Permissions…**; a doc's settings menu has it too, and each row of a project's **People in this project** opens it. It shows only to someone who can change that project's, folder's, doc's or play's rules. The list on the left holds the roles, then the people, with a rule here; **Add role or person** lists another. A folder or a doc refuses a role or person who isn't in its project, and says to add them to the project first.

Pick a role or a person to set its rule, one row per permission, each with three buttons:

- **Allow** (check): they have it here, whatever their roles say.
- **Fallback** (slash): this rule says nothing, so the answer comes from further out. The line under the row says what they get then, and from where; a project or role it names is a link to that project's panel or that role's rule.
- **Deny** (cross): they don't have it here, whatever their roles say.

A project carries **See this project** (being in it) and every project area; a folder and a doc carry the docs permissions and **Manage this folder's access** or **Manage this doc's access**; a play carries **See this play**, **Run**, **Edit** and **Delete**, and a play someone can't see is gone from their lists, run menus and auto plays. Between the rules for a person's roles, Allow wins, and a rule naming the person beats them.

A stack carries the stacks and deploys permissions; its rules reach its deploys and branch deployments, and a Deny of **Read** hides it, deploys included (see [Who can see and deploy a stack](/docs/guide/stacks-and-deploys/#who-can-see-and-deploy-a-stack)).

A folder's rules reach every doc in it. They sit between the project and the doc: a folder's rule beats the project's, and a doc's own rule beats its folder's, whichever role or person each names. Deny **Read** on a folder for a role and its docs drop out of that role's lists, search and links, unless a doc's own rule lets them back in. Moving a doc to another folder changes which folder's rules reach it at once, and deleting a folder deletes its rules as its docs move to Main. **Remove from this doc** (or project, or play) sets every row back to Fallback. Nothing changes until you press **Save**, and anyone else with the panel open sees the change without a refresh.

You can't allow a permission you don't hold on that project, folder, doc or play, or lift a Deny on one: its button shows a lock and says why. Adding a Deny is never limited. **Check access** shows what one person can do here, each answer with the rule or role that decided it.

Changing a project's rules takes **Manage this project's access** there, which its creator has; a folder's takes **Manage this folder's access** (`permissions:write`) on it; a doc's takes **Manage this doc's access** on it, which its creator has; a play's takes `plays:write`.

## Why someone can or can't

Every answer has a source: the Owner role, the person's roles, a rule on a project, a folder, a doc, a stack or a play for one of their roles or for them, or not being in the project. When Nexul refuses something you can see, the message names it, such as `docs:write required (Denied on doc Pricing for bob)` or `tickets:write required (Not allowed: role Client doesn't grant it)`. Something you can't see is not found and says nothing more, so it never confirms it exists.

To see every answer on one doc, folder, stack, play, project, or instance area at once, each with its source and the answer it would have without that entity's own rule:

```
GET /api/permissions/explain?resource_type=doc&resource_id=<id>&user_id=<id>
```

Leave out `user_id` for your own answers, which take only being able to open it. `role_id` in its place answers for a role, as someone holding only that role would get them; the panel's lines under a role's Fallback come from it. Someone else's, or a role's, take managing it: `permissions:write` on a doc or folder, `stacks:write` on a stack, `plays:write` on a play, `projects:access` on a project, or being the instance Owner for an instance area (`resource_type=instance_area&resource_id=dns`). Agents ask the same with `permission_overwrite_list` and a `user_id`.

## Names and pictures

Every member of a workspace sees its people's login, display name, and picture, and of their roles only who is an Owner. Set your own name and picture in **Settings → Profile**; a change reaches everyone's open screens without a refresh.

## Sign-in accounts and GitHub

**Settings → Profile** lists the accounts you sign in with under **Sign-in accounts**. **Link** adds another provider to the same profile, and **Unlink** removes one, never your last.

Under **GitHub**, the same page shows whether your own GitHub account lists your repositories: **Connected** with your GitHub login, **Reconnect needed** after GitHub refused to refresh it, or **Not connected**. Signing in with GitHub connects it. If you sign in with Google or Discord, **Connect GitHub** runs GitHub's sign-in and links it to this profile; a GitHub account already linked to someone else is refused. **Disconnect** forgets the token, and unlinking your GitHub sign-in does too. Nexul lists repositories only with your own GitHub, never with someone else's or the instance's; see [GitHub App](/docs/guide/github-app/).

## Over MCP

Agents manage people with `account_list` (each person's `roles` per workspace), `account_update`, `invitation_create`, `invitation_list`, `invitation_delete`, `role_update`, `role_delete`, and the `permission_overwrite_*` tools. `account_update` gives and takes away roles per workspace with `add_roles` and `remove_roles`, and `invitation_create` takes `roles` per workspace, at least one; neither sets anything per person. `permission_overwrite_update` sets a `state` of `allow`, `fallback`, or `deny` for people (`user_ids`) or roles (`role_ids`) on a doc, folder, stack, play, project, channel, or instance area (a folder's ids are `project_get`'s `doc_folders`, a stack's `stack_list`'s): allowing `projects:open` on a project lists a role or a person in it, denying it keeps a person out, denying `channels:read` on a channel hides it and denying `channels:post` keeps someone from posting there, with `channels:write` on the channel, and allowing `dns:write` on the `dns` instance area hands out DNS alone, which only the instance Owner may do. Template roles are `template_get` and `template_update` with `kind` `role` at `scope` `instance`: the shipped keys are `client`, `junior_developer`, `senior_developer`, `devops`, and `administrator`, a body is JSON with `name`, `permissions`, and `welcome_steps`, a body with no `key` adds one, `reset` returns a shipped one to its default, and `delete` removes one you added. `workspace_update` with `finish_access_setup` ends the one-time setup after an upgrade, and `workspace_list` says whether it's done. On a `computer`, its owner [shares it](/docs/guide/paired-computers/#share-a-computer) with people, and nobody else, an Owner included, can see or change its rules. See [MCP server](/docs/guide/mcp-server/).
