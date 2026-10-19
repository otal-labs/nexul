---
title: Projects and repositories
description: Create a project with the wizard, attach the repository it deploys from, and add more services later.
sidebar:
  order: 9
---

A project holds a piece of your work: its board, docs, memories, repository, and the stacks it deploys. Split projects however suits you, by layer (frontend, backend) or by product area. A workspace starts with none.

## Creating a project

Press **New project** in the sidebar. A new workspace shows the same button on its empty board and docs list. The project wizard takes you from nothing to a deployed service, or stops at the repository step for a project that has no code to deploy yet:

1. **Info.** Name the project and give it a prefix: 2 to 5 letters or digits, starting with a letter. Ticket keys start with it, such as `BE-42`. A prefix is unique within its workspace.
2. **Who's in it.** Tick the roles that can open the project. Your own roles start ticked, and you're always in. Add people by name or keep someone out later, from the project's settings.
3. **Repository.** Say where the project's tests live, then pick the repository the project's code lives in. Nexul scans it and shows what it found, then you choose: **Deploy it** carries on to Service, and **Attach without deploying** attaches the repository and goes straight to Done. A project with no code yet, such as one for a client's docs, takes **No repository yet** under the search instead, which also goes to Done. Either way the deploy steps are marked skipped. The search lists the repositories your own GitHub account can open wherever the [GitHub App](/docs/guide/github-app/) is installed, and the wizard scans the one you pick with your GitHub too. If your GitHub isn't connected, the step shows **Connect GitHub to see your repositories** instead; **Connect GitHub** links it to your profile and brings you to **Settings → Profile**, and you come back to the wizard from there. **Install it on an account or organisation you manage** under the search installs the App somewhere new.
4. **Service.** Check the service the scan proposes and pick the machine to run it on. See [Runners](/docs/guide/runners/).
5. **Environment.** Only if the repository has a `.env.example`: fill in its values.
6. **Reach.** Optional: give the service a hostname now, or later from the stack page.
7. **Deploy branches.** Optional: deploy other branches, such as `feature/*`, as their own copies. See [Stacks and deploys](/docs/guide/stacks-and-deploys/).
8. **Done.** Start the project's [interview](/docs/guide/interview/), check what's left for later, then press **Finish**. A project set up without a service lists deploying as one row, **Open Repository** or **Deploy it**, and offers **Docs**, **Board** and **Invite people** under **Where to next**; each finishes setup on the way there.

Skipping the interview asks if you're sure. Agents still work, but without the project's rules, and the board shows a banner until the interview is done.

### Skipping steps and coming back

Click any step in the row at the top to open it, in any order. Each one shows whether it's done, skipped, or not visited yet. **Skip for now** marks a step skipped and moves on to the next one; on the Repository step, **No repository yet** takes its place. A step that needs an earlier one, such as **Service** before a repository is picked, says so and takes you there.

The project exists from the moment you press **Continue** on the Info step, and stays in setup until you press **Finish** on the last step. **Finish** works with steps skipped; the Done step lists them, each with a way back. Until then, the project's part of the sidebar shows only **Continue setup**, on every device and for everyone who can change the project, and it reopens the wizard at the first step that's neither done nor skipped. If setup already created a service, resuming uses that same stack even when another service was added later. Detected Environment fields and saved stack values return on another device; **Continue** stays on Service when progress cannot be saved, so it can be retried without creating again. Everyone else sees **Being set up**. The board, docs, and settings still open from a link while setup is open. Adding a service from the project's page makes a new one beside the service setup recorded and never changes the setup. If Repository is done but this browser has no scan of it (another device, or after a reload), Service asks you to scan it again and Repository stays done. Only people who can read the project's stacks see which service setup recorded.

