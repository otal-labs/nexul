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
6. Press **Create link**, then **Copy link**. It's shown once.
7. Send the link however you like. The person opens it, signs in with one of the instance's sign-in providers, reviews what they're being given, and accepts.

A link works once. The Owner role is never offered; ownership doesn't move through an invitation. **Active invitation links** lists the links not used yet; **Revoke invitation** kills one.

## Manage people

**Settings → Team** lists everyone on the instance: a dot on each picture for who is online, when the others were last seen, their role or how many workspaces they are in, and **Disabled** or **Removed** for an account that can't sign in. Open a person to see one tab per workspace they're in. Each tab holds their **Role**, **Edit overrides**, their projects, and **Remove from workspace**; the **+** after the last tab adds them to another workspace.

Nothing applies as you edit. A tab with changes shows a dot, **Confirm** applies them all, and **Cancel** asks before throwing them away. If one change is refused, the dialog says which and keeps the rest waiting for the next **Confirm**.

Account actions apply at once, each after its own confirmation: **Disable** stops someone signing in and keeps their access, **Reactivate** undoes it, and **Remove account** removes them. **Restore** brings a removed account back with no workspace access.

Seeing everyone takes `accounts:read`. Changing someone in a workspace takes `members:write` there. Without `accounts:read`, someone who manages members finds Team under **Configuration** instead, showing only the workspaces they manage.

## Roles

Every workspace has an **Owner** role. It can't be renamed, deleted, or limited: the Owner holds every permission and bypasses overrides. Every other role is yours to define in **Configuration → Roles**.

A role sets one level per area: **None**, **Read**, **Write**, or **Delete**, each level including the ones before it. Some areas carry extra switches, such as docs' **Thread**, **Clone**, and **Lock**. The role editor has two parts:

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

## Names and pictures

Every member of a workspace sees its people's login, display name, and picture, and nothing about their roles. Set your own name and picture in **Settings → Profile**; a change reaches everyone's open screens without a refresh.

## Over MCP

Agents manage people with `account_list`, `account_update`, `invitation_create`, `invitation_list`, `invitation_delete`, `role_update`, `role_delete`, and the `permission_overwrite_*` tools. `account_update` and `invitation_create` take `every_project` (`role` or `none`) and `project_access` per workspace. See [MCP server](/docs/guide/mcp-server/).
