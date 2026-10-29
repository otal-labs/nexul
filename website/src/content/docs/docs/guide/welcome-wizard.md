---
title: Welcome wizard
description: What a newcomer sets up on their first sign-in, how roles decide the steps, and where an admin sees how far each person got.
sidebar:
  order: 9
---

The Welcome wizard is where each person sets up what is theirs to set up: their GitHub link, their computer, their phone. The steps come from their roles, so a client sees a short list and a developer a longer one. It opens by itself the first time someone signs in, and after that it waits under **Welcome** in their account menu.

## What a newcomer sees

Say alice accepts an invitation to Acme as a Junior Developer. When she signs in for the first time, the Welcome wizard opens at `/wizard/welcome`. The step list sits on the left with how many are done, and the step itself on the right.

**Profile** always comes first: the name and picture her team sees. Her sign-in filled them in, so **Continue** with nothing changed is fine. After Profile come the steps ticked on any role she holds, in any workspace, always in this order:

- **Connect GitHub**: links her own GitHub account, so the project wizard lists the repositories she can open and tickets show her pull requests. Nexul reads with it and never pushes as her. Signing in with GitHub already did this; someone who signs in with Google or Discord presses **Connect GitHub** here.
- **Your computer**: the same flow as **Add a computer** in her settings. She runs one command on the computer where her agents should run, watches **Computer connected**, **T3 Code found** and **Paired** turn green, then sets it up. **Continue** waits until the setup is confirmed. See [Paired computers](/docs/guide/paired-computers/).
- **Your projects**: each project she's in, grouped by workspace, with her access level; clicking one opens its board. A project where her access differs by area lists the areas too. If nobody has added her to one yet, the step says so and asks her to tell whoever invited her; it ticks itself the moment they add her.
- **Add a deploy server**: name a [runner](/docs/guide/runners/), run its install command on the server, and wait for **Server connected**. Only a server she added herself counts.
- **Get the phone app**: a link to the app and a QR code to scan with it. **Phone connected** turns green once the phone signs in. This step stays done even if she later signs the phone out. See [Phone app](/docs/guide/phone-app/).

A step counts as done when the thing it sets up exists, not when someone clicked through it. A step already done when she reaches it shows what it found, such as "Connected as @alice", with only **Continue**. Every step in the list can be opened, in any order.

### Skip for now and Finish later

Every step but Profile has **Skip for now**. A skipped step shows as **Skipped** and can still be done any time: open it from the wizard or the account menu and do it.

**Finish later**, beside the logo above the step list, leaves the wizard from any step. Leaving it once, this way or from the end screen, counts as the first run, so it never opens by itself again.

The end screen reads "You're set up" when everything is done. Otherwise it reads "2 things left for later" and lists each step that isn't done, with **Open** beside it. Its button goes to the workspace, or reads **Create your first project** for someone who can create one in a workspace that has none.

## Welcome in the account menu

The account menu is the name and picture at the bottom of the sidebar. While someone has Welcome steps, its first row is **Welcome**, with how many are left, and under it one row per step that isn't done; skipped ones say **Skipped**. Each row opens the wizard on that step. When nothing is left to do, the row shows how many steps are done instead, such as "3/3", and stays, so the wizard is always one click away.

A red dot on the picture means a step is still to do. Only steps left to do light it: a skipped step doesn't, and neither does a done one. **Your projects** doesn't light it at first either, because nobody can add themselves to a project. The dot clears by itself as steps get done, in every open tab.

A skip lasts until the step is given to the person again. That happens when a role they hold gains the step, or when they are given a role that has it. The step then lights the dot again until it's done or skipped once more. So if bob skipped **Your computer** and you need him to add it after all, switching it on for his role brings the dot back for him, without a refresh. To give a step a role already has, switch it off, save, switch it back on and save again.

The phone app has no Welcome row and no dot. Every step but the phone's is done on the web.

## Choosing the steps a role gives

Steps belong to roles. In **Access → Roles**, open a role's **Edit**; the role editor's **Welcome** tab, after **In projects**, has one switch per step. Switching one on asks everyone holding the role, including people who skipped it. A step gives no permission, so anyone who can edit the role (`roles:write`) can switch any step on. Duplicating or cloning a role copies its steps.

The **Owner** role can't be edited. Its steps are fixed: **Connect GitHub** and **Your computer**.

New workspaces start with five [template roles](/docs/guide/people-and-access/#template-roles), each with its own steps:

| Role | Steps after Profile |
|---|---|
| Client | Your projects |
| Junior Developer | Your computer, Your projects |
| Senior Developer | Connect GitHub, Your computer |
| DevOps | Add a deploy server |
| Administrator | Connect GitHub |

Change what new workspaces get in **Settings → Templates → Roles**, in the same editor with the same **Welcome** tab. A workspace's copy is its own: editing a template later doesn't change the roles of workspaces that already exist.

Someone with roles in two workspaces gets one wizard with both sets of steps. Sam, a Senior Developer at Acme and a Client at Northwind, sees Connect GitHub, Your computer and Your projects in one list. A step done once counts everywhere.

## Seeing how far someone got

**Settings → Team** shows a line under each person who hasn't finished: "2 of 3 set up", then the first thing left, such as "computer not added", and the steps they skipped. Its dot is amber while a step is left and grey once only skips remain. The line goes away once every step is done, and it updates by itself. It names "not in a project yet" whenever the person is in no project, even if their roles don't give that step, because adding them to one is yours to do.

Open the person for the detail: when they joined and who invited them, then each step left or skipped with the reason it isn't done ("No paired computer with its setup confirmed") or when they skipped it.

The **Invitations** tab lists under **Accepted in the last 30 days** who each link let in, with the same line. See [People and access](/docs/guide/people-and-access/#setup-progress).

## Over MCP

`account_get` and `account_list` return each person's `welcome` progress, so an agent asked why bob can't run plays sees that his computer isn't added. `role_update` takes `welcome_steps`, a list of `github`, `computer`, `projects`, `deploy_server` and `phone`, and `workspace_list` returns each role's list. Template roles take the same list through `template_update`. Skipping is a person's own choice and has no tool.
