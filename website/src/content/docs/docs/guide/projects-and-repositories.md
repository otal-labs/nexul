---
title: Projects and repositories
description: Create a project with the wizard, attach the repository it deploys from, and add more services later.
sidebar:
  order: 9
---

A project holds a piece of your work: its board, docs, memories, repository, and the stacks it deploys. Split projects however suits you, by layer (frontend, backend) or by product area. A workspace starts with none.

## Creating a project

Press **New project** in the sidebar. A new workspace shows the same button on its empty board and docs list. The project wizard takes you from nothing to a deployed service:

1. **Info.** Name the project and give it a prefix: 2 to 5 letters or digits, starting with a letter. Ticket keys start with it, such as `BE-42`. A prefix is unique within its workspace.
2. **Repository.** Say where the project's tests live, then pick the repository to deploy from the repositories this workspace can see through your [GitHub App](/docs/guide/github-app/). The wizard scans it for a Dockerfile or compose file. Scans and repository links stay within that workspace's assigned accounts; naming another workspace's repository directly does not grant access. Once the App's private key is set, a workspace sees the accounts assigned to it; **Install it on another account or organisation** under the search installs the App somewhere new and assigns that account to this workspace.
3. **Service.** Check the service the scan proposes and pick the machine to run it on. See [Runners](/docs/guide/runners/).
4. **Environment.** Only if the repository has a `.env.example`: fill in its values.
5. **Reach.** Optional: give the service a hostname now, or later from the stack page.
6. **Deploy branches.** Optional: deploy other branches, such as `feature/*`, as their own copies. See [Stacks and deploys](/docs/guide/stacks-and-deploys/).
7. **Done.** Start the project's [interview](/docs/guide/interview/), check what's left for later, then press **Finish**.

Skipping the interview asks if you're sure. Agents still work, but without the project's rules, and the board shows a banner until the interview is done.

### Skipping steps and coming back

Click any step in the row at the top to open it, in any order. Each one shows whether it's done, skipped, or not visited yet. **Skip for now** marks a step skipped and moves on to the next one. A step that needs an earlier one, such as **Service** before a repository is picked, says so and takes you there.

The project exists from the moment you press **Continue** on the Info step, and stays in setup until you press **Finish** on the last step. **Finish** works with steps skipped; the Done step lists them, each with a way back. Until then, the project's part of the sidebar shows only **Continue setup**, on every device and for everyone who can change the project, and it reopens the wizard at the first step that's neither done nor skipped. If setup already created a service, resuming uses that same stack even when another service was added later. Detected Environment fields and saved stack values return on another device; **Continue** stays on Service when progress cannot be saved, so it can be retried without creating again. Everyone else sees **Being set up**. The board, docs, and settings still open from a link while setup is open.

After **Finish**, the sidebar lists the project's pages. To revisit a step later, open the project's **Settings → General** and press **Open the wizard**. The project stays set up.

A project made over [MCP](/docs/guide/mcp-server/) or the API is set up straight away, unless the call asks for setup with `setup_finished` set to `false`.

## Add another service or repository

From an existing project:

- **New service**, in the project's **Settings → Services**, reopens the wizard at the repository step to deploy something else.
- **Add repo**, in **Settings → Repositories**, attaches a repository without deploying it.
- **Import from this machine**, on the [Runners](/docs/guide/runners/) page next to a machine, adopts what's already running there as unmanaged stacks. Press **Attach repository** on one to give it a repository and make it deployable.

A repository belongs to one project. Attaching one that's already in another project is refused straight away.

## A tests repository

If the project's tests live in a repository of their own, choose **In a separate repository** at the repository step and pick it. It's attached for agents to read and run, and never deployed: Nexul refuses to build a stack from it. It shows in **Settings → Repositories** with a `tests` marker, and the interview starts from it when it asks about testing.

## Rename a workspace

A workspace has a name and a URL slug, as in `/acme/board`. Change both in **Configuration → General**, with `workspaces:write`.

A new name changes nothing else. A new slug moves every link: old links stop working and aren't redirected, and the page warns you before you save. A slug is lower-case letters and digits joined by dashes, up to 48 characters, and not already taken. Everyone with the workspace open moves to the new address without a reload.

## Who can see a project

Everyone in the workspace sees every project at their role's level, unless they're held to chosen projects. A project's **Settings → General** lists those people under **People with access**. See [People and access](/docs/guide/people-and-access/).