After **Finish**, the sidebar lists the project's pages. Right-clicking the project in the sidebar or its switcher offers **Project settings** and, to whoever manages who's in it, **Permissions…**. To revisit a step later, open the project's **Settings → General** and press **Open the wizard**. The project stays set up. The repository step opens on your saved answer about tests and the repository already attached; **Keep it** keeps that repository, so you can add a tests repository without picking and scanning it again.

A project made over [MCP](/docs/guide/mcp-server/) or the API is set up straight away, unless the call asks for setup with `setup_finished` set to `false`.

## Add another service or repository

From an existing project:

- **New service**, in the project's **Settings → Services**, reopens the wizard at the repository step to deploy something else. If exactly one attached repository has no service yet, it skips the pick and opens **Service** for that repository.
- **Deploy this repository** sits in **Settings → Services** beside each attached repository no service builds from, marked **Attached, not deployed yet**. It opens **Service** for that repository, and the row goes once a service deploys it. Only people who can read the project's stacks and add services see it.
- **Add repo**, in **Settings → Repositories**, attaches a repository without deploying it.
- **Import from this machine**, on the [Runners](/docs/guide/runners/) page next to a machine, adopts what's already running there as unmanaged stacks. Press **Attach repository** on one to give it a repository and make it deployable.

A repository belongs to one project. Attaching one that's already in another project is refused straight away. You can attach only a repository your own GitHub account lists; attaching it is what lets the workspace's deploys, webhooks and pull request reads use its account through the App, and the account then shows as **Used by** this workspace under **Settings → Connectors → GitHub App**.

## A tests repository

If the project's tests live in a repository of their own, choose **In a separate repository** at the repository step and pick it. It's attached for agents to read and run, and never deployed: Nexul refuses to build a stack from it. It shows in **Settings → Repositories** with a `tests` marker, and the interview starts from it when it asks about testing. It has to be a different repository from the one the project deploys, so the tests search leaves that one out.

## Rename a workspace

A workspace has a name and a URL slug, as in `/acme/board`. Change both in **Configuration → General**, with `workspaces:write`.

A new name changes nothing else. A new slug moves every link: old links stop working and aren't redirected, and the page warns you before you save. A slug is lower-case letters and digits joined by dashes, up to 48 characters, and not already taken. Everyone with the workspace open moves to the new address without a reload.

## Archive a workspace

An Owner can archive a workspace in **Configuration → Danger zone**, with `workspaces:delete`, which Owner holds and the Administrator template role does not. The confirmation says how many people lose the workspace and how many projects go with it. Nothing is deleted: the workspace leaves every list, search, inbox and live update for everyone in it, its address stops resolving, and anyone with it open moves to the next workspace they can open.

The instance Owner brings it back from **Settings → Instance → Archived workspaces** with **Restore**, which takes `instance:write`. Its projects, tickets, docs, people and roles return exactly as they were. There is no permanent delete yet. The instance's own workspace, the one whose Owner is the instance Owner, can't be archived.

Nothing automatic runs in an archived workspace. Its automations stop receiving events and lose their connections and secrets, its auto plays queue nothing and start nothing, and a push to one of its repositories deploys nothing. Restoring resumes all of it from then on: what happened while the workspace was archived is skipped, not replayed, and runs that were already waiting in a queue carry on.

Agents archive with `workspace_update` and `archived` set to `true`, restore with `archived` set to `false`, and list what is archived with `workspace_list` and `archived` set to `true`; a member's `workspace_list` never shows an archived workspace.

## Who can see a project

Projects are opt-in. Someone sees a project only once it lists them, or a role they hold; everything in a project they aren't in, its name included, is hidden from them. Inside a project they're in, their roles decide what they can do there. Workspace Owners see every project.

Creating a project takes **Create projects**. The new project lists you and each of your roles, so it opens for you and for whoever holds those roles, and you can manage who else is in it: the wizard's **Who's in it** step, then **People in this project** in the project's **Settings → General**. See [People in this project](/docs/guide/people-and-access/#people-in-this-project).

Projects made before the **Who's in it** step existed count it as done.
